package grades

import (
	"context"
	"fmt"
	"io"
	"math"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type Service struct {
	pool *pgxpool.Pool
	repo *Repository
	q    *db.Queries
}

func NewService(pool *pgxpool.Pool, repo *Repository) *Service {
	return &Service{pool: pool, repo: repo, q: db.New(pool)}
}

func (s *Service) CreateScheme(ctx context.Context, courseID, lecturerID int64, components []ComponentInput) (SchemeResponse, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, lecturerID, db.UserRoleLecturer); err != nil {
		return SchemeResponse{}, ErrForbidden
	}

	_, err := s.repo.GetSchemeByCourse(ctx, courseID)
	if err == nil {
		return SchemeResponse{}, ErrSchemeExists
	}

	if err := s.validateWeights(components); err != nil {
		return SchemeResponse{}, fmt.Errorf("%w: %v", ErrValidation, err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return SchemeResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	scheme, err := qtx.CreateGradeScheme(ctx, db.CreateGradeSchemeParams{
		CourseID:  courseID,
		CreatedBy: lecturerID,
	})
	if err != nil {
		return SchemeResponse{}, err
	}

	var savedComponents []db.GradeComponent
	// Map from input index to DB ID for parents
	idMap := make(map[int]int64)

	// First pass: top-level components (ParentIndex == nil)
	for i, c := range components {
		if c.ParentIndex == nil {
			var st, ak pgtype.Text
			if c.SourceType != nil {
				st = pgtype.Text{String: *c.SourceType, Valid: true}
			}
			if c.AutoKind != nil {
				ak = pgtype.Text{String: *c.AutoKind, Valid: true}
			}
			var weight pgtype.Numeric
			_ = weight.Scan(fmt.Sprintf("%f", c.Weight))

			comp, err := qtx.InsertGradeComponent(ctx, db.InsertGradeComponentParams{
				SchemeID:   scheme.ID,
				ParentID:   pgtype.Int8{},
				Name:       c.Name,
				Weight:     weight,
				SourceType: st,
				AutoKind:   ak,
			})
			if err != nil {
				return SchemeResponse{}, err
			}
			idMap[i] = comp.ID
			savedComponents = append(savedComponents, comp)
		}
	}

	// Second pass: sub-components
	for _, c := range components {
		if c.ParentIndex != nil {
			parentID, ok := idMap[*c.ParentIndex]
			if !ok {
				return SchemeResponse{}, fmt.Errorf("%w: invalid parent index", ErrValidation)
			}
			var st, ak pgtype.Text
			if c.SourceType != nil {
				st = pgtype.Text{String: *c.SourceType, Valid: true}
			}
			if c.AutoKind != nil {
				ak = pgtype.Text{String: *c.AutoKind, Valid: true}
			}
			var weight pgtype.Numeric
			_ = weight.Scan(fmt.Sprintf("%f", c.Weight))

			comp, err := qtx.InsertGradeComponent(ctx, db.InsertGradeComponentParams{
				SchemeID:   scheme.ID,
				ParentID:   pgtype.Int8{Int64: parentID, Valid: true},
				Name:       c.Name,
				Weight:     weight,
				SourceType: st,
				AutoKind:   ak,
			})
			if err != nil {
				return SchemeResponse{}, err
			}
			savedComponents = append(savedComponents, comp)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SchemeResponse{}, err
	}

	return s.mapSchemeResponse(scheme, savedComponents), nil
}

func (s *Service) validateWeights(components []ComponentInput) error {
	sums := map[int]float64{} // keyed by parent index, -1 for root
	for _, c := range components {
		k := -1
		if c.ParentIndex != nil {
			k = *c.ParentIndex
		}
		sums[k] += c.Weight
	}
	for parent, total := range sums {
		if math.Abs(total-100) > 0.001 {
			if parent == -1 {
				return fmt.Errorf("top-level weights sum to %.2f, must be 100", total)
			}
			return fmt.Errorf("weights under parent index %d sum to %.2f, must be 100", parent, total)
		}
	}
	return nil
}

func (s *Service) DeleteScheme(ctx context.Context, courseID, lecturerID int64) error {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, lecturerID, db.UserRoleLecturer); err != nil {
		return ErrForbidden
	}

	scheme, err := s.repo.GetSchemeByCourse(ctx, courseID)
	if err != nil {
		return ErrNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	scores, err := qtx.CountSchemeScores(ctx, scheme.ID)
	if err != nil {
		return err
	}
	pubs, err := qtx.CountSchemePublications(ctx, scheme.ID)
	if err != nil {
		return err
	}
	if scores > 0 || pubs > 0 {
		return ErrSchemeImmutable
	}

	err = qtx.DeleteSchemeComponents(ctx, scheme.ID)
	if err != nil {
		return err
	}

	deleted, err := qtx.DeleteSchemeIfEmpty(ctx, db.DeleteSchemeIfEmptyParams{
		ID:       scheme.ID,
		CourseID: courseID,
	})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrSchemeImmutable // Should not happen since we already checked
	}

	return tx.Commit(ctx)
}

func (s *Service) EnterScore(ctx context.Context, courseID, componentID, studentID int64, score float64, lecturerID int64) error {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, lecturerID, db.UserRoleLecturer); err != nil {
		return ErrForbidden
	}

	if score < 0 || score > 100 {
		return fmt.Errorf("%w: score must be between 0 and 100", ErrValidation)
	}

	// Ensure component exists and belongs to the course
	comps, err := s.repo.ListSchemeComponents(ctx, courseID)
	if err != nil {
		return err
	}
	var target *db.GradeComponent
	for _, c := range comps {
		if c.ID == componentID {
			target = &c
			break
		}
	}
	if target == nil {
		return ErrNotFound
	}
	if !target.SourceType.Valid || target.SourceType.String != "MANUAL" {
		return fmt.Errorf("%w: component is not MANUAL", ErrValidation)
	}

	var num pgtype.Numeric
	_ = num.Scan(fmt.Sprintf("%f", score))
	
	return s.repo.UpsertGradeScore(ctx, db.UpsertGradeScoreParams{
		ComponentID: componentID,
		StudentID:   studentID,
		Score:       num,
	})
}

func (s *Service) ComputeOverallForStudent(ctx context.Context, courseID, studentID int64) (OverallResponse, error) {
	comps, err := s.repo.ListSchemeComponents(ctx, courseID)
	if err != nil {
		return OverallResponse{}, err
	}
	if len(comps) == 0 {
		return OverallResponse{}, ErrNotFound
	}

	// Fetch manual scores
	scores, err := s.repo.ListScoresForStudent(ctx, db.ListScoresForStudentParams{
		CourseID:  courseID,
		StudentID: studentID,
	})
	if err != nil {
		return OverallResponse{}, err
	}
	scoreMap := make(map[int64]float64)
	for _, sc := range scores {
		f, _ := sc.Score.Float64Value()
		scoreMap[sc.ComponentID] = f.Float64
	}

	var computed []ComputedComponent
	overall := 0.0

	// Helper to compute a leaf's value
	computeLeaf := func(c db.GradeComponent) float64 {
		if !c.SourceType.Valid {
			return 0
		}
		if c.SourceType.String == "MANUAL" {
			if val, ok := scoreMap[c.ID]; ok {
				return val
			}
			return 0 // missing = 0
		}
		if c.SourceType.String == "AUTO" {
			if c.AutoKind.String == "QUIZ_AVERAGE" {
				res, err := s.repo.ComputeQuizAverage(ctx, db.ComputeQuizAverageParams{
					CourseID:  courseID,
					StudentID: studentID,
				})
				if err == nil {
					f, _ := res.QuizAverage.Float64Value()
					return f.Float64
				}
			} else if c.AutoKind.String == "ASSIGNMENT_AVERAGE" {
				res, err := s.repo.ComputeAssignmentAverage(ctx, db.ComputeAssignmentAverageParams{
					CourseID:  courseID,
					StudentID: studentID,
				})
				if err == nil {
					f, _ := res.AssignmentAverage.Float64Value()
					return f.Float64
				}
			}
		}
		return 0
	}

	// Compute values
	for _, c := range comps {
		if !c.ParentID.Valid { // Top-level
			val := 0.0
			if c.SourceType.Valid {
				// It's a leaf
				val = computeLeaf(c)
			} else {
				// It's a composite, compute children
				childSum := 0.0
				for _, child := range comps {
					if child.ParentID.Valid && child.ParentID.Int64 == c.ID {
						w, _ := child.Weight.Float64Value()
						cv := computeLeaf(child)
						childSum += cv * (w.Float64 / 100.0)
					}
				}
				val = childSum
			}
			
			// Round to 2 decimals
			val = math.Round(val*100) / 100
			computed = append(computed, ComputedComponent{
				ComponentID: c.ID,
				Score:       val,
			})
			w, _ := c.Weight.Float64Value()
			overall += val * (w.Float64 / 100.0)
		}
	}
	overall = math.Round(overall*100) / 100

	return OverallResponse{
		StudentID:  studentID,
		Overall:    overall,
		Components: computed,
	}, nil
}

func (s *Service) GetScheme(ctx context.Context, courseID, userID int64, role string) (SchemeResponse, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, userID, db.UserRole(role)); err != nil {
		return SchemeResponse{}, ErrForbidden
	}

	scheme, err := s.repo.GetSchemeByCourse(ctx, courseID)
	if err != nil {
		return SchemeResponse{}, ErrNotFound
	}
	comps, err := s.repo.ListSchemeComponents(ctx, courseID)
	if err != nil {
		return SchemeResponse{}, err
	}
	return s.mapSchemeResponse(scheme, comps), nil
}

func (s *Service) mapSchemeResponse(scheme db.GradeScheme, components []db.GradeComponent) SchemeResponse {
	var resp []ComponentResponse
	for _, c := range components {
		var pid *int64
		if c.ParentID.Valid {
			pid = &c.ParentID.Int64
		}
		var st, ak *string
		if c.SourceType.Valid {
			v := c.SourceType.String
			st = &v
		}
		if c.AutoKind.Valid {
			v := c.AutoKind.String
			ak = &v
		}
		w, _ := c.Weight.Float64Value()
		resp = append(resp, ComponentResponse{
			ID:         c.ID,
			ParentID:   pid,
			Name:       c.Name,
			Weight:     w.Float64,
			SourceType: st,
			AutoKind:   ak,
		})
	}
	return SchemeResponse{
		ID:         scheme.ID,
		CourseID:   scheme.CourseID,
		Components: resp,
	}
}

func (s *Service) ImportScoresCSV(ctx context.Context, courseID, componentID, lecturerID int64, r io.Reader) ([]RowError, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, lecturerID, db.UserRoleLecturer); err != nil {
		return nil, ErrForbidden
	}

	comps, err := s.repo.ListSchemeComponents(ctx, courseID)
	if err != nil {
		return nil, err
	}
	var target *db.GradeComponent
	for _, c := range comps {
		if c.ID == componentID {
			target = &c
			break
		}
	}
	if target == nil {
		return nil, ErrNotFound
	}
	if !target.SourceType.Valid || target.SourceType.String != "MANUAL" {
		return nil, fmt.Errorf("%w: component is not MANUAL", ErrValidation)
	}

	parsed, rowErrs := ParseScoreCSV(r)
	if len(rowErrs) > 0 {
		return rowErrs, ErrValidation
	}
	if len(parsed) == 0 {
		return nil, fmt.Errorf("%w: file has no valid data", ErrValidation)
	}

	usernames := make([]string, len(parsed))
	for i, p := range parsed {
		usernames[i] = p.Username
	}

	validUsers, err := s.q.GetUserIDsByRole(ctx, db.GetUserIDsByRoleParams{
		Column1: usernames,
		Role:    db.UserRoleStudent,
	})
	if err != nil {
		return nil, err
	}

	validMap := make(map[string]int64)
	for _, u := range validUsers {
		validMap[u.Username] = u.ID
	}

	for _, p := range parsed {
		if _, ok := validMap[p.Username]; !ok {
			rowErrs = append(rowErrs, RowError{
				Row:     p.RowIndex,
				Field:   "student_id",
				Message: "invalid, not found, or not enrolled student",
			})
		}
	}

	if len(rowErrs) > 0 {
		return rowErrs, ErrValidation
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)

	for _, p := range parsed {
		uid := validMap[p.Username]
		var num pgtype.Numeric
		_ = num.Scan(fmt.Sprintf("%f", p.Score))
		
		err := qtx.UpsertGradeScore(ctx, db.UpsertGradeScoreParams{
			ComponentID: componentID,
			StudentID:   uid,
			Score:       num,
		})
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Service) PublishComponent(ctx context.Context, courseID, componentID, lecturerID int64) error {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, lecturerID, db.UserRoleLecturer); err != nil {
		return ErrForbidden
	}

	comps, err := s.repo.ListSchemeComponents(ctx, courseID)
	if err != nil {
		return err
	}
	var target *db.GradeComponent
	for _, c := range comps {
		if c.ID == componentID {
			target = &c
			break
		}
	}
	if target == nil {
		return ErrNotFound
	}
	if target.ParentID.Valid {
		return fmt.Errorf("%w: component is not top-level", ErrValidation)
	}

	students, err := s.q.ListCourseStudents(ctx, courseID)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	for _, st := range students {
		// compute live value
		res, err := s.ComputeOverallForStudent(ctx, courseID, st.StudentID)
		if err != nil {
			return err
		}
		var val float64
		for _, c := range res.Components {
			if c.ComponentID == componentID {
				val = c.Score
				break
			}
		}

		var num pgtype.Numeric
		_ = num.Scan(fmt.Sprintf("%f", val))

		err = qtx.UpsertGradePublication(ctx, db.UpsertGradePublicationParams{
			ComponentID: componentID,
			StudentID:   st.StudentID,
			Value:       num,
		})
		if err != nil {
			return err
		}

		_, err = qtx.InsertNotification(ctx, db.InsertNotificationParams{
			RecipientID:  st.StudentID,
			Type:         "GRADE_PUBLISHED",
			Title:        "Grades available",
			Body:         fmt.Sprintf("Your %s grade is available.", target.Name),
			ResourceType: pgtype.Text{String: "course", Valid: true},
			ResourceID:   pgtype.Int8{Int64: courseID, Valid: true},
			Link:         pgtype.Text{String: fmt.Sprintf("/courses/%d/grades", courseID), Valid: true},
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Service) GetStudentGrades(ctx context.Context, courseID, studentID int64) (StudentGradesResponse, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, studentID, db.UserRoleStudent); err != nil {
		return StudentGradesResponse{}, ErrForbidden
	}

	totalTop, err := s.q.CountTopLevelComponents(ctx, courseID)
	if err != nil {
		return StudentGradesResponse{}, err
	}

	pubs, err := s.q.ListPublicationsForStudent(ctx, db.ListPublicationsForStudentParams{
		CourseID:  courseID,
		StudentID: studentID,
	})
	if err != nil {
		return StudentGradesResponse{}, err
	}

	comps, err := s.repo.ListSchemeComponents(ctx, courseID)
	if err != nil {
		return StudentGradesResponse{}, err
	}
	
	weightMap := make(map[int64]float64)
	for _, c := range comps {
		if !c.ParentID.Valid {
			w, _ := c.Weight.Float64Value()
			weightMap[c.ID] = w.Float64
		}
	}

	var componentsResp []ComputedComponent
	overall := 0.0

	for _, p := range pubs {
		v, _ := p.Value.Float64Value()
		componentsResp = append(componentsResp, ComputedComponent{
			ComponentID: p.ComponentID,
			Score:       v.Float64,
		})
		if w, ok := weightMap[p.ComponentID]; ok {
			overall += v.Float64 * (w / 100.0)
		}
	}

	overall = math.Round(overall*100) / 100

	var ovr *float64
	if totalTop > 0 && int64(len(pubs)) == totalTop {
		ovr = &overall
	}

	return StudentGradesResponse{
		StudentID:  studentID,
		Overall:    ovr,
		Components: componentsResp,
	}, nil
}

package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
	"github.com/panadolextra91/myiu-lite/backend/internal/auditlogs"
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

func (s *Service) CreateAccount(ctx context.Context, role db.UserRole, username, fullName, rawDOB string, actorID int64) (int64, error) {
	parsedDOB, hash, err := deriveDefaults(rawDOB)
	if err != nil {
		return 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)

	id, err := qtx.CreateUser(ctx, db.CreateUserParams{
		Username:           username,
		PasswordHash:       hash,
		Role:               role,
		FullName:           pgtype.Text{String: fullName, Valid: true},
		DateOfBirth:        pgtype.Date{Time: parsedDOB, Valid: true},
		MustChangePassword: true,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return 0, ErrDuplicateUser
		}
		return 0, err
	}

	meta, _ := json.Marshal(map[string]string{"username": username, "role": string(role)})
	if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.ACCOUNT_CREATE, auditlogs.TargetTypeUser, &id, nil, meta); err != nil {
		return 0, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *Service) ImportAccounts(ctx context.Context, role db.UserRole, file io.Reader, actorID int64) (int, []RowError, error) {
	var parsed []ParsedAccount
	var rowErrs []RowError

	if role == db.UserRoleStudent {
		parsed, rowErrs = parseStudentCSV(file)
	} else if role == db.UserRoleLecturer {
		parsed, rowErrs = parseLecturerCSV(file)
	}

	if len(rowErrs) > 0 {
		return 0, rowErrs, nil
	}
	if len(parsed) == 0 {
		return 0, nil, nil
	}

	// Two-phase validation
	var ids []string
	seen := make(map[string]int)
	for _, p := range parsed {
		if p.ID == "" {
			rowErrs = append(rowErrs, RowError{Row: p.RowIndex, Field: "id", Message: "cannot be empty"})
		}
		if p.FullName == "" {
			rowErrs = append(rowErrs, RowError{Row: p.RowIndex, Field: "full_name", Message: "cannot be empty"})
		}
		_, _, err := deriveDefaults(p.DOB)
		if err != nil {
			rowErrs = append(rowErrs, RowError{Row: p.RowIndex, Field: "dob", Message: "invalid format, expected DD/MM/YYYY"})
		}

		if prev, ok := seen[p.ID]; ok {
			rowErrs = append(rowErrs, RowError{Row: p.RowIndex, Field: "id", Message: fmt.Sprintf("duplicate ID in file (matches row %d)", prev)})
		} else {
			seen[p.ID] = p.RowIndex
			ids = append(ids, p.ID)
		}
	}

	if len(rowErrs) > 0 {
		return 0, rowErrs, nil
	}

	// Check duplicates against DB
	existing, err := s.repo.GetActiveUsernames(ctx, ids)
	if err != nil {
		return 0, nil, err
	}
	if len(existing) > 0 {
		existMap := make(map[string]bool)
		for _, e := range existing {
			existMap[e] = true
		}
		for _, p := range parsed {
			if existMap[p.ID] {
				rowErrs = append(rowErrs, RowError{Row: p.RowIndex, Field: "id", Message: "user already exists in system"})
			}
		}
	}

	if len(rowErrs) > 0 {
		return 0, rowErrs, nil
	}

	// Transactional insert
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.repo.WithTx(tx)
	count := 0

	for _, p := range parsed {
		dob, hash, _ := deriveDefaults(p.DOB)
		_, err := qtx.CreateUser(ctx, db.CreateUserParams{
			Username:           p.ID,
			PasswordHash:       hash,
			Role:               role,
			FullName:           pgtype.Text{String: p.FullName, Valid: true},
			DateOfBirth:        pgtype.Date{Time: dob, Valid: true},
			MustChangePassword: true,
		})
		if err != nil {
			return 0, nil, err
		}
		count++
	}

	action := auditlogs.IMPORT_STUDENTS
	if role == db.UserRoleLecturer {
		action = auditlogs.IMPORT_LECTURERS
	}
	affectedCount := int64(count)
	err = auditlogs.WriteAudit(ctx, qtx, actorID, action, auditlogs.TargetTypeUser, nil, &affectedCount, nil)
	if err != nil {
		return 0, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, err
	}

	return count, nil, nil
}

func (s *Service) ResetPassword(ctx context.Context, userID, actorID int64) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ErrUserNotFound
	}

	if user.IsSystem {
		return ErrUserNotFound
	}

	if !user.DateOfBirth.Valid {
		return errors.New("user has no date of birth")
	}

	pwStr := user.DateOfBirth.Time.Format(dobPasswordLayout)
	hash, err := bcrypt.GenerateFromPassword([]byte(pwStr), 12)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)

	err = qtx.ResetUserPassword(ctx, db.ResetUserPasswordParams{
		ID:           userID,
		PasswordHash: string(hash),
	})
	if err != nil {
		return err
	}

	if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.PASSWORD_RESET, auditlogs.TargetTypeUser, &userID, nil, nil); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) ListUsers(ctx context.Context, role *db.UserRole, search *string, limit, offset int32) ([]db.User, int64, error) {
	arg := db.ListUsersParams{Limit: limit, Offset: offset}
	cArg := db.CountUsersParams{}
	if role != nil {
		arg.Role = db.NullUserRole{UserRole: *role, Valid: true}
		cArg.Role = db.NullUserRole{UserRole: *role, Valid: true}
	}
	if search != nil && *search != "" {
		arg.Search = pgtype.Text{String: *search, Valid: true}
		cArg.Search = pgtype.Text{String: *search, Valid: true}
	}

	users, err := s.repo.ListUsers(ctx, arg)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountUsers(ctx, cArg)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

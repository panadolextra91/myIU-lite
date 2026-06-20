# Phase 5 Wave 2 Summary: Grade Scheme & Computation Engine

## What was Accomplished

### 1. Grade Scheme Structure & Logic (`internal/grades/service.go`)
- Developed the `SchemeService` to handle CRUD operations for `Scheme`, `Component`, and `Score`.
- Implemented `validateWeights` to guarantee components belonging to a parent component recursively sum up to 100%.
- Prevented scheme deletion if there are scores recorded or if grades are published.

### 2. Manual CSV Grade Import (`internal/grades/csv.go`)
- Added robust CSV processing ensuring an "all-or-nothing" database transaction pattern.
- Validated column structure, matching IDs, score constraints, and enrollment status.
- Aggregated all validation errors into structured 422 JSON arrays.

### 3. Computation Engine (`internal/grades/service.go`)
- Developed `ComputeOverallForStudent` for live grade compilation across the hierarchy.
- **Rules applied:**
  - `MANUAL`: Reads score directly.
  - `AUTO: QUIZ_AVERAGE`: Computed dynamically across all quiz attempts.
  - `AUTO: ASSIGNMENT_AVERAGE`: Computed dynamically across finalized assignments.
  - Missed submissions automatically default to `0`.

### 4. Endpoints & Integrations
- Bound the Grades domain cleanly in `cmd/api/main.go` and `handler.go`.
- Designed Lecturer UI (`Gradebook.tsx`) with nested hook-form arrays.
- Refactored `grades-api.ts` into standard library format.
- Covered with integration test suits (`compute_test.go`, `csv_test.go`).
- Database constraints perfectly match the API requirements.
- Full UI for CSV imports and live rendering of schemes and overall student averages.

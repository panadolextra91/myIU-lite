# Phase 4 - Wave 4 Summary

## Completed Work

1. **Attempt State Machine & SQL Layer**
   - Created `quiz_attempts.sql` providing all necessary database operations: StartAttempt, GetAttemptByID, CountAttempts, UpdateAttemptAnswer, and idempotent marking (`MarkAttemptSubmitted`, `MarkAttemptAutoSubmitted`).
   - Implemented `quizzes/attempt.go` encapsulating a highly secure and robust state machine that strictly governs quiz taking.
   - Enforced retake bounds via exact DB `CountAttempts` validation, guaranteeing students cannot exceed their retake configuration.
   - Employed `math/rand/v2` perfectly seeded with the specific `attempt_id` to immutably draw and shuffle an exact subset of questions (M-of-N).

2. **Secure DTO Boundary & Idempotency**
   - Expanded `dto.go` with `StudentQuizAttemptView` that enforces Pattern 3: it exclusively serves options without `IsCorrect` properties. 
   - Conditional answer key revelation: `CorrectOptions` map is strictly returned ONLY if the quiz window is verifiably closed (`now > close_at`) and the attempt is terminal.
   - Guaranteed idempotent grading via SQL-level `execrows` bounds (`UPDATE ... WHERE status='IN_PROGRESS'`); subsequent grade requests simply replay the stored `Score`, rendering double-submission bugs impossible.

3. **Student Quizzes UI**
   - Expanded `coursework-api.ts` with strongly-typed `startAttempt`, `getAttempt`, and `submitAttempt` methods reflecting our rigid DTO boundaries.
   - Engineered the `StudentQuizzes.tsx` interface driving a seamless exam experience for the student.
   - Features dynamic input rendering using Shadcn Radio buttons (Single) and Checkboxes (Multi) based on the question type.
   - Review Mode dynamically shifts depending on the quiz window bounds, cleanly notifying the student if correct answers are suppressed or intelligently highlighting selections in red/green if the window has concluded.

## Next Steps

- This officially concludes the assignments and quizzes implementation (Phase 4).
- The subsequent phases can confidently rely on this robust academic structure to facilitate dashboard aggregation, scheduling, and analytics.

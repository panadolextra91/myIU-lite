# Information Architecture — myIU Lite Frontend

> **Purpose of this doc:** a complete, design-agnostic map of every screen, content block, data field, action, and state in the current frontend. It describes **WHAT** the UI contains and how it is **structured/hierarchised** — *not* how it should look. Pair it with your own `DESIGN.md` (colors, type, spacing, styling) when prompting Stitch. The screen and component counts here are fixed; Stitch should restyle, not add or remove.
>
> **Scope:** 22 page-screens + the app shell + the reusable component set, derived from `frontend/src`.
> - Shared/auth: 3 · Student: 6 · Lecturer: 6 · Admin: 7
> - Roles: **Student**, **Lecturer**, **Admin** (route-gated).

---

## 1. Product & Actors

**myIU Lite** — a university course-management platform. Three actors share one SPA, gated by role:

| Actor | Can do |
|-------|--------|
| **Student** | Submit assignments, take quizzes, view grades & announcements, send requests to lecturers, read notifications |
| **Lecturer** | Create/grade/finalize assignments, build quizzes, manage a grade scheme & publish grades, post announcements, reply to student requests |
| **Admin** | Manage accounts (CSV + manual), manage courses, enroll students, assign lecturers, read the audit trail |

The same chrome (top header, notifications, theme toggle) wraps all roles; only Admin gets a left sidebar.

---

## 2. Sitemap (route map)

```
/                              → redirect to /login
/login                         Login                         (public)
/change-password               Change Password               (auth, forced on first login)

  (authenticated app shell below — every route wrapped by the header layout)
/notifications                 Notifications inbox           (all roles)

/student                       Student Dashboard             (role: student)
/student/assignments           Student Assignments
/student/quizzes               Student Quizzes (list + attempt detail, same route)
/student/courses/:id/grades        Student Grades
/student/courses/:id/announcements Student Announcements (deep-link variant: /:announcementId)
/student/courses/:id/requests      Student Requests

/lecturer                      Lecturer Dashboard            (role: lecturer)
/lecturer/assignments          Lecturer Assignments
/lecturer/courses/:id/quizzes      Lecturer Quizzes (builder)
/lecturer/courses/:id/gradebook    Lecturer Gradebook
/lecturer/courses/:id/announcements Lecturer Announcements
/lecturer/requests             Lecturer Request Inbox

/admin                         Admin Dashboard               (role: admin, sidebar layout)
/admin/accounts                Accounts
/admin/courses                 Courses
/admin/courses/:id             Course Detail (tabbed)
/admin/enrollment              Student Enrollment (CSV)
/admin/lecturers               Lecturer Assignment (CSV)
/admin/audit                   Audit Logs
```

**Navigation note (IA gap to be aware of):** Admin navigates via the **left sidebar**. Student and Lecturer currently have **no in-app navigation menu** — their Dashboard is a bare welcome card and sub-pages are reached by direct URL. Most student/lecturer routes are also **course-scoped** (`/courses/:id/...`) but no course-picker UI exists yet. If your redesign should add nav for these roles, that is a structural decision to make in `DESIGN.md` — the current screens don't include it.

---

## 3. Global Chrome (app shell)

### 3.1 Top header — all authenticated roles
Sticky top bar, full width. Left → right:
- **Left:** sidebar toggle (admin only) · product wordmark "myIU Lite".
- **Right (when signed in):** `username (role)` label · **theme toggle** (light/dark) · **notification bell** (with unread count badge) · **Logout** button.

### 3.2 Admin sidebar (admin only)
Collapsible left sidebar, expanded by default; becomes a drawer on mobile. Grouped nav:

| Group | Items (label → route) |
|-------|----------------------|
| General | Dashboard → `/admin` |
| User Management | Accounts → `/admin/accounts` |
| Academic Management | Courses → `/admin/courses` · Student Enrollment → `/admin/enrollment` · Lecturer Assignment → `/admin/lecturers` |
| System | Audit Logs → `/admin/audit` |

Active item highlights based on the current path. Each item has a leading icon.

### 3.3 Notification bell (all roles)
A bell control in the header. Shows an unread-count badge (numeric; "99+" when over 99); badge hidden at zero. Links to `/notifications`.

### 3.4 Theme toggle (all roles)
Switches light/dark. Both themes are first-class (no pure-black dark mode).

---

## 4. Screens

> Each screen lists: **Purpose · Content blocks (in visual order, with hierarchy) · Data fields/columns · Primary actions · Modals · States (loading / empty / error)**. Styling intentionally omitted.

---

### 4A. Shared / Auth

#### Login — `/login`
- **Purpose:** authenticate; routes to role home on success.
- **Layout:** single centered card on a plain page.
- **Content:** card header (lock icon + "Login" title + subtitle "Enter your credentials to access your account") → optional inline error banner → form.
- **Form fields:** Username (text) · Password (password).
- **Actions:** "Sign in" (full-width submit).
- **States:** inline error banner on failed auth.

#### Change Password — `/change-password`
- **Purpose:** forced password change on first login.
- **Layout:** single centered card.
- **Content:** card header (key icon + "Change Password" + subtitle "You must change your password before continuing.") → optional inline error banner → form.
- **Form fields:** Current Password · New Password · Confirm New Password (all password).
- **Actions:** "Change Password" (full-width submit). On success → redirect to login.
- **States:** inline error banner (wrong current pwd, mismatch, too short, etc.).

#### Notifications — `/notifications`
- **Purpose:** read in-app notifications; click to mark-read and follow its link.
- **Layout:** narrow centered column, page heading "Notifications".
- **Content:** vertical list of notification cards. Each card: title (with an unread dot when unread) + relative timestamp + body text. Whole card is clickable.
- **States:** loading → 3 skeleton rows · empty → "No notifications yet."

---

### 4B. Student

#### Student Dashboard — `/student`
- **Purpose:** landing/home.
- **Content:** single card — title "Student area", body "Welcome to the student dashboard." *(placeholder; no widgets/nav yet)*.

#### Student Assignments — `/student/assignments`
- **Purpose:** view assignments for a course and submit files.
- **Layout:** header row (title "Assignments" + a numeric **Course ID** input) → table.
- **Table columns:** Title · Deadline · Status / Submissions · Submit.
- **Per row:** title · formatted deadline · submission status (download link "Download Submission v{n}", optional "Late: {duration}", or "No submission this session") · submit cell = file input (`.pdf,.zip`) + "Submit" button (disabled until a file is chosen).
- **States:** empty → "No assignments found." row.

#### Student Quizzes — `/student/quizzes` (two views, one route)
- **Purpose:** browse available quizzes; take/resume/review an attempt.
- **List view:** heading "Course Quizzes" → 2-col card grid. Each quiz card: title + status label ("Available" / "Opens at {time}" / "Closed at {time}") + meta line "Questions: {n} | Max Grade: {n}" + full-width action button ("Take / Resume Quiz" or "Review Attempt").
- **Attempt-detail view:** header "Attempt #{n}" + "Back to Quizzes" button → optional score card ("Score: {n}" + reveal-status note) → vertical list of question cards.
  - **Question card:** "Question {i}" + prompt + options list. Single-choice = radio group; multi-choice = checkboxes. Options visually mark selected / correct / incorrect after the window closes; controls disabled in terminal states.
  - **Submit:** "Submit Attempt" (only while in progress).

#### Student Grades — `/student/courses/:id/grades`
- **Purpose:** view published grade components and overall.
- **Layout:** heading "My Grades" → grades table.
- **Table columns:** Component · Weight · Score (right-aligned).
- **Per row:** component name · "{weight}%" · score, or "Not published" when unpublished. Final **Overall** row: "Overall" + overall score or "Pending".
- **States:** loading → skeleton · no-scheme → "Not Available" info card ("No grade scheme has been set for this course yet.").

#### Student Announcements — `/student/courses/:id/announcements` (+ `/:announcementId` deep-link)
- **Purpose:** read course announcements; deep-link can scroll to one.
- **Layout:** header (title "Announcements" + subtitle "View all announcements for this course.") → list.
- **Per card:** title + relative timestamp + body (preserves line breaks).
- **States:** loading → 3 skeleton cards · empty → "No announcements found."

#### Student Requests — `/student/courses/:id/requests`
- **Purpose:** send a request to a lecturer; view history + replies.
- **Layout:** header (title "My Requests" + subtitle) → "Send Request" card → "Request History" list.
- **Send Request form:** Send To (lecturer dropdown) · Request Type (Leave Early / Absence / Custom) — these two side-by-side on desktop · Title (text, e.g. "Medical Appointment") · Details (textarea). Action: "Send Request" ("Sending…" while pending).
- **History per card:** status badge (**Pending / Approved / Denied**) + title; subtitle "Type: {type} | Sent {relative}"; body; and, once answered, a "Lecturer Reply" block (reply text + reply time, or "No additional notes provided.").
- **States:** loading → 2 skeleton cards · empty → "You haven't sent any requests."

---

### 4C. Lecturer

#### Lecturer Dashboard — `/lecturer`
- **Content:** single card — title "Lecturer area", body "Welcome to the lecturer dashboard." *(placeholder)*.

#### Lecturer Assignments — `/lecturer/assignments`
- **Purpose:** create assignments; grade submissions; finalize.
- **Layout:** header row (title "Assignments" + numeric **Course ID** input + "Create Assignment" button) → table.
- **Table columns:** ID · Title · Deadline · Max Score · Accept Late · Finalized · Actions.
- **Per row Actions:** "Grade" · "Finalize" (only when not yet finalized).
- **Create Assignment modal** ("New Assignment"): Title · Description (optional) · Deadline (datetime) · Accept Late Submissions (checkbox) · Late Threshold (Days) (optional number) · Max Score (number, default 100). Action: "Save".
- **Grade Submission modal** ("Grade Submission"): Submission ID (number) + "Download" button · Score (0–100) · Feedback (optional). Action: "Submit Grade".
- **States:** empty → "No assignments found." row.

#### Lecturer Quizzes (builder) — `/lecturer/courses/:id/quizzes`  *(most complex screen)*
- **Purpose:** create quizzes, then author questions (UI form or CSV import).
- **Layout:** header (title "Quizzes" + "Create Quiz" button) → 2-col quiz card grid → conditional **authoring panel** when a quiz is selected.
- **Quiz card:** title + meta "Pool: {n} | Questions: {n} | Max Grade: {n}" + date-range line. Clicking a card opens its authoring panel.
- **Create Quiz modal** ("Create New Quiz"): Title · Pool Size · Max Questions · Max Grade · Retake Count (numbers, mostly 2-col) · Shuffle Questions (switch) · Open At · Close At (datetime). Action: "Create".
- **Authoring panel** ("Author Questions for Quiz #{id}"), two sub-sections:
  - **Import via CSV:** file input (`.csv`) + "Import CSV" button + helper text `Format: question,A,B,C,D,correct`.
  - **Add UI Question:** prompt input · question-type toggle (Single Choice / Multi Choice) · options list (each option = a radio/checkbox correctness marker + text input + a delete control when >2 options) · "Add Option" · "Save Question" (full width).
- **States:** loading placeholder for the question area · empty grid → "No quizzes created yet."

#### Lecturer Gradebook — `/lecturer/courses/:id/gradebook`
- **Purpose:** define a weighted grade scheme; import manual scores via CSV; publish.
- **Layout:** header (title "Gradebook" + either "Create Scheme" or "Delete Scheme") → "Scheme Structure" table (when a scheme exists) → "Live Overall Grades" table (when grades exist).
- **Create Grade Scheme modal** (component field-array). Per component row: Name · Weight (number) · Source (MANUAL / AUTO / Composite) · Auto Kind (Quizzes / Assignments — only when AUTO) · Parent Index (optional) · Remove. Plus "Add Component" and "Save Scheme".
- **Scheme Structure table** columns: Component · Weight · Source · Actions (per-row: a CSV upload for MANUAL components; a "Publish" button for top-level components).
- **Live Overall Grades table** columns: Student ID · one column per non-parent component · **Overall**.
- **States:** loading → skeleton · no-scheme → only the "Create Scheme" action shows.

#### Lecturer Announcements — `/lecturer/courses/:id/announcements`
- **Purpose:** compose announcements (broadcast or targeted); view sent history.
- **Layout:** header (title "Announcements" + subtitle) → "Compose Announcement" card → "Sent Announcements" list.
- **Compose form:** Title (text) · Body (textarea) · Audience (All Enrolled Students / Specific Students) · when "Specific" → a scrollable checkbox grid of enrolled students (`Full Name (username)`). Note "Announcements cannot be edited or deleted once sent." Action: "Send Announcement".
- **Sent history per card:** title + relative time; subtitle "To: All Students | Specific Students"; body.
- **States:** student-list loading → skeletons / "No students enrolled." · history loading → skeleton cards · history empty → "You haven't sent any announcements yet."

#### Lecturer Request Inbox — `/lecturer/requests`
- **Purpose:** read student requests; reply approve/deny.
- **Layout:** header (title "Request Inbox" + subtitle) → list of request cards.
- **Per card:** status badge (**Pending / Approved / Denied**) + title; subtitle "Type: {type} | From Student ID: {id} | Sent {relative}"; body; a "Reply" button (only while Pending); once answered, a "Your Reply" block (note + reply time, or "No additional notes provided.").
- **Reply modal** ("Reply to Request"): Decision (Approve / Deny) · Note (optional textarea). Action: "Send Reply".
- **States:** loading → 3 skeleton cards · empty → "Your inbox is empty."

---

### 4D. Admin

#### Admin Dashboard — `/admin`
- **Content:** single card — title "Admin area", body "Welcome to the admin dashboard." *(placeholder)*.

#### Accounts — `/admin/accounts`
- **Purpose:** manage student/lecturer accounts; CSV import; manual create; reset password.
- **Layout:** header (title "Accounts" + subtitle) + action buttons ("Import Students", "Import Lecturers", "Create Manual") → optional CSV-error panel → filter bar → table → pagination.
- **Filter bar:** search input ("Search ID or name…") · role filter (All Roles / Student / Lecturer / Admin).
- **Table columns:** ID / Username · Full Name · Role (badge) · Status (badge: "Active" / "Needs Password Change") · DOB · Joined · action.
- **Per-row action:** "Reset Pwd" → confirmation dialog ("Reset Password?" — resets to DOB-as-password, forces change next login).
- **Create Account modal** ("Create Account"): Role (Student / Lecturer) · ID (Username) (e.g. "S12345") · Full Name · Date of Birth (DD/MM/YYYY). Action: "Create".
- **CSV-error panel (conditional):** "CSV Import Failed — {N} validation errors" + scrollable error table (Row · Field · Message).
- **States:** loading → "Loading…" row · empty → "No accounts found." · pagination "Showing X to Y of Z" + Previous/Next.

#### Courses — `/admin/courses`
- **Purpose:** create/edit/soft-delete courses.
- **Layout:** header (title "Courses" + subtitle + "Create Course") → filter bar → table → pagination.
- **Filter bar:** search ("Search code or name…") · term filter ("Filter by term…").
- **Table columns:** Code (links to Course Detail) · Name · Term · Start Date · End Date · Actions (Edit · Delete).
- **Create/Edit Course modal** ("Create Course" / "Edit Course"): Course Code ("CS101") · Course Name ("Intro to Computer Science") · Term ("Spring 2026") · Start Date · End Date (dates, 2-col; end ≥ start). Action: "Create" / "Update".
- **Delete:** confirmation dialog ("Soft-delete Course?" — hidden from lists, history preserved).
- **States:** loading → "Loading…" · empty → "No courses found." · pagination as above.

#### Course Detail — `/admin/courses/:id`
- **Purpose:** view a course; manage enrolled students & assigned lecturers.
- **Layout:** header ("{code} - {name}" + "{term} • {start} to {end}") → tabs: **Overview · Students · Lecturers**.
- **Overview tab:** stat cards — "Students Enrolled" (count) · "Lecturers" (count).
- **Students tab:** table (Student ID · Name · action "Remove" → "Remove Student?" confirm). Empty → "No students enrolled."
- **Lecturers tab:** table (Lecturer ID · Name · action "Unassign" → "Unassign Lecturer?" confirm). Empty → "No lecturers assigned."
- **States:** per-tab loading → "Loading…".

#### Student Enrollment — `/admin/enrollment`
- **Purpose:** bulk-enroll students into a course via CSV.
- **Layout:** header (title "Student Enrollment" + subtitle) → course selector + "Import Student CSV" button → "CSV Format Requirements" box → optional error table.
- **Course selector:** dropdown of courses (`{code} - {name} ({term})`); skeleton while loading.
- **Format box:** bulleted rules (column `student_id`, valid active IDs, rejected on any invalid/duplicate, already-enrolled skipped).
- **Error table (conditional):** Row · Field · Error Message, preceded by "Import failed…" message.

#### Lecturer Assignment — `/admin/lecturers`
- **Purpose:** bulk-assign lecturers to a course via CSV. *(Same structure as Enrollment.)*
- **Layout:** header (title "Lecturer Assignment" + subtitle) → course selector + "Import Lecturer CSV" → "CSV Format Requirements" box (column `lecturer_id`, valid active IDs, reject on invalid/duplicate, already-assigned skipped) → optional error table (Row · Field · Error Message).

#### Audit Logs — `/admin/audit`
- **Purpose:** read the system-wide admin audit trail.
- **Layout:** header (title "Audit Logs" + subtitle) → "Recent Activity" card containing a table.
- **Table columns:** Timestamp · Actor ID (or "SYSTEM") · Action · Target ("{type} #{id}") · Affected (count).
- **States:** loading → "Loading…" · empty → "No logs found."

---

## 5. Reusable component inventory

These are the shared UI primitives the screens compose from (shadcn/ui). Counts/identity are fixed; restyle in place.

| Primitive | Used by |
|-----------|---------|
| **Button** (variants: default / outline / ghost / link / destructive; sizes incl. icon) | every screen |
| **Card** (Header / Title / Description / Content) | dashboards, lists, panels |
| **Table** (Header / Body / Row / Head / Cell) | assignments, grades, accounts, courses, audit, gradebook |
| **Dialog** (modal) | create/edit forms, grade, quiz create, reply, scheme |
| **AlertDialog** (confirm) | reset pwd, delete course, remove student, unassign lecturer |
| **Form** (Field / Item / Label / Control / Message) | login, change pwd, requests, create flows |
| **Input** (text / number / password / date / datetime-local / file) | all forms & filters |
| **Textarea** | request details, announcement body, reply note |
| **Select** (Trigger / Content / Item / Value) | role, request type, audience, course pickers, scheme source |
| **Checkbox** | accept-late, multi-choice options, specific-student picker |
| **RadioGroup** | single-choice quiz options, question-type toggle |
| **Switch** | shuffle questions |
| **Tabs** (List / Trigger / Content) | Course Detail |
| **Badge** | role, account status, request status, notification count |
| **Skeleton** | loading states across lists/tables/cards |
| **Sidebar** (Group / Menu / Item / Button / Trigger) | admin shell |
| **Sheet** | mobile sidebar drawer |
| **Separator**, **Label**, **Tooltip** | supporting |
| **Sonner (toasts)** | success/error feedback after mutations |

App-shell components (not shadcn primitives): **AppLayout** (header + outlet, sidebar for admin), **AdminSidebar**, **NotificationBell**, **ModeToggle**, **ThemeProvider**.

---

## 6. Cross-cutting interaction patterns

Restyle these consistently everywhere they appear:

- **Forms** — labeled fields, inline validation messages under each field, full-width or footer submit; submit disables + shows "…ing" text while pending.
- **Tables** — header row + body rows; numeric columns right-aligned; explicit empty-state row; "Loading…" placeholder or skeleton while fetching.
- **Modals** — create/edit share one dialog whose title flips (Create vs Edit). Destructive actions always go through a confirm dialog with an explicit consequence sentence.
- **CSV import** — hidden file input behind a button; on failure a structured error table (**Row · Field · Message**) with a summary line; never partial-applies.
- **Status indicators** — request status (Pending / Approved / Denied), account status (Active / Needs Password Change), quiz status (Available / Opens at / Closed at). All are text-labeled (not color-only) for accessibility.
- **Relative timestamps** — notifications, announcements, requests show "x ago"; audit log shows absolute `yyyy-MM-dd HH:mm:ss`.
- **Pagination** — "Showing X to Y of Z" + Previous/Next, disabled at bounds (Accounts, Courses).
- **Loading** — skeleton placeholders only; no shimmer/bounce/decorative animation.
- **Theming** — every screen must work in both light and dark; both are first-class.
- **Accessibility** — keyboard navigable, visible focus, WCAG AA contrast, semantic structure, no color-only meaning.

---

*Generated from `frontend/src` for the Stitch redesign. Screen & component counts are fixed — change layout/visual design only.*

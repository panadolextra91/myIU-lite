# Stitch Redesign — Screen Manifest

Generated designs for the **Dark Academia** redesign of myIU Lite. Use this to fetch each screen during implementation (`get_screen` via the Stitch MCP, or open in Stitch web).

- **Stitch project:** `projects/17051294795705954583` (title: "myIU Lite — Dark Academia Redesign")
- **Design system asset:** `assets/1ba89e16d08d4638930f15be6db5f0bb` (name: "Academic Manuscript")
- **Source specs:** `IA.md` (content/structure) + `DESIGN.md` (visual language)
- Durable reference = `projectId` + `screenId`. Screenshot URLs are temporary signed links and will expire — re-fetch via `get_screen`.

> **Dashboards:** generated as editorial "table of contents" landings (welcome heading + links to the role's existing sections). No invented stats/KPI/charts — they also fill the student/lecturer nav gap noted in `IA.md`.
>
> **Known cleanups at implement time (build from IA, not pixel-copy):**
> - Strip invented footers — a few screens snuck one in: `Student Grades` ("University Archive Register · 1924–2024"), `Student Dashboard`, `Admin Dashboard`. Do **not** render them.
> - `Student Dashboard` "Assignments" row shows a broken icon token `DRIVE_TEXT` — use a proper Lucide icon (e.g. `FileText`).
> - Admin sidebar labels drifted ("Overview"/"Enrollment"/"Faculty"/"Assignment") — keep the IA labels (Dashboard / Student Enrollment / Lecturer Assignment) from `AdminSidebar.tsx`.

| # | Screen | Route | Stitch screenId |
|---|--------|-------|-----------------|
| 1 | Login | `/login` | `65131062f9664ed186b59da7c7cd8081` |
| 2 | Change Password | `/change-password` | `ebd5829ca76d43ce9870d2b806fa3bca` |
| 3 | Notifications | `/notifications` | `9cb6db7742bf45f9a241fee9478f031d` |
| 4 | Student Assignments | `/student/assignments` | `af85343ad78748f48e1df9119b7bf88b` |
| 5 | Student Quizzes (list) | `/student/quizzes` | `53e7570d90e140cfba993b06719b2f6f` |
| 6 | Student Quiz Attempt | `/student/quizzes` (attempt) | `54c69d51d1214c708a9e90acc8705984` |
| 7 | Student Grades | `/student/courses/:id/grades` | `e51a7fc72a31413f9e97c2c7376b9bef` |
| 8 | Student Announcements | `/student/courses/:id/announcements` | `3b3850853d56483dacfe53f777140fce` |
| 9 | Student Requests | `/student/courses/:id/requests` | `7b3f0390f56e46878081a734cf6fca25` |
| 10 | Lecturer Assignments | `/lecturer/assignments` | `82089e92e43547bcb81ac7d7af29a5c9` |
| 11 | Lecturer Quizzes (builder) | `/lecturer/courses/:id/quizzes` | `750efe92a8fd49f5a565fdcf7f3371a9` |
| 12 | Lecturer Gradebook | `/lecturer/courses/:id/gradebook` | `77a1234d8c1f466492b74553be34a914` |
| 13 | Lecturer Announcements | `/lecturer/courses/:id/announcements` | `60ebef9d2c504134979fb36b49da36c4` |
| 14 | Lecturer Request Inbox | `/lecturer/requests` | `fb08684ccaba4a9b883df2a05a7ca2f8` |
| 15 | Admin Accounts | `/admin/accounts` | `04715249a60144b9bce74eb5210c4805` |
| 16 | Admin Courses | `/admin/courses` | `faa09fb86d964a7cbe7f43a209f437aa` |
| 17 | Admin Course Detail | `/admin/courses/:id` | `6eb0bead6970464e89e33e5849575760` |
| 18 | Admin Student Enrollment | `/admin/enrollment` | `e25a2fe638bc49dbb26dbb9eab5cbd06` |
| 19 | Admin Lecturer Assignment | `/admin/lecturers` | `09bea3d717c446469cafc3ae8f1f6d5c` |
| 20 | Admin Audit Logs | `/admin/audit` | `ba97867f8d6a48c4a6cf54ad4d5b5e17` |
| 21 | Student Dashboard | `/student` | `36f6c0fc135e40ea8872dcd333dd76ad` |
| 22 | Lecturer Dashboard | `/lecturer` | `a5536f0bde6a42cd99149f13e4605725` |
| 23 | Admin Dashboard | `/admin` | `3c3fab59fe3648e5978a7c928d40f30b` |

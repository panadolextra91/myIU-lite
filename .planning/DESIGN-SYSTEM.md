# D-05 — UI Design System & Theme Specification

> Single source of truth for UI across the whole project. There is **no per-phase UI-SPEC**;
> every frontend phase reads and conforms to this file.

## Decision

The system will use a minimal, accessibility-focused design system built entirely on shadcn/ui components. No custom UI components will be developed unless shadcn/ui does not provide an equivalent.

The UI prioritizes readability, low visual noise, and maintainability over branding-heavy aesthetics.

---

## Theme Strategy

The application supports both **Light Theme** and **Dark Theme**. Each theme uses a separate color scheme optimized for long administrative and academic workflows.

### Light Theme

| Token          | Color   |
| -------------- | ------- |
| Background     | #F8FAFC |
| Primary Text   | #0F172A |
| Secondary Text | #64748B |
| Primary Accent | #2563EB |
| Success Accent | #16A34A |
| Warning Accent | #D97706 |

Characteristics:

- Clean academic appearance
- High readability
- Minimal eye fatigue
- Suitable for office and classroom environments

### Dark Theme

| Token          | Color   |
| -------------- | ------- |
| Background     | #0F172A |
| Primary Text   | #F8FAFC |
| Secondary Text | #94A3B8 |
| Primary Accent | #3B82F6 |
| Success Accent | #22C55E |
| Warning Accent | #F59E0B |

Characteristics:

- Comfortable for extended usage
- Consistent contrast ratio
- No pure-black backgrounds

---

## Component Styling

### Border Radius

Global border radius:

```css
6px
```

Applied to:

- Buttons
- Cards
- Inputs
- Dialogs
- Tables
- Dropdowns
- Popovers

### Shadow

All cards and elevated surfaces use subtle shadows only.

```css
box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
```

Avoid heavy floating-card effects.

### Loading State

Use shadcn/ui **Skeleton** components for:

- Tables
- Dashboard cards
- Assignment lists
- Quiz lists
- Announcements
- Grade views

Do not use loading animations beyond standard skeleton placeholders. No shimmer effects. No bouncing loaders. No decorative animations.

---

## Navigation

### Sidebar

Desktop:

- Expandable/collapsible sidebar
- Default state: expanded

Mobile:

- Hamburger menu
- Drawer-based navigation

Sidebar contains:

- Dashboard
- Courses
- Assignments
- Quizzes
- Announcements
- Requests
- Grades
- Administration (role-based)

---

## Icons

Use **Lucide React** exclusively.

Rationale:

- Native compatibility with shadcn/ui
- Lightweight
- Consistent stroke design
- Excellent TypeScript support

Examples: Home, BookOpen, GraduationCap, FileText, ClipboardList, Bell, Settings, Users, ShieldCheck

---

## Accessibility

Minimum **WCAG AA** contrast target.

Requirements:

- Keyboard navigation supported
- Visible focus states
- Semantic HTML from shadcn/ui
- No color-only status indicators

---

## Design Principle

The system should feel like:

> "University administration software that students can understand immediately."

Not:

- A startup dashboard
- A crypto exchange
- A social network
- A marketing website

---
*Recorded: 2026-06-19 (decision D-05)*

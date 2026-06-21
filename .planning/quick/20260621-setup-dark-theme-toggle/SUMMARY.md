---
status: complete
---

# Setup dark theme and toggle button in header

**Execution Summary:**
- Cleaned up `index.css` to use proper CSS variables mapped from `DESIGN-SYSTEM.md` for both Light and Dark themes.
- Implemented `ThemeProvider` using `next-themes` in `theme-provider.tsx` and wrapped `App` inside `main.tsx`.
- Created `ModeToggle` component with Lucide `Sun` and `Moon` icons.
- Added `<ModeToggle />` in `AppLayout.tsx` next to the notification bell.

All UI updates conform strictly to the standard theme and design guidelines.

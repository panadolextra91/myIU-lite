---
status: in-progress
---

# Setup dark theme and toggle button in header

## Task Description
Implement the dark theme specification from `DESIGN-SYSTEM.md` and add a dark mode toggle button in the application header.

## Steps
1. **Update `index.css`**: Map the specific HEX colors from `DESIGN-SYSTEM.md` to the `:root` (Light Theme) and `.dark` (Dark Theme) CSS variables. Remove redundant/default Shadcn colors.
2. **Create `theme-provider.tsx`**: Create a wrapper component using `next-themes` to manage the theme state (light, dark, system).
3. **Update `main.tsx`**: Wrap the `<App />` with the `<ThemeProvider>`.
4. **Create `mode-toggle.tsx`**: Create a UI button (using Lucide icons `Sun` and `Moon`) that toggles the theme.
5. **Update `AppLayout.tsx`**: Insert the `<ModeToggle />` component next to the `<NotificationBell />` in the header.

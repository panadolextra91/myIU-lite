# DESIGN.md

# myIU Lite Redesign

## Theme: Modern Dark Academia

---

# Design Intent

Redesign the existing application using the visual language of **Dark Academia**, while preserving every screen, workflow, route, interaction, and component described in `IA.md`.

Do **not** redesign the product.

Do **not** invent new features.

Do **not** remove functionality.

Only redesign the visual presentation.

The application should feel like software built for a prestigious university library rather than a modern startup.

Think:

* Oxford Library
* Cambridge Reading Room
* Yale Manuscript Collection
* Quiet study spaces
* Leather-bound books
* Linen paper
* Walnut furniture
* Brass lamps
* Fountain pens
* Academic journals

Avoid fantasy.

Avoid medieval roleplay.

Avoid Harry Potter aesthetics.

Avoid castles, magic, candles, skulls, ravens, or theatrical gothic decoration.

This is a modern web application inspired by timeless academic elegance.

---

# Core Principles

The interface should prioritize:

* Reading before interaction
* Calm visual rhythm
* Strong typography
* Editorial composition
* Information hierarchy
* Spacious layouts
* Quiet confidence

The application should never feel playful or flashy.

---

# Visual Personality

The interface should evoke:

* printed books
* research papers
* university yearbooks
* archival documents
* museum catalogs
* library collections

Every page should feel carefully typeset rather than assembled from generic UI cards.

---

# Color Palette

## Background

Paper

#F6F2EA

---

## Surface

Ivory

#FBF8F3

---

## Elevated Surface

#FFFFFF

Use sparingly.

---

## Primary Text

Ink Black

#1F1D1A

---

## Secondary Text

Warm Gray

#5D564F

---

## Border

#D7CEC2

Borders should define hierarchy more than shadows.

---

## Primary Accent

Forest Green

#3F5B4B

Primary buttons

Links

Focused controls

Navigation highlights

---

## Secondary Accent

Oxford Blue

#243447

Charts

Information

Secondary actions

---

## Highlight

Antique Gold

#A7864B

Use only for:

* active navigation
* important statistics
* highlighted metadata

Never use gold for large surfaces.

---

## Success

#476A52

---

## Warning

#A07A3F

---

## Error

#8B3A3A

---

## Neutral

Stone

#E7E0D6

---

# Dark Theme — Color Palette

The application keeps its light/dark toggle. Dark mode is the **same Dark Academia world at night**: a walnut reading room lit by a single brass lamp. Warm near-black browns (never pure black, never cold slate), aged-parchment text, and the same Forest Green / Oxford Blue / Antique Gold accents **lifted for legibility** on dark surfaces.

Each token below is the dark counterpart of the light token with the same name.

| Token | Name | Hex |
| --- | --- | --- |
| Background | Walnut Night | #1C1813 |
| Surface | Aged Oak | #24201A |
| Elevated Surface | Lamplit Wood (use sparingly) | #2C2720 |
| Primary Text | Parchment | #ECE5D8 |
| Secondary Text | Faded Ink | #ACA290 |
| Border | Dim Walnut | #3A3328 |
| Primary Accent | Sage Green (Forest, lifted) | #7BA088 |
| Secondary Accent | Steel Blue (Oxford, lifted) | #7C97B2 |
| Highlight | Brass (Antique Gold, lifted) | #C6A867 |
| Success | Muted Forest | #6FA07D |
| Warning | Aged Amber | #C39A5A |
| Error | Muted Wine | #BD6B6B |
| Neutral | Dark Stone | #2E2820 |

Notes:

* Backgrounds are warm brown-black — not slate, never #000.
* Hierarchy still comes from borders and spacing, not shadows. Keep "borders over elevation" in dark too.
* Antique Gold finally reads well on dark — still reserve it for active navigation, key statistics, and highlighted metadata. Never large surfaces.
* Focus state: Sage Green border (the dark counterpart of the Forest Green focus).
* Primary buttons: Sage Green fill with Walnut Night text. Danger: Muted Wine.
* WCAG AA preserved: Parchment, Faded Ink, and all accents are legible on Walnut Night.

---

# Typography

Typography is the primary visual identity.

Use typography to create hierarchy rather than oversized cards.

## Heading Font

Cormorant Garamond

Fallback:

EB Garamond

Large elegant headings.

Never bold-heavy.

---

## Body Font

Inter

Readable.

Modern.

Neutral.

---

## Numbers

JetBrains Mono

Used for:

* grades
* IDs
* percentages
* timestamps
* statistics

---

## Scale

H1
40px

H2
30px

H3
24px

Body
16px

Caption
14px

Metadata
13px

Use generous line-height.

---

# Layout

Desktop-first.

Content width:

1280px

Wide margins.

Large whitespace.

Every section should breathe.

Avoid visual clutter.

---

# Grid

Use a consistent 12-column layout.

Tables should occupy available width.

Forms should never appear cramped.

---

# Border Radius

Cards

8px

Buttons

8px

Inputs

8px

Dialogs

12px

Never exceed 12px.

Avoid overly rounded components.

---

# Shadows

Minimal.

Use:

0 1px 2px rgba(0,0,0,0.06)

Do not create floating interfaces.

Prefer borders over elevation.

---

# Borders

Very important.

Most hierarchy should come from:

* spacing
* typography
* borders

Not shadows.

---

# Cards

Cards resemble archival paper sheets.

Use:

* generous padding
* subtle borders
* almost no shadow

Avoid dashboard-style floating cards.

---

# Tables

Tables are one of the primary interface elements.

Treat them like elegant printed documents.

Header

Uppercase

Small letter spacing

Muted color

Thin bottom border

Rows

Large vertical padding

Subtle hover

No zebra striping

Numeric columns right-aligned.

---

# Forms

Forms should resemble well-designed paper forms.

Labels always visible.

Large spacing.

Comfortable inputs.

No floating labels.

Focus state:

Forest Green border.

No glowing effects.

---

# Buttons

Primary

Forest Green

Solid

Secondary

Outline

Ghost

Transparent

Danger

Muted Wine

Buttons should feel restrained.

---

# Navigation

Navigation resembles a table of contents.

Large spacing.

Elegant typography.

Simple line icons.

No oversized icons.

No colorful navigation.

---

# Icons

Lucide Icons.

Outline only.

Stroke:

1.5

Keep icons secondary to typography.

---

# Charts

Use muted colors.

Forest

Oxford Blue

Antique Gold

Warm Brown

Avoid neon.

Avoid gradients.

Avoid excessive chart decoration.

---

# Status Badges

Success

Muted Forest

Pending

Warm Brown

Warning

Muted Gold

Error

Muted Wine

Information

Oxford Blue

Badges should be readable without relying only on color.

---

# Images

Photography should be used sparingly.

Avoid illustrations.

Avoid 3D graphics.

Avoid abstract gradients.

If images exist, they should feel editorial.

---

# Empty States

Minimal.

Use:

* outline icon
* concise explanation
* optional action button

Do not use cartoon illustrations.

---

# Skeleton Loading

Warm paper colors.

Very subtle shimmer.

No bright gray placeholders.

Rounded corners.

Consistent spacing.

---

# Hover States

Cards

Slight border darkening.

Tiny shadow.

Buttons

Subtle darkening.

Links

Animated underline.

Tables

Soft paper-colored background.

Navigation

Background tint.

No scaling.

---

# Motion

Motion should feel quiet.

Hover

150ms

Page transition

200ms fade

Dialogs

Fade + slight upward movement

Dropdown

Fade only

Avoid bounce.

Avoid spring animation.

Avoid excessive movement.

---

# Scroll Experience

Scrolling should resemble reading a carefully edited publication.

Alternate naturally between:

* headings
* tables
* forms
* whitespace

Avoid long uninterrupted walls of content.

---

# Responsive Design

Desktop

Elegant workspace.

Tablet

Maintain reading comfort.

Mobile

Readable.

Forms stack naturally.

Tables may become cards where appropriate.

Do not sacrifice hierarchy.

---

# Accessibility

WCAG AA contrast.

Keyboard navigation.

Visible focus rings.

Readable typography.

Large interaction targets.

Reduced motion support.

---

# Visual Constraints

Do not introduce:

* Glassmorphism
* Heavy gradients
* Neon colors
* Floating dashboards
* Oversized KPI cards
* Excessive shadows
* Decorative textures
* Fantasy ornaments
* Gothic symbols
* Medieval props

Dark Academia should be expressed through:

* typography
* composition
* spacing
* restrained colors
* editorial rhythm

not decorative objects.

---

# Overall Feeling

Users should feel as if they are studying inside a quiet university reading room.

The interface should communicate knowledge, discipline, and timeless academic craftsmanship.

It should feel premium, calm, trustworthy, and deeply focused.

The user should never think:

"This is a startup dashboard."

Instead, they should feel:

"This is software designed with the same care as a beautifully printed academic publication."

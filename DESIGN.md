# Design

## Source of truth
- Status: Active
- Last refreshed: 2026-09-01
- Primary product surfaces: create, unlock, outline, focused branch, search, settings.

## Brand
- Personality: calm, precise, private, and lightweight.
- Trust signals: restrained interface, clear persistence state, predictable keyboard behavior, no third-party assets.
- Avoid: warm editorial styling, heavy shadows, dashboard chrome, decorative explanations, and technical security jargon in product copy.

## Product goals
- Goals: keep capture immediate; make structure and retrieval feel like one continuous document; keep access controls understandable.
- Non-goals: schemas, dashboards, rich text, collaboration, or visible infrastructure concepts.
- Success signals: after unlock the outline dominates the viewport; every common action is keyboard-accessible; search reshapes the existing tree rather than opening a second content surface.

## Personas and jobs
- Primary personas: individual knowledge workers and technical users.
- User jobs: capture a thought, organize it by nesting, recall it through text/metadata, and reopen it privately elsewhere.
- Key contexts of use: sustained desktop writing and quick retrieval.

## Information architecture
- Primary navigation: persistent compact header with brand, search, theme, persistence, and settings.
- Core routes/screens: create/unlock at `/`; outline at `/p/<id>#<secret>`.
- Content hierarchy: breadcrumb, filtered or focused outline, block rows.

## Design principles
- Restrained precision: silver surfaces, neutral typography, thin borders, violet/blue accents.
- One surface: search, focus, and editing operate on the same tree.
- Progressive disclosure: settings and autocomplete appear only when invoked.
- Tradeoffs: text and keyboard reliability take priority over animation and decorative UI.

## Visual language
- Color: `#f7f8fb` / `#1d2027` page surfaces, raised neutral panels, violet `#786bea`, blue `#4c85d9`, red only for destructive/error states.
- Typography: local system sans-serif; headings around 650 weight with tight tracking.
- Spacing/layout rhythm: 4/8px base, 24–72px structural gaps, outline width around 72ch.
- Shape/radius/elevation: 10–14px radii, one-pixel borders, minimal shadows.
- Motion: subtle functional transitions only; disabled for reduced motion.
- Imagery/iconography: compact geometric wordmark and text/icons only.

## Components
- Existing components to reuse: `App.svelte`, `BlockNode.svelte`, core tree/search modules.
- New/changed components: theme toggle, block bullet control, inline autocomplete, tree-filtered search state.
- Variants and states: light/dark; saved/saving/error/conflict; matched/context block; collapsed/focused block.
- Token/component ownership: `web/src/ui/styles.css` owns tokens and themes; Svelte components own interaction state.

## Accessibility
- Target standard: WCAG 2.2 AA where practical.
- Keyboard/focus behavior: ArrowUp/ArrowDown cross block boundaries only at textarea edges; Shift+Enter inserts a newline; focus-visible outlines remain prominent.
- Contrast/readability: text and muted states retain readable contrast in both themes.
- Screen-reader semantics: named controls, real buttons, breadcrumbs, status messages.
- Reduced motion and sensory considerations: no essential hover-only control; reduced-motion removes transitions.

## Responsive behavior
- Supported breakpoints/devices: modern desktop and mobile browsers from 360px.
- Layout adaptations: auth panel becomes single-column; header compacts; outline indentation narrows.
- Touch/hover differences: block bullets remain visible and tappable; actions do not depend on hover.

## Interaction states
- Loading: short neutral status without implementation detail.
- Empty: immediate editable block or `No matches` during search.
- Error: concise recovery-oriented copy.
- Success: quiet saved indicator.
- Disabled: controls visually muted and non-interactive.
- Offline/slow network: preserve local editing and show a compact save error.

## Content voice
- Tone: short, human, direct.
- Terminology: page, link, password, block, search; avoid ciphertext, KDF, opaque storage, fragment, CAS, or implementation explanations.
- Microcopy rules: say what the user can do or what happened; do not explain internal mechanisms on screens or tooltips.

## Implementation constraints
- Framework/styling system: Svelte 5, TypeScript, plain CSS variables, no UI dependency.
- Design-token constraints: light/dark share violet/blue accents; surface/text/border tokens switch by theme.
- Performance constraints: local search should remain responsive at 10,000 blocks; no external fonts or images.
- Compatibility constraints: modern evergreen browsers and native Web Crypto.
- Test/screenshot expectations: Svelte checks, Vitest, Playwright, Go tests, and light/dark screenshots for material UI changes.

## Open questions
- None for the current iteration.

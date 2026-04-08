# Design System — AI Mafia

## Product Context
- **What this is:** AI Mafia is a multiplayer social deduction game where humans play Mafia against AI opponents. The core tension is whether you can tell the AI from real players.
- **Who it's for:** Friends playing together online; anyone who wants to test their instincts against AI that reasons and bluffs.
- **Space/industry:** Multiplayer party / social deduction games
- **Project type:** Web app — real-time game platform

## Aesthetic Direction
- **Direction:** Industrial Noir / Dossier ("Case File")
- **Decoration level:** Intentional — texture over glow, no decorative blobs
- **Mood:** A psychological thriller disguised as a game UI. Every screen should feel like reading a classified dossier. Gravity over bounce. The social deduction tension lives in the visual language itself — not sci-fi, not casual party game, but something that takes the paranoia seriously.
- **Key differentiators from the category:**
  1. Serif display type — literally nobody in social deduction games uses this; reads as theatrical and slightly menacing
  2. Warm amber/gold instead of blue/purple/neon — complete departure from current palette and category conventions
  3. Player list rendered as a ledger (monospaced rows, "ELIMINATED" stamp) not an avatar grid

## Typography
- **Display/Hero:** Instrument Serif — dramatic thick/thin strokes, reads sinister at large scale. Use at 48px+ with letter-spacing: -0.02em. For emotional headers and landing headlines.
- **Body:** DM Sans — neutral, slightly cold. Creates productive tension against the serif. Weights 300, 400, 500, 600. Line-height 1.6.
- **UI/Labels:** DM Sans — same as body. Font-size 13-15px for UI elements.
- **Data/Machine/Votes:** JetBrains Mono — clinical. When you see this font you are looking at machine output or game state, not a person. Use for: timers, vote tallies, player IDs, role labels, status stamps, round counts.
- **Code:** JetBrains Mono
- **Loading:** Google Fonts CDN
  ```html
  <link href="https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=DM+Sans:ital,opsz,wght@0,9..40,300;0,9..40,400;0,9..40,500;0,9..40,600;1,9..40,300&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  ```
- **Scale:**
  - xs: 11px (labels, stamps, metadata)
  - sm: 13px (table content, secondary)
  - base: 15px (body text)
  - md: 20px (section headings)
  - lg: 28px (timers in JetBrains Mono)
  - xl: 40px (sub-hero)
  - 2xl: 52-64px (hero display, Instrument Serif)

## Color
- **Approach:** Restrained — one gold accent, warm neutrals. Color is rare and meaningful.

```css
--bg:            #0E0C09;   /* warm near-black — theater not void */
--surface:       #181410;   /* cards, panels */
--surface-high:  #221E17;   /* elevated elements, hover states */
--surface-border:#2E2820;   /* borders, dividers */
--accent:        #C4963A;   /* aged gold — candlelight, not neon */
--accent-dim:    rgba(196, 150, 58, 0.15);
--accent-glow:   rgba(196, 150, 58, 0.08);
--text:          #ECE7DE;   /* warm cream — aged paper, not white */
--text-muted:    #786F62;   /* footnotes, timestamps, labels */
--text-dim:      #4A4438;   /* disabled, placeholder */
--danger:        #8C1F1F;   /* deep crimson — mafia, eliminated */
--danger-dim:    rgba(140, 31, 31, 0.20);
--police:        #3D7FA8;   /* slate blue — the one cool tone, detective/info */
--police-dim:    rgba(61, 127, 168, 0.15);
```

- **Semantic:**
  - Mafia role: `--danger` (#8C1F1F)
  - Police/Detective role: `--police` (#3D7FA8)
  - Citizen role: `--text-muted` (#786F62)
  - Alive status: green #3A6A3A (dot only)
  - Eliminated: `--danger`
- **Dark mode:** This is a dark-only design. No light mode variant.
- **Texture:** Subtle paper grain overlay at 4% opacity on all surfaces via CSS background-image (SVG noise pattern). Applied as a fixed `::before` on `body`.

## Spacing
- **Base unit:** 8px
- **Density:** Comfortable
- **Scale:**
  - 2xs: 2px
  - xs: 4px
  - sm: 8px
  - md: 16px
  - lg: 24px
  - xl: 32px
  - 2xl: 48px
  - 3xl: 64px
  - 4xl: 80px

## Layout
- **Approach:** Grid-disciplined with asymmetric composition for game screens
- **Game room:** Three-column layout — player ledger (240px) / chat (flex 1) / vote+role panel (280px)
- **Lobby:** Two-column — create room sidebar (320px) / room list (flex 1)
- **Max content width:** 1200px
- **Border radius:** Minimal — never bubbly
  - sm: 2px (buttons, inputs, stamps)
  - md: 4px (cards, panels)
  - lg: 6px (containers, mocks)
  - No `rounded-full` except dots/indicators

## Motion
- **Approach:** Intentional — deliberate fades, no bounces or spring animations
- **Easing:** enter: ease-out · exit: ease-in · move: ease-in-out
- **Duration:**
  - micro: 50-100ms (hover states, focus rings)
  - short: 150-200ms (button presses, badge transitions)
  - medium: 250-350ms (panel reveals, phase transitions)
  - long: 400-600ms (result overlay, game-over screen)
- **Never:** keyframe bounces, scale-up entrance animations, particle effects, glow pulses

## Component Patterns

### Player Ledger (not avatar grid)
```
01  •  Alex
02  •  Morgan  [ELIMINATED]  ← rotated 6°, monospace, danger color
03  •  Jamie
```
- Numbered rows, monospace numbering
- Status dot (6px circle): alive = #3A6A3A, dead = var(--danger)
- Dead names: strikethrough with danger color, text-dim color
- "ELIMINATED" / "KILLED" stamp: JetBrains Mono, danger, rotated ±6°, thin border

### Vote Bars
- Thin 2px bars (not thick progress bars)
- Vote tallies in JetBrains Mono, text-muted

### Phase Header
- Phase name: Instrument Serif, 22px
- Round/detail: JetBrains Mono, text-muted
- Timer: JetBrains Mono, 28-32px, accent color

### Badges / Status Pills
- Background: dimmed semantic color (--accent-dim, --danger-dim, --police-dim)
- Text: semantic color
- Font: JetBrains Mono, 10-11px, uppercase, letter-spacing 0.1em
- Radius: sm (2px)

### AI / Human Tag (post-game reveal only)
- AI: accent text on accent-dim background
- Human: text-muted on surface-high background

## Decisions Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-03-31 | Initial design system created | /design-consultation based on competitive research and two independent design voices |
| 2026-03-31 | Chose Instrument Serif for display | Nobody in social deduction game category uses serif; reads as theatrical and sinister |
| 2026-03-31 | Chose warm amber gold #C4963A as accent | Complete departure from blue/purple/neon category conventions; aged gold fits dossier theme |
| 2026-03-31 | Player list as ledger, not avatar grid | Reinforces the "case file" product thesis; players feel like suspects in a file |
| 2026-03-31 | Dropped glassmorphism | Replace glow/blur with paper grain texture; more coherent with noir aesthetic |
| 2026-03-31 | Minimal border radius (2-6px) | Bubbly rounds contradict the industrial/serious tone |
| 2026-03-31 | AI identity revealed only post-game | Product design decision carried into UI: in-game players listed identically, AI/Human revealed in results like a court document |

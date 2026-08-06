---
version: alpha
name: Ratatoskr
description: Conservative local disk scanner with a sharp terminal-first visual system.
colors:
  primary: "#120F0C"
  on-primary: "#F3E9D7"
  secondary: "#CDBFA9"
  tertiary: "#C44724"
  on-tertiary: "#FFFFFF"
  neutral: "#17110E"
  surface: "#1F1712"
  surface-strong: "#2A1D16"
  border: "#3A2B22"
  muted: "#6D6258"
  safe: "#55B86B"
  safe-text: "#89DD98"
  cautious: "#E6A23C"
  cautious-text: "#F0BA63"
  dangerous: "#E24A2A"
  dangerous-text: "#E17762"
  report: "#B84A35"
  focus: "#FFB000"
typography:
  h1:
    fontFamily: Georgia, "Times New Roman", serif
    fontSize: 4.8rem
    fontWeight: 800
    lineHeight: 0.92
    letterSpacing: 0
  h2:
    fontFamily: Georgia, "Times New Roman", serif
    fontSize: 3rem
    fontWeight: 800
    lineHeight: 1
    letterSpacing: 0
  h3:
    fontFamily: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif
    fontSize: 1.1rem
    fontWeight: 900
    lineHeight: 1.25
    letterSpacing: 0
  body:
    fontFamily: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif
    fontSize: 1rem
    fontWeight: 500
    lineHeight: 1.65
    letterSpacing: 0
  label:
    fontFamily: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif
    fontSize: 0.76rem
    fontWeight: 900
    lineHeight: 1.2
    letterSpacing: 0.13em
  mono:
    fontFamily: '"JetBrains Mono", "Cascadia Code", "SFMono-Regular", Menlo, Consolas, monospace'
    fontSize: 0.92rem
    fontWeight: 700
    lineHeight: 1.65
    letterSpacing: 0
rounded:
  xs: 4px
  sm: 6px
  md: 8px
  lg: 10px
  pill: 999px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  xxl: 48px
components:
  page:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
  button-primary:
    backgroundColor: "{colors.tertiary}"
    textColor: "{colors.on-tertiary}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: 14px 18px
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.secondary}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: 14px 18px
  panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.lg}"
    padding: 24px
  panel-strong:
    backgroundColor: "{colors.surface-strong}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.lg}"
    padding: 24px
  app-shell:
    backgroundColor: "{colors.neutral}"
    textColor: "{colors.on-primary}"
  muted-copy:
    backgroundColor: "{colors.muted}"
    textColor: "{colors.on-primary}"
    typography: "{typography.body}"
  divider:
    backgroundColor: "{colors.border}"
    height: 1px
  focus-ring:
    backgroundColor: "{colors.focus}"
    size: 2px
  terminal:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.mono}"
    rounded: "{rounded.lg}"
    padding: 20px
  badge-safe:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.safe-text}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: 6px 10px
  badge-cautious:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.cautious-text}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: 6px 10px
  badge-dangerous:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.dangerous-text}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: 6px 10px
  badge-report:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.dangerous-text}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: 6px 10px
  risk-bar-safe:
    backgroundColor: "{colors.safe}"
    height: 4px
  risk-bar-cautious:
    backgroundColor: "{colors.cautious}"
    height: 4px
  risk-bar-dangerous:
    backgroundColor: "{colors.dangerous}"
    height: 4px
  risk-bar-report:
    backgroundColor: "{colors.report}"
    height: 4px
---

## Overview

Ratatoskr should look like a competent terminal tool that wandered out of the roots with evidence, not a lifestyle app trying to sell cleanliness as a personality.

The visual system is dark, warm, and tactical. Bark-black surfaces. Bone text. Rust for action. Amber for attention. Green only for genuinely safe findings. Red-orange for dangerous or report-only paths. The mythology is seasoning. The safety model is the meal.

Use the product shape as the design test: a user should understand risk, consequence, and next action before they notice the squirrel.

## Colors

- **Primary (`#120F0C`)** is the root background. Use it for full-page dark mode, terminal surfaces, and the deepest UI layer.
- **On-primary (`#F3E9D7`)** is bone text. Use it for headings and important body copy.
- **Secondary (`#CDBFA9`)** is aged paper. Use it for body copy, navigation, captions, and secondary controls.
- **Tertiary (`#C44724`)** is rust action. Use it for primary buttons and selected states.
- **Surface (`#1F1712`)** and **surface-strong (`#2A1D16`)** are bark layers. Use them for panels, cards, terminals, and grouped controls.
- **Border (`#3A2B22`)** is quiet structure. Use it when spacing alone is not enough.
- **Muted (`#6D6258`)** is branch metadata. Use it sparingly; it can get muddy fast.
- **Safe (`#55B86B`)** and **safe-text (`#89DD98`)** belong to narrow, cleanable findings only.
- **Cautious (`#E6A23C`)** and **cautious-text (`#F0BA63`)** mean inspect first.
- **Dangerous (`#E24A2A`)** and **report (`#B84A35`)** mean do not clean by default.
- **Focus (`#FFB000`)** is the eye glint. Use it for focus rings, terminal lights, and tiny points of attention.

Avoid letting every screen become brown and orange. Every risk view needs green, amber, and red-orange present together so the safety model reads before the mood does.

## Typography

Headlines use Georgia because Ratatoskr has folklore in the bones. The UI uses Inter or the system sans stack because the product still has work to do. Terminal output uses JetBrains Mono when available, then common monospace fallbacks.

Use big serif type for first-viewport positioning and major section headings only. Dense UI, cards, tables, rule lists, and command output should stay compact. No viewport-scaled type. No negative letter spacing.

Labels are uppercase, heavy, and small. Keep them short: `SAFE`, `REPORT-ONLY`, `RULE`, `CONSEQUENCE`. If a label needs a sentence, it is body copy wearing the wrong hat.

## Layout

Pages should feel like a CLI report made readable: structured, grouped, and honest about what matters.

Use a max content width around `1160px`. Prefer full-width dark bands with constrained inner content over floating section cards. Cards are for repeated items, risk summaries, terminals, and compact references. Do not put cards inside cards.

Hero layouts may use a real or generated Ratatoskr image, but the product claim must lead. First viewport signal: name, safety promise, first command, and risk contract.

Keep terminal blocks wide enough to read. Wrap cautiously. A command that looks mangled loses trust.

## Elevation & Depth

Depth should be low and smoky. Use shadow to separate a terminal or raised panel from the root background, not to make the interface feel fluffy.

Preferred shadow:

```css
box-shadow: 0 30px 90px rgba(0, 0, 0, 0.46);
```

Use inset glows only for terminal atmosphere and risk panels. Decorative glow should never overpower content or reduce contrast.

## Shapes

Default radius is `8px`. Compact controls can use `6px`. Dense panels and terminal blocks can use `10px`. Pills are for badges only.

Do not drift into soft SaaS bubbles. Ratatoskr is compact, sharp, and useful. The corners can breathe; they should not purr.

## Components

Primary buttons use rust backgrounds with white text. They are for real actions: install, run, inspect, download. Secondary buttons use bark surfaces with paper text and border structure.

Risk badges must encode the safety model consistently:

- `safe`: green text, only for generated waste matched by narrow rules.
- `cautious`: amber text, for rebuildable but review-worthy candidates.
- `dangerous`: red-orange text, for protected, unknown, or unsafe paths.
- `report-only`: red-brown text, for things Ratatoskr names but does not clean.

Terminals should show plausible commands and output. No fake magic. No command should imply deletion unless the surrounding copy makes the confirmation model explicit.

Navigation should be short and scannable. Avoid decorative menu labels that hide the real destination.

## Do's and Don'ts

Do:

- Lead with evidence, consequence, and next action.
- Keep copy direct and slightly dry.
- Show path, size, risk, rule, reason, and cleanability together when possible.
- Use mythology as a small accent, not as the operating model.
- Keep generated images specific: filesystem trees, terminal output, roots, branches, cautious scout energy.

Don't:

- Use one-click-cleaner language.
- Treat large, old, cache, or log as automatically disposable.
- Use vague AI-cleanup claims.
- Use bright consumer-cleaner blue, clinical white dashboards, or generic purple gradients.
- Hide uncertainty behind decorative confidence.
- Add mascot flourishes where the user needs risk clarity.

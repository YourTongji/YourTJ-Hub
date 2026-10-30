# Nickname gradient contrast floor relaxed to 3:1 on endpoint stops

## Status

Accepted
Class: feature

## Context and Problem Statement

Nickname styles paint the member name with a two-stop gradient. Definition validation originally
required 4.5:1 — the WCAG AA floor for ordinary text — against both supported theme backgrounds on
every palette colour, and additionally rejected the definition when any interpolated point of the
gradient fell below it. Several palettes that render legibly were therefore rejected, even though
decorated names are semibold display text at 13–24 px and the strict floor also forced
administrators to abandon whole hue families. [PR #1157](https://github.com/YourTongji/YourTJ-Hub/pull/1157)
introduced that rule while [#1156](https://github.com/YourTongji/YourTJ-Hub/issues/1156) acceptance
criterion 2 recorded ordinary-text 4.5:1 as the bar; this decision supersedes that bar for nickname
gradients and the criterion is read as 3:1 for them from now on.
[Issue #1312](https://github.com/YourTongji/YourTJ-Hub/issues/1312) asks for a rule that keeps the
gate meaningful without rejecting legible palettes.

## Decision Drivers

- Keep a real contrast gate instead of removing validation entirely.
- Match the typography the gate protects: semibold display text at list/profile sizes.
- Make the accepted set identical across the Go authority gate, the Web admin/presentation check and
  the Flutter renderer; drifting thresholds already caused Web and mobile to disagree.
- Keep the client fallback to ordinary text when the actual background is genuinely unreadable.
- Avoid a rule that depends on a sampling resolution nobody can state precisely.

## Considered Options

- Keep 4.5:1 and require higher-contrast palettes everywhere.
- Use 3:1 (the WCAG AA large-text floor) at a single global threshold, sampling only the gradient's
  endpoint stops.
- Keep 3:1 but add size-tiered thresholds or a solid-colour downgrade for small nickname sizes.

## Decision Outcome

Nickname gradients are validated at 3:1 against the supported light (`#FFFFFF`, `#F3F4F6`) and dark
(`#111827`, `#1F2937`) theme backgrounds. The same threshold applies to every nickname size; there is
no size tier and no solid-colour downgrade. Validation samples the first and last gradient stop only
and rejects the definition if any endpoint misses the floor on any supported background. The ramp
between the stops is not interpolated, because a sample count is an implementation detail that
neither administrators nor clients can predict, while the endpoints are exactly what the colour
picker shows. Because the catalog re-validates every stored definition on read, definitions that
the old floor kept quietly hidden start being served again without an administrator action. All
three implementations carry the same named constant:
`NicknameContrastMin` in `app/service/decorationservice/definitions.go`, `NICKNAME_CONTRAST_MIN` in
`resource/src/runtime/decoration-style.ts`, and `nicknameContrastMin` in
`apps/mobile/packages/forum_app/lib/src/widgets/user_decoration.dart`. The web admin console always
shows the measured lowest stop/background contrast below its live preview and keeps painting the
entered gradient there; a definition below the floor is flagged in that same spot and cannot be
saved.

Clients measure the background they actually paint on and render ordinary text when it cannot meet
3:1; Web keeps its forced-colour fallback. Strokes, glow and image-filled nickname text remain
out of scope. The user-facing contract is stated in
[profile decorations](../product/profile-decorations.md).

## Pros and Cons of the Options

- Keeping 4.5:1 needs no documentation or client change, but rejects legible display-size palettes,
  would have to re-word #1156 acceptance criterion 2, and treats semibold display text as body copy.
- A single 3:1 endpoint threshold states one rule that administrators can check by eye, keeps the
  Go gate, the admin preview and Flutter in agreement, and stops grading pixels the design never
  promised. It gives up contrast enforcement in the middle of the gradient, where an sRGB
  interpolation between two approved endpoints can dip slightly against the background, and it
  accepts palettes that would fail an ordinary-text audit.
- Size-tiered thresholds or a small-size solid-colour downgrade would protect list rows most
  strictly, but add rendering branches and per-surface thresholds across Web and Flutter, and give
  the same definition different appearances at different sizes.

## Links

- [Nickname gradient contrast issue](https://github.com/YourTongji/YourTJ-Hub/issues/1312)
- [Profile decorations implementation, including the accepted palettes](https://github.com/YourTongji/YourTJ-Hub/pull/1157)
- [WCAG 2.2 SC 1.4.3 Contrast (Minimum), large-text 3:1](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html)
- [Profile decorations product reference](../product/profile-decorations.md)

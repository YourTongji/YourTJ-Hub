# Local flutter_math_fork adaptation

Source: flutter_math_fork 0.7.4 from pub.dev, under the included Apache-2.0 license.
The bundled KaTeX fonts retain their MIT license in `lib/katex_fonts/LICENSE`.

The platform switches in selectable math, selection gestures and the editable
line accept additional platforms through their existing Material branches.
This allows the upstream formula renderer to compile with Flutter OH without
changing Android/iOS behavior, formula layout, fonts or the shared editor UI.

Remove this override when upstream accepts Flutter OH's additional platform.
The application markdown/rich-content tests cover the shared renderer; run them
with Flutter OH as well as stock Flutter when upgrading this copy.

# Avatar-frame artwork

This directory contains optional, same-origin transparent frame artwork. An asset file does not
create or enable a decoration option; administrators create definitions separately in
`/admin/decorations`. Files are embedded in the Go binary at build time.

Follow [the avatar-frame artwork generation spec](../../../../../docs/product/avatar-frame-generation.md)
for alpha, geometry, review and filename requirements. The current unconfigured sample file is
`tea-garden-v1.png` (1254 × 1254 RGBA); its original pixels and alpha channel are preserved.

Prompt used for that sample:

> Use case: stylized-concept. Asset type: transparent PNG circular avatar frame overlay for a university community forum. Generate ONE refined tea-garden themed circular avatar frame, no avatar portrait inside. Flat front view, centered and perfectly circular on a square canvas. A slender warm ivory porcelain ring decorated with a few delicate green tea leaves and fine muted gold accents, polished illustrated game cosmetic style with clear silhouettes readable at 40px avatar size. Keep decoration restrained, especially leave the lower right badge corner clear. Composition: frame outer diameter roughly 92% of canvas, inner opening roughly 80% of canvas, all art only in the narrow annulus. The complete inner circular opening and the entire exterior of the frame MUST be truly alpha transparent, not white, black, checkerboard, shadowed background or opaque fill. No opaque pixels anywhere in the central hole, no portrait, no text, no watermark, no backdrop, no detached floating decorations. Design usable on both light and dark UI backgrounds. Crisp smooth antialiased edges, all elements contained within canvas, no drop shadow extending into inner opening.

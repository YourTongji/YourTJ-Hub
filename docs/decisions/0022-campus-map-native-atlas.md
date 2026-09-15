# Native campus atlas with reusable geographic data

## Status

Accepted
Class: architecture

## Context and Problem Statement

The campus map needs clear building-level discovery and finer sports-facility
information within YourTJ Hub. YourTJ Pulse offers useful OSM geometry and a map
renderer, but its React/Cloudflare collaboration application and separate 3D town
experience do not fit the forum's Vue and single-binary deployment boundary.

## Decision Drivers

- Preserve real building and path geometry while making the interface visually coherent.
- Keep anonymous building and sports-place discovery simple.
- Keep the existing Go + Vue deployment and identity boundaries.
- Make source attribution, data limitations and updates explicit.

## Considered Options

- Embed or fork the complete YourTJ Pulse application.
- Draw an independent illustrated SVG campus map and maintain custom coordinates.
- Reuse the OSM geometry and MapLibre renderer in a native Hub page.

## Decision Outcome

Use a native Vue campus atlas at `/map`, with MapLibre and a versioned, compact
OSM-derived GeoJSON snapshot. Import useful geometry and tags from a pinned Pulse
snapshot; implement the page layout, local map style, DOM labels, catalog and place
selection in Hub. Preserve original OSM IDs for place links and keep the derived
database under ODbL with visible attribution.

The page's assets and CSP worker are served by the existing forum binary. It has no
new API, persistence service or identity boundary. Geographic coverage and uncalibrated official-plan views are listed in the product
contract; official indoor-sports detail remains incomplete. Pseudo-height is illustrative and is not a measured building model.

## Pros and Cons of the Options

### Complete Pulse application

Existing collaboration functions are reusable, but its separate React and
Cloudflare stack expands deployment and identity responsibilities beyond the
campus-finding requirement. Its visual shell still needs substantial redesign.

### Independent illustrated SVG

Offers precise art direction and easy control over a bounded image coordinate
system. Re-drawing every footprint duplicates useful geographic work and makes
subsequent map corrections more expensive.

### Native Hub atlas with geographic data

Retains real geometry and efficient rendering while fitting Hub navigation and
deployment. Styling is controlled locally without a tile-provider token. It still
requires explicit work to verify official venue names and indoor room locations;
an OSM polygon alone is insufficient evidence for those details.

## Links

- [Campus map product contract](../product/campus-map.md)
- [Pinned data provenance and import](../../apps/gooseforum/resource/src/site/campus-map/data/README.md)
- [YourTJ Pulse source snapshot](https://github.com/WALKERKILLER/YourTJ-Pulse/tree/4a8d2925dd2b7a5d654a1c2eda43a3adb5fde45a)
- [OpenStreetMap copyright and database license](https://www.openstreetmap.org/copyright)

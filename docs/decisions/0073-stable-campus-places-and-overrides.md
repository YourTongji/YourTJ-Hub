# Stable campus places and explicit location overrides

## Status

Accepted
Class: architecture

## Context and Problem Statement

Course location strings mix building names, shorthand, room continuations,
conditional alternatives and faculty-dependent descriptions. A semester-scoped
exact dictionary preserves offline corrections but unnecessarily makes stable
buildings and new simple rooms depend on re-authorizing every semester's inputs.
Unrestricted parsing also loses suffix conditions and can incorrectly inherit
another building's letter prefix. Offline extraction is useful maintenance evidence,
not automatically verified building identity.

## Decision Drivers

- Reuse stable buildings across semesters, including when the term is unavailable.
- Preserve original text, rooms/floors and condition scope without guessing coverage.
- Give explicit corrections and safety blocks priority over generic parsing.
- Separate name recognition, complete extraction and map coverage measurements.
- Keep JSON/Schema/PR maintenance, no admin editor, browser model calls or new API.
- Retain the embedded frontend and single-binary deployment.

## Considered Options

- Require a complete exact dictionary for every semester.
- Restore unrestricted heuristics and infer building/condition continuations.
- Use a verified stable catalog, conservative simple parsing and prioritized overrides.
- Send unseen private course text to a model at runtime.

## Decision Outcome

Use a campus-scoped JSON catalog with independently evidenced names, aliases and
feature IDs. Source map names and previously confirmed project mappings can enter
the catalog; the complete extracted `place` set is only a candidate audit source.
In Siping, confirmed 南/北 plus numeric room text identifies the south/north teaching
building; the shorthand alone is insufficient. Room digits never imply floor data.

Match exact original text against explicit JSON overrides before parsing. A matching
term-scoped rule beats a campus-wide rule; the scope requires matching provided
calendar identities. Only semester-dependent interpretations, such as faculty-derived
names, require that scope. A block or extraction concern is final and cannot fall
through to general parsing. Duplicate or overlapping scopes fail validation.

On a miss, parse anchored longest verified names and simple room/floor formats.
Resolve a member's own verified letter alias before considering numeric continuation.
Unknown names/letters, duplicate map identities, missing features and conflicting
campuses remain unconfirmed. Multiple members require explicit user selection.

Keep time conditions independent of building recognition. Single-member parity,
week-range and weekday conditions can be evaluated. Dates, unsupported notes and
unclear continued-list scope cannot confirm a place-scoped schedule. Complex text
uses retained explicit overrides; all original members and pending state remain
visible. Source dictionary assets remain maintenance/regression inputs, not runtime
semester prerequisites. Preparation never promotes candidate review or adds aliases.

JSON Schema and tests validate shape, exact source fragments, destinations, scope
priority and no-fallback blocks. Place-text audits do not use map identities; runtime
audits report extraction/member/map changes separately for identical source keys.
Neither candidate-name hits nor unchanged map coverage prove semantic accuracy.

## Pros and Cons of the Options

- Whole-semester dictionaries preserve arbitrary exceptions, but block unchanged
  buildings and unseen simple rooms when a term or exact key is missing.
- Broad heuristics increase apparent coverage while making unsupported condition
  scope and cross-building inheritance difficult to correct reliably.
- Stable catalog plus overrides preserves corrections and supports a small safe
  grammar; maintaining complex exceptions and manually verifying names remain necessary.
- Runtime model calls can process unseen formats, but introduce privacy, latency,
  cost and external availability into private timetable viewing.

## Links

- [Rejected semester-dictionary proposal](0072-offline-campus-location-dictionaries.md)
- [Campus-map product contract](../product/campus-map.md)
- [Catalog, override Schema and maintenance](../../apps/gooseforum/resource/scripts/campus-locations/README.md)
- [Source map data and attribution](../../apps/gooseforum/resource/src/site/campus-map/data/README.md)
- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)
- [Map deployment boundary](0022-campus-map-native-atlas.md)

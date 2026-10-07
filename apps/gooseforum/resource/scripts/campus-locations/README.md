# Course location matching and offline extraction

> Doc type: reference
>
> Status: Active
>
> Last verified: 2026-10-05

`Partial`: the runtime uses a cross-semester verified-name catalog, conservative
building/room parsing and exact-text exception overrides. The source extraction
and migrated overrides remain pending semantic review; Schema validation is not
approval. No runtime or local maintenance command calls a model.

## Owned artifacts

| Artifact | Responsibility |
|---|---|
| [Place catalog](../../src/site/campus-map/data/locations/places.json) | Campus-scoped names, verified aliases, map identities and evidence; independent of course semesters. |
| [Override configuration](../../src/site/campus-map/data/locations/overrides.json) | Exact original text, replacement members/conditions or blocking status, optional term scope and provenance. |
| [Runtime schema](runtime.schema.json) and [validator](runtime-config.mjs) | Catalog/override shape, source snippets, scope conflicts and destination existence. |
| [Conservative parser](../../src/site/campus-map/deterministic-location.ts) | Anchored longest-name matching, simple rooms/floors and numeric same-building continuation; not general natural-language interpretation. |
| [System prompt](prompt.system.zh.txt), [examples](prompt.fewshot.messages.json) and [result schema](result.schema.json) | Offline source extraction with twelve v2 examples; `time` is a nonempty array or JSON `null`. |
| [TypeScript types](../../src/site/campus-map/location-types.ts) | Extraction, runtime configuration and presentation contracts. |
| [Source dictionary](../../src/site/campus-map/data/locations/2026-2027-1.json) | Public calendar-122 candidate, source fingerprints and fixed regression inputs; not imported by the runtime. |
| [Dictionary tool](dictionary.mjs) | Frozen request preparation, result assembly and conflict-preserving same-semester merge. |
| [Migration tool](migrate.mjs) | Prepares pending exception candidates; never adds building aliases or publishes configuration. |
| [Place-text audit](audit-place-text.mjs) | Candidate-name recognition from the complete same-campus `place` set, without any map or override lookup. |
| [Runtime audit](audit.mjs) | Member/map coverage and comparison for the exact same public source keys. |

## Lookup priority and scope

1. A matching term-scoped override takes precedence over a campus-wide override.
   A scope requires a matching calendar ID or registered school-calendar name;
   when both are supplied, both must agree. Unknown terms do not use that scope.
2. A campus-wide override matches campus identity plus exact original text.
   `action: block` or extraction concerns are final and prohibit parser fallback.
3. An override miss uses the stable catalog/parser. Missing term information does
   not prevent stable building identification or new simple classroom numbers.

Names and aliases are campus-specific. Stable catalog entries come from verified
GeoJSON names/aliases or explicitly documented project mappings, not from automatic
promotion of model-extracted `place` values. The catalog includes source-confirmed
Jiading letter aliases and existing Siping north/south teaching-building mappings.
**In Siping, 南/北 followed by a numeric room identifies 南教学楼/北教学楼**;
the numeric room is retained. A bare direction or `北二楼` is not that shorthand.

The parser matches known names at the beginning of a member, after recognized
time prefixes, rather than searching for a known building inside an unknown name.
An independently recognized letter alias wins over room inheritance: `A101、B201`
is not two rooms in 安楼. Only plain numeric continuations can inherit a building;
unknown letters reset inheritance. Multiple rooms still need explicit destination
selection, even in one building. Room numbers do not imply a floor; explicit floor
text remains detail rather than an indoor coordinate.

Known single-member prefix/suffix week and weekday conditions remain separate from
building identification. Date notes, unsupported notes, unclear condition scope
and ambiguous continued lists cannot confirm a place-scoped schedule. Complex
conditions use explicit overrides. All original text stays visible.

## Source and review boundaries

The public source dictionary is a deduplicated undergraduate PK snapshot for
calendar 122, synchronized on 2026-09-24: 702 inputs, 779 members and 89 flagged
results. It contains no student identities or private timetables. `_meta` retains
source, model/candidate and prompt fingerprints; the maintained v2 prompt is not
claimed to be the historical model-run prompt.

The runtime catalog has 37 entries. Exact-equivalent conservative parsing covers
332 source inputs; 370 exception rules retain the remaining extracted semantics.
All 370 migrated rules retain their pending state; 89 are blocking rules, and 39
faculty/context-derived rules have explicit term scope. This is not semantic
accuracy or a claim that all unflagged inputs have been manually reviewed.
Explicit complex text and safety blocks are campus-wide; faculty-derived names
are restricted to the source term. Removing term scope from the entire candidate
is not a substitute for separating stable identities and exceptions.

Source candidates remain available for future extraction/review and regression
comparison. New `place` values are candidates only. Add a stable alias only with
independent building-identity evidence; otherwise retain an exception or an
unconfirmed original. Raw snapshots, provider requests/responses, credentials,
reports and screenshots belong in ignored `research/`.

## Local commands

Run from `apps/gooseforum/resource` with the locked dependencies installed.
[JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12) is validated by
[Ajv 2020](https://ajv.js.org/json-schema.html#draft-2020-12).

```bash
# Validate runtime configuration and the independent offline source/examples.
node scripts/campus-locations/runtime-config.mjs
node scripts/campus-locations/dictionary.mjs check

# Audit candidate-name recognition, not map coverage or full semantic parsing.
node scripts/campus-locations/audit-place-text.mjs ../../../research/place-text.json

# Record runtime member/map coverage; compare only identical source inputs.
node scripts/campus-locations/audit.mjs ../../../research/location-before.json
node scripts/campus-locations/audit.mjs ../../../research/location-after.json ../../../research/location-before.json

# Prepare exception candidates against the current stable parser, never auto-publish.
node scripts/campus-locations/migrate.mjs src/site/campus-map/data/locations/2026-2027-1.json ../../../research/candidate-overrides.json

# Existing offline extraction workflow (request materials only, no provider call).
node scripts/campus-locations/dictionary.mjs prepare path/to/inputs.json path/to/new-requests 122
node scripts/campus-locations/dictionary.mjs assemble path/to/manifest.json path/to/results.json path/to/metadata.json path/to/new-candidate.json
node scripts/campus-locations/dictionary.mjs merge path/to/current.json path/to/candidate.json path/to/new-merged.json
```

Outputs refuse to overwrite existing files. Input preparation accepts only
`{ course_campus, raw }` records; it deduplicates exact source keys, freezes input,
prompt, example and Schema fingerprints and prepares batches of at most 20.
Assembly requires exactly the frozen response IDs and unchanged source fragments.
Same-semester merge preserves identical values, rejects conflicts/cross-term
inputs and retains parent provenance; neither assembly nor merge promotes review
status. Review all members, not only flagged rows.

The place-text audit uses longest non-overlapping literal candidate-name matches
and the confirmed Siping numeric north/south shorthand. An input with one name hit
may still have missed members, ambiguous conditions or unverified extracted names;
its hit count must never be reported as complete parsing accuracy or map coverage.
The runtime audit has separate member/target coverage and retains source hashes.

## Maintaining and publishing configuration

Edit catalog identities with evidence. Review candidate overrides against their
original text: choose `replace` or `block`, retain member kinds/conditions, explain
corrections and add term scope only for genuinely semester-dependent interpretation.
Duplicate/overlapping scopes are rejected. A model rerun cannot overwrite a human
correction or silently clear concerns. There is no admin editor or automatic
activation of an extraction asset.

Run runtime/dictionary validation, location and panel regression tests, typecheck,
i18n checks, production build and desktop/375px browser checks. Publish through
ordinary PR review. The [product contract](../../../../../docs/product/campus-map.md)
owns destination selection and schedule filtering.

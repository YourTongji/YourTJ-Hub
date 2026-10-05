# Offline course locations

> Doc type: reference
>
> Status: Active
>
> Last verified: 2026-10-05

`Current`: the map consumes semester-scoped offline extraction assets. The bundled
2026–2027 first-semester dictionary has `review_status: review_2_pending`: its
complete semantic review remains `Partial`. Schema validation does not approve it.

## Owned artifacts

| Artifact | Responsibility |
|---|---|
| [System prompt](prompt.system.zh.txt) | Source-text extraction rules, including the explicitly confirmed Siping north/south shorthand. |
| [Few-shot messages](prompt.fewshot.messages.json) | Twelve user/assistant examples in the v2 result format. |
| [Result schema](result.schema.json) | Required fields, member kinds and relationships; `time` is a nonempty array or JSON `null`. |
| [TypeScript types](../../src/site/campus-map/location-types.ts) | The same extraction contract and map presentation hints. |
| [Semester dictionary](../../src/site/campus-map/data/locations/2026-2027-1.json) | Exact course-campus/raw-text keys, extracted members and source fingerprints. |
| [Local tool](dictionary.mjs) | Request preparation, result assembly, validation and conflict-preserving merge. |

The dictionary derives from a deduplicated undergraduate PK course-location
snapshot for calendar 122, synchronized on 2026-09-24. It contains no student
identity or private timetable. Source, original v1 prompt, model run and corrected
candidate SHA-256 identities are recorded in `_meta`; the candidate contains model
output plus explicit user corrections. The maintained v2 prompt adds `time` and
the confirmed rules; it was not the prompt used for that historical model run.
Raw snapshots, requests, responses, credentials and review working files stay in
ignored `research/`.

`named` means that the text names a place. It does not establish a trusted map
identity. The runtime independently matches extracted names to campus features;
the prompt never receives map IDs or coordinates. `generic`, `online`, `pending`,
`no_room` and `unknown` remain distinct. `needs_review` indicates an extraction
concern, whereas `_meta.review_status` describes the whole asset's review state.
`needs_review: false` does not mean a person approved that row.

## Local commands

Run from `apps/gooseforum/resource` after installing the locked development
dependencies. [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12) is
validated by [Ajv's 2020 implementation](https://ajv.js.org/json-schema.html#draft-2020-12).
Neither these commands nor map browsing calls a model or reads credentials.

```bash
# Validate the shipped dictionary and prompt examples.
node scripts/campus-locations/dictionary.mjs check

# Validate another complete dictionary before editing an asset.
node scripts/campus-locations/dictionary.mjs check path/to/dictionary.json

# Freeze source keys and prepare batches of at most 20 inputs.
node scripts/campus-locations/dictionary.mjs prepare path/to/inputs.json path/to/new-requests 122

# Assemble an ID-keyed response into a campus/raw dictionary.
node scripts/campus-locations/dictionary.mjs assemble path/to/manifest.json path/to/results.json path/to/metadata.json path/to/new-candidate.json

# Add new keys to the same semester; conflicts require explicit human editing.
node scripts/campus-locations/dictionary.mjs merge path/to/current.json path/to/candidate.json path/to/new-merged.json
```

Preparation accepts only a nonempty array of `{ "course_campus": "四平路校区",
"raw": "北115" }` objects. Extra fields are rejected, exact repeated keys are
deduplicated, and source keys are not normalized. IDs bind to the calendar and the
two source strings. The manifest freezes inputs, prompt, examples and schema
fingerprints. Request files contain `messages` and a local `result_schema`; configure
the chosen provider separately, and do not assume it accepts that schema as an API
parameter. JSON-only prompting still needs local validation.

Assembly accepts the frozen manifest, a JSON object containing exactly its IDs,
and a metadata object with the fields described by `LocationDictionarySource`.
Set the actual calendar ID, term and verified school-calendar display names,
source audience/date and requested model. Assembly sets the input count,
fingerprints, correction rule and pending review state. It rejects missing/extra
IDs, changed source identities, malformed members and rewritten conditions.
Review all members, not only flagged rows, before declaring a dictionary reviewed.

Merge requires identical schema, semester identity, display-name aliases and
audience. Identical existing values are retained; a conflicting value stops the
merge instead of overwriting a human correction. Merged metadata retains parent
sources and composite fingerprints, returns to pending review, and uses the
oldest source sync date so a newer batch does not make older records appear fresh.
Commands refuse to overwrite output files and existing request directories.

## Production incorporation

Review the candidate and its source metadata, then copy the validated result into
the semester asset and inspect the diff. Explicitly edit conflicting existing
values with their review evidence; a model rerun does not authorize those edits.
Update the runtime's registered semester asset when adding another semester.
Run the location tests, typecheck, i18n gate, production build and relevant browser
checks. Publication follows the repository's ordinary PR process; reviewing a
candidate alone does not publish it.

The runtime requires a matching PK calendar ID or an explicitly registered school
calendar name. The two source keys must match exactly. Uncovered, changed or
different-semester text remains visible as an unlocated original string. There is
no heuristic fallback or browser model call. Flagged entries have no map target;
the asset-wide pending-review notice remains visible for other extracted entries.
The [product contract](../../../../../docs/product/campus-map.md) owns destination
choices and schedule filtering.

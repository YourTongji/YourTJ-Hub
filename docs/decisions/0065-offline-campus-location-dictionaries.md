# Semester-scoped offline campus location dictionaries

## Status

Proposed
Class: architecture

## Context and Problem Statement

Official course locations contain room lists, institutional qualifiers, remote
classes and temporal alternatives. Runtime heuristics combine text extraction and
map matching, making manual corrections difficult to preserve. A finite offline
extraction already exists, with explicit user corrections and unresolved concerns.
The user requires retaining its pending-review state and replacing the previous
parser without falling back to it.

## Decision Drivers

- Preserve original source text, member relationships and assigned conditions.
- Retain manual corrections and make model/schema provenance inspectable.
- Keep model calls outside private timetable browsing and production requests.
- Preserve the embedded frontend and single-binary deployment.
- Make uncovered inputs visible without guessing a building or reusing another term.

## Considered Options

- Continue extending runtime parsing rules.
- Call a language model during map browsing.
- Consume a semester-scoped offline dictionary and independently resolve map IDs.
- Prefer the dictionary but fall back to the previous parser on a miss.

## Decision Outcome

Consume versioned JSON assets keyed by semester, exact course campus and exact
location text. PK queries provide their calendar ID; private timetables provide
the existing school-calendar name, matched only against registered names. Unknown
semester identities and uncovered text remain unlocated originals.

Extraction preserves kind, relation, detail, campus text, address, `time`, other
conditions and unresolved condition scope. `time` is JSON null when absent. Whole
asset review state and per-result concerns are distinct. Pending review remains
visible, and flagged results have no target. Nonphysical records cannot create
targets. Independently maintained campus identities and unique feature-name
matches resolve physical destinations; no model output supplies map IDs.

Known per-member week and weekday conditions can constrain scoped schedules.
Alternatives, review concerns, unresolved scope and unsupported conditions cannot
confirm an arrangement. Multiple members require an explicit destination choice.

Local tooling prepares frozen bounded prompts, validates schema/source text,
assembles exact result IDs and merges only new same-semester keys. Conflicting
values require explicit review instead of replacement; assembly and merge never
promote an output to reviewed. Assets, prompt, examples, schema and TypeScript
types travel in the same change. Research inputs and model-call records remain
outside Git. Runtime and maintenance tools make no network model calls.

## Pros and Cons of the Options

- Runtime rules are cheap to execute, but each ambiguous source format requires
  more inference and has no durable per-input correction.
- Online model calls can process unseen text, but introduce privacy, latency,
  cost and provider availability into timetable browsing.
- Offline dictionaries preserve reviewed corrections and reproducible inputs;
  they require explicit updates when a term or source text changes and do not
  themselves establish the correctness of extraction or map identity.
- A heuristic fallback increases coverage but reintroduces the interpretation
  that this change replaces and hides missing dictionary coverage.

## Links

- [Campus-map product contract](../product/campus-map.md)
- [Maintained dictionary, prompts and tooling](../../apps/gooseforum/resource/scripts/campus-locations/README.md)
- [Map deployment boundary](0022-campus-map-native-atlas.md)
- [JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)
- [Ajv Draft 2020-12 support](https://ajv.js.org/json-schema.html#draft-2020-12)

# Selected Dart wire-model generation

## Status

Accepted
Class: architecture

## Context and Problem Statement

The controlled OpenAPI contract generates Web types, but mobile mirrors can drift when schema
changes are copied manually. Existing Dart models also contain tolerant decoding and custom
helpers, so replacing them all would change client behavior beyond contract ownership.

## Decision Drivers

- Derive selected field types and requiredness from the controlled schema.
- Preserve current class names, import paths, null omission, and strict response casts.
- Make stale generated output fail CI, including changes only to the mobile artifact.
- Retain explicit ownership for custom parsing and models outside supported generation semantics.

## Considered Options

- Generate an explicit subset of simple wire models with a small repository-owned renderer.
- Replace the entire mobile API layer with a general OpenAPI client generator.
- Keep all Dart mirrors manual and rely on fixtures alone.

## Decision Outcome

Use `dart-models.json` as an explicit selection registry. The renderer reads field definitions
from `components/schemas.yaml` and generates `openapi_models.dart`. Activation status and push
register/unregister inputs are selected; their original files re-export the classes. The contract
check compares the artifact without modifying it, and mobile core `gen/` edits select that CI job.

Supported schemas are closed, flat objects with string/integer/boolean fields. Unsupported selected
schema keywords fail generation. Optional fields use nullable types and omit null request values.
Enums remain strings; validation constraints stay server-side. This is `Partial` model generation,
not transport generation or a replacement for unlisted compatibility parsers.

## Pros and Cons of the Options

### Selected wire models

- Removes duplicated field definitions while preserving current client call sites.
- Runs inside the existing Node contract toolchain without another runtime dependency.
- Requires maintaining the small renderer and explicitly handling unsupported schema evolution.

### General client generator

- Can provide broader schema and transport coverage.
- Requires evaluating generated API conventions against existing Dio authentication, envelope
  handling, and custom parsing; that migration is larger than the selected DTO ownership change.

### Manual mirrors and fixtures

- Preserves all current behavior without new tooling.
- Fixtures only exercise their samples and do not ensure every selected field or requiredness
  change is reflected in the mobile declaration.

## Links

- [Model registry](../../packages/api-contract/dart-models.json)
- [Generator and ownership guide](../../packages/api-contract/README.md)
- [Contract status](../architecture/contracts-and-data.md#contract-status)

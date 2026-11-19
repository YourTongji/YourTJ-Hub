# API contract tooling

> Doc type: development guide
>
> Status: Active (implementation `Partial`)
>
> Owner: Platform maintainers
>
> Last verified: 2026-11-12

`openapi.yaml` is the controlled HTTP contract entry. The
[contract architecture](../../docs/architecture/contracts-and-data.md#contract-status) owns coverage
semantics; [route coverage](coverage-matrix.md) explains the generated operation inventory.

## Commands

Run from this directory after `pnpm install --frozen-lockfile`:

```sh
pnpm check          # tests, lint, bundle, fixtures, TS generation, route coverage, Dart freshness
pnpm generate:dart  # update the committed selected Dart wire models
pnpm check:dart     # compare the Dart artifact without rewriting it
```

## Dart ownership

`Partial`: [dart-models.json](dart-models.json) selects the models, their existing public class names,
and serialization direction. Field names, wire types, and requiredness come exclusively from
[components/schemas.yaml](components/schemas.yaml). The output is
[openapi_models.dart](../../apps/mobile/packages/core/lib/src/gen/openapi_models.dart).

| Schema | Dart class | Direction |
| --- | --- | --- |
| `ActivationStatus` | `ActivationStatus` | Response decoding |
| `PushDeviceRegisterRequest` | `RegisterPushDeviceInput` | Request encoding |
| `PushDeviceUnregisterRequest` | `UnregisterPushDeviceInput` | Request encoding |

The existing `activation_status.dart` and `push_device.dart` imports re-export these classes, so
repositories and `core.dart` consumers keep their public API. Other models remain hand-maintained.
Do not add another handwritten definition for a selected class.

The generator accepts closed, flat objects containing `string`, `integer`, or `boolean` fields.
Required fields retain strict casts; optional fields use nullable Dart types, and null optional
request fields are omitted. Enums remain strings to preserve the existing API; enum/range/length
validation remains the server's responsibility. Formats, defaults, null unions, references,
composition, arrays, and nested objects are unsupported and fail for selected schemas. A schema
change requiring one of those semantics must first define and test its client behavior; it must
not be silently approximated by the generator.

Adding a selected model requires a registry entry, removal or replacement of its manual definition,
and tests through the existing client import. Keep custom getters and tolerant compatibility parsers
outside the generated artifact. The generator introduces no Dart runtime dependency and does not
generate HTTP transport, authentication, response envelopes, or application state.

`pnpm check` rejects missing/stale artifacts. CI also selects the contract job for edits under mobile
core `lib/src/gen/`, so editing only the generated file does not bypass freshness verification.
Flutter tests consume the shared activation fixture and verify push request omission behavior;
Node tests verify schema evolution, unsupported shapes, and non-mutating freshness checks.

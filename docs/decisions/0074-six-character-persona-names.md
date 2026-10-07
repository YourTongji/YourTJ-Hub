# Six-character freely combined persona names

## Status

Accepted
Class: feature

## Context and Problem Statement

The full THUOCL pool includes terms that make poor personal aliases and vary greatly in
length. The user requests six-character phrases such as 躲进云里的猫 and 抱着松果的熊,
and explicitly welcomes unusual combinations rather than grammatical filtering.
This replaces the naming choice in [0066](0066-persistent-anonymous-forum-persona.md),
while carrying forward its public/private persona boundary, one-account slot, quota,
yearly lock, governance, audited reveal and indefinite restricted retention.

## Decision Drivers

- Exactly six Han characters in every newly generated candidate.
- Large variety from independent reusable word components.
- Playful combinations without a semantic filtering service.
- Reproducible source in the existing single binary with no runtime dependency.
- Preserve confirmed identities and persisted candidates across source updates.

## Considered Options

- Continue sampling the complete THUOCL word pool.
- Maintain a large static list of complete six-character names.
- Combine action, scene/object and animal components freely.
- Restrict component combinations by grammar or semantic compatibility.

## Decision Outcome

Chosen: versioned repository-authored component lists (`phrase6-v1`), independently and
uniformly combined as two-character action + two-character scene/object + 的 +
one-character animal. The source has 36 actions, 163 scenes/objects and 32 animals,
producing 187,776 combinations. Ten unique random indices form a batch using
cryptographic randomness. Different batches and personas may share names.
No semantic filter rejects a combination such as 躲进松果的猫.

Confirmed names are not rewritten. Persisted batches retain their exact words until their
existing expiry, and confirmation still selects the owner-persisted batch/index. This avoids
invalidating ambiguous retries or bypassing the yearly lock. Name payloads remain strings
because previously selected names can have different lengths or scripts.

## Pros and Cons of the Options

- Full THUOCL: broad coverage but cannot guarantee the requested phrase shape or length.
- Complete name list: predictable results but repeats manual phrase authoring and offers
  less variety for the same source size.
- Free components: compact source and diverse playful names; deliberately accepts unusual
  imagery and does not promise grammatical or semantic consistency.
- Compatibility filtering: can make phrases more conventional but excludes the playful
  combinations the user explicitly wants and adds maintenance or runtime dependencies.

## Links

- [Component lists](../../apps/gooseforum/app/bundles/anonymousnames/data/phrases.json).
- [Generator and source rules](../../apps/gooseforum/app/bundles/anonymousnames/README.md).
- [Product behavior](../product/anonymous-identity.md).
- [0066: superseded naming choice and retained persona boundaries](0066-persistent-anonymous-forum-persona.md).

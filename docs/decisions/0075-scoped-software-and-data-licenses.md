# Scoped licenses for software and community data

## Status

Accepted
Class: process

## Context and Problem Statement

The repository previously stated that it had no root-level license. Its only root-visible software
license was the MIT license retained by the GooseForum fork. YourTJ software, Wiki pages and course
reviews therefore had no clear repository-level license notice. Wiki content is stored in the
separate `YourTJ-Wiki` repository, and review data is stored by the running service.

## Decision Drivers

- Preserve the MIT grant for code derived from the GooseForum upstream.
- Give YourTJ-originated software a standard software license.
- License Wiki and course-review data separately from software.
- Preserve third-party notices and authors’ rights in content outside the selected scope.
- Make historical scope and redistribution obligations discoverable.

## Considered Options

- Keep only the GooseForum MIT notice and leave the rest without a public grant.
- Apply one license to the entire repository and all content.
- Use scoped licenses for software, upstream code, data and third-party material.

## Decision Outcome

Use GPL-3.0-only for YourTJ-originated software and repository documentation, retain MIT for
GooseForum upstream material, and use CC BY-NC-SA 4.0 for authorized Wiki and course-review data.
The grant covers the historical corpus effective 2026-08-06, the Hub repository initialization date.
The rights holders authorized this historical scope in writing.

Keep CC-licensed data separate from GPL software releases. Preserve third-party license and
attribution notices. Other user content remains under its authors’ rights unless an explicit grant
states otherwise. The detailed scope and release obligations live in
[licensing.md](../development/licensing.md).

## Pros and Cons of the Options

- **Keep only the GooseForum MIT notice:** avoids choosing a new license, but leaves first-party
  software and project datasets without a clear public grant.
- **Apply one license to everything:** creates a simpler notice, but would misstate third-party
  rights and conflate code with noncommercial data.
- **Use scoped licenses:** matches the source and type of each work and preserves upstream rights;
  requires clear separation, attribution and release checks for each license.

## Links

- [GNU license compatibility](https://www.gnu.org/licenses/license-compatibility.en.html).
- [CC BY-NC-SA 4.0 legal text](https://creativecommons.org/licenses/by-nc-sa/4.0/legalcode.en).
- [CC BY-NC-SA compatible licenses](https://creativecommons.org/compatible-licenses/).
- [Apple Developer Program License Agreement](https://developer.apple.com/support/terms/apple-developer-program-license-agreement/).
- [Repository licensing scope](../development/licensing.md).

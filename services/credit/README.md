# Cross-platform points settlement

`Planned`: this directory reserves the deployment home for [credit](https://github.com/linux-do/credit).
There is no deployed service or forum integration here. The forum-local reward ledger is implemented
inside `apps/gooseforum` and is distinct from this planned settlement service.

The target integration uses the forum's built-in OIDC Provider (numeric `sub` = `users.id`) and the
credit merchant distribution API. Product constraints, current ledger limits and integration assumptions
are maintained in the [points specification](../../docs/product/credit-and-escrow.md).

# Release intent examples

| Request | Action |
| --- | --- |
| “看看 Android 这次更新了什么” | Inspect source/evidence and draft Android prose; no PR, tag or release mutation |
| “准备双端 patch，iOS 先 TestFlight” | `prepare --scope mobile --bump patch --ios-destination testflight --apply` |
| “只发 Android” | Scope android; iOS notes and Apple operations are absent |
| “main 又进了代码” | Keep the candidate's sourceSha; including new source requires a new/revised reviewed request |
| “把这个 TestFlight 版本上架” | `promote-ios --release <original-id> --to app-store --apply`; new store notes/review, same Apple build |
| “帮我 approve 然后发掉” | Prepare/validate all concrete material; report that the final-head human review is still required |

A rejected store version needing only wording changes can use another existing-build promotion
request with fresh human review. A binary change requires a new source/version/build identity.
The CLI's JSON error is not permission to guess tags, bypass missing checks or retry indefinitely.

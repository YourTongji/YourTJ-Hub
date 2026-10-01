# Recovery choices

| Observed state | Next action |
| --- | --- |
| Oryn failed | Fill retained draft files from evidence; validate and obtain human review |
| No human review / stale approval | Finish review; do not attempt a publisher bypass |
| Some Android assets uploaded | Restore original signed APK artifact, verify identity/digests, upload only missing matching assets |
| Android available, alias failed | Retry `android-alias`; preserve immutable version assets |
| Android succeeded, iOS timed out | Retry only `ios-testflight` or the previously approved `ios-app-store` target |
| Apple build already processing | Query exact version/build; wait/reconcile; no duplicate upload |
| Apple build exists, artifact expired | Continue from Apple build without downloading/rebuilding IPA |
| Apple rejected text | Prepare a reviewed existing-build store request with corrected iOS prose |
| Apple rejected binary | New binary identity and release request |
| Production health/identity check failed | Inspect image/config rollback receipt and migration compatibility before retrying |
| API permission/network error | Report unknown and repair access; no synthetic “not found” fallback |

A retry is one bounded workflow run. If the same failure remains, surface the evidence and corrective
step; do not start repeated runs that consume signing/Apple resources without a changed condition.

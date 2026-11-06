# Platform release tooling

The public entry is [scripts/release/cli.py](../release/cli.py); the
[release runbook](../../docs/operations/releases.md) owns production RC, explicit Confirm and original-artifact recovery.
These helpers are invoked by trusted reusable platform publishers, with source under `RELEASE_SOURCE_ROOT`
when it differs from the controller checkout. They never select a version or promote dev.

- `prepare_push.py`: validate Android client-only push configuration before building.
- `build_ios.py`: validate signing/entitlements and produce the signed IPA with an ephemeral keychain.
- `install_asc.py`: install a checksum-verified pinned ASC CLI.
- `publish_android.py`: verify APK identity/signature/architecture and immutable GitHub assets. Requires
  `ANDROID_NOTES_PATH`; no iOS/store fallback. `--verify-only` performs local validation only.
- `publish_android_latest.py`: refresh the mutable download alias without regressing versions.
- `publish_ios.py`: query before upload, resume exact Apple builds, use explicit independent
  `IOS_TESTFLIGHT_NOTES_PATH` and `IOS_STORE_NOTES_PATH`; promotion can restrict execution to App Store.
- `download_harmony_cli.py`, `build_harmony.py`, `sign_harmony.py`: checksum-pin the official Linux SDK,
  build ARM64 Release, and sign/verify both HAP and APP without passwords in argv or logs.
- `harmony_api.py`, `publish_harmony.py`: fixed-application Service Account authentication, testing-only
  upload, legality polling, receipt-bound recovery and submission of an existing invitation draft.
  The separate main-only APP Test workflow does not enter the Android/iOS production release controller.

Run `python3 -m unittest discover -s scripts/mobile-release -p 'test_*.py'` without production keys.
On macOS the effective-signing test also requires `flutter pub get` in the selected source's forum_app.
Publisher-only recovery does not run native build verification. The workflow preserves IPA/dSYM/logs
before upload. Signing secrets and review credentials never enter Git, model input or public artifacts.

See [mobile signing/push operations](../../docs/operations/mobile-releases.md) for credentials and
physical-device acceptance. Successful script tests do not establish actual Apple review approval.

Production RC calls the platform builders once. Confirm promotes those original artifacts. RC has production configuration
and final version/build identity; it is not `MOBILE_PREVIEW=true`. The RC internal TestFlight
publisher is `publish_preview_ios.py` with `RC_BUILD=true`. An explicitly selected RC is restored
by the normal publisher without another build. Source tests stay in CI, not release jobs.

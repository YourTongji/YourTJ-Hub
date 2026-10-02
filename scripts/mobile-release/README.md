# Platform release tooling

The public entry is [scripts/release/cli.py](../release/cli.py); the
[release runbook](../../docs/operations/releases.md) owns preparation, final human approval and recovery.
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

Run `python3 -m unittest discover -s scripts/mobile-release -p 'test_*.py'` without production keys.
On macOS the effective-signing test also requires `flutter pub get` in the selected source's forum_app.
Publisher-only recovery does not run native build verification. The workflow preserves IPA/dSYM/logs
before upload. Signing secrets and review credentials never enter Git, model input or public artifacts.

See [mobile signing/push operations](../../docs/operations/mobile-releases.md) for credentials and
physical-device acceptance. Successful script tests do not establish actual Apple review approval.

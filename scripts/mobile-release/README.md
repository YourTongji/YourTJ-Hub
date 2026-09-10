# Mobile release tooling

These scripts run from the repository root. The [operations runbook](../../docs/operations/mobile-releases.md)
owns versioning, GitHub environment secrets, signing backup/rotation and recovery.

- `install_asc.py <destination>` pins the official CLI binary and verifies its SHA-256.
- `build_ios.py` validates a distribution profile and signs Runner with an ephemeral keychain.
- `publish_ios.py` resumes exact version/build uploads and submits both Apple distribution channels.
- `publish_android.py [--verify-only]` validates split APKs and publishes immutable-name assets.
- `prepare_mobile.py` prepares the production PR and reserves patch/minor/major version tags,
  or resumes an existing tag without consuming a build number.
- `prepare_server.py` opens the production PR or tags already reviewed main source; mobile tags
  never influence server version increments.

Run release-script regressions without signing credentials or external mutations:

```bash
python3 -m unittest discover -s scripts/mobile-release -p 'test_*.py'
actionlint .github/workflows/release-mobile.yml .github/workflows/release-to-main.yml .github/workflows/ci-mobile.yml
```

The publish commands mutate external services. `--verify-only` on the Android publisher writes local
checksums only. Build logs and outputs belong under ignored `apps/mobile/build/release/`; private inputs
must come from an ignored local file or the CI runner's private temporary directory. Tests never print
real review contacts, passwords or signing material.

Native push activation and client/provider credential separation are documented in
[mobile release operations](../../docs/operations/mobile-releases.md#native-push-activation-and-verification).
`prepare_push.py` validates Android client-only identifiers and selected OEM adapters before signing;
`build_ios.py` requires a production APNs entitlement in the distribution profile.

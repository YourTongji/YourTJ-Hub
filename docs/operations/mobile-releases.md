# Mobile releases

> Doc type: operations runbook
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-07

`Partial`: signed Android APK and iOS archive/export scripts, release validation and update-client
checks exist. GitHub-hosted signing/upload requires the release workflow on `main`, configured
`mobile-release` environment secrets, and a successful release run. Apple review and the Android
system installer remain independent gates; a completed upload does not mean distribution approval.

## Version and activation

1. Open GitHub Actions → **Release / mobile** → **Run workflow**, choose `patch`, `minor` or
   `major`, and leave the optional recovery tag empty. Both `dev` and `main` are accepted as the
   workflow branch. If dev and main differ, the workflow opens/reuses their release PR and stops.
   Merge it through normal review/CI, then rerun with the intended bump. A dev dispatch forwards
   to a main run once the trees agree, keeping signing secrets restricted to main.
2. The workflow increments the highest stable mobile release version (falling back to the pubspec
   floor) and increments the highest base build number independently. For example, from `1.2.3+8`:
   `patch` → `1.2.4+9`, `minor` → `1.3.0+9`, `major` → `2.0.0+9`. Server tags and prerelease names
   are excluded. The bootstrap floor is `1.0.0+1`, matching the existing Apple build; the first
   patch reservation is therefore `1.0.1+2`.
3. An annotated `mobile-vX.Y.Z` tag records `{schema: 1, version, buildNumber}` and points at the
   reviewed main commit. These immutable values are passed to Flutter's build flags for both
   platforms. No manual pubspec edit or tag push is needed. The source pubspec is a development
   floor, not the current distributed version. Raising it explicitly raises the next release floor.
4. The same workflow builds and publishes both channels. There is deliberately no tag-push trigger:
   automatic tag creation must not start a second upload. A single concurrency group serializes
   version reservation and distribution. A commit already carrying a mobile tag requires the
   recovery field instead of silently allocating another version.

```bash
# Normal release (also available as a dropdown in GitHub Actions):
gh workflow run release-mobile.yml --ref main -f bump=patch
# Resume the exact failed version; bump is ignored when tag is supplied:
gh workflow run release-mobile.yml --ref main -f bump=patch -f tag=mobile-v1.0.1
# Resume only an already uploaded iOS build; keep the existing App Store review queue:
gh workflow run release-mobile.yml --ref main -f tag=mobile-v1.0.2 -f recover_ios=true -f testflight_only=true
```

Recovery still checks that the tag's source is reachable from main, but does not require newer dev
changes to be promoted. Existing lightweight tags from the manual workflow are accepted only if
their source pubspec exactly matches their version. Annotated tags must carry valid release metadata;
never move or rewrite a release tag. An uncertain API response after reserving a tag should be
resolved by inspecting that tag and using recovery, rather than selecting another bump.

Update public store metadata/screenshots under `apps/mobile/store/` through normal product PRs.
A pending Apple version can block creation of the next App Store version. The default combined
submission reports the blocking version/state and fails, even if TestFlight succeeded. Select
`testflight_only=true` to explicitly omit App Store submission; the job summary identifies that
omission and preserves the existing review queue. It never withdraws another version automatically.
Apple agreements, review decisions and the system installer are not bypassed.

Server tags remain `vX.Y.Z`. **Release / main** opens/reuses the `dev` → `main` PR when the trees
differ and stops. After that PR passes checks and merges, rerun to tag the approved main commit,
publish server binaries and dispatch production deployment. It never pushes to the main branch.
Only exact server version tags participate in server version calculation; mobile releases do not
replace GitHub's server `latest` release.

## Signing environment and secrets

GitHub Settings → Environments → **mobile-release** allows branch `main` and tag `mobile-v*`.
The `mobile-release-tags` ruleset restricts creation, updates and deletion of those tags to repository
administrators. Keep those policies together: environment tag matching alone does not prove a tag
contains a reviewed workflow. The existing repository `RELEASE_TOKEN` is used only for PR creation, main workflow dispatch and
annotated tag/ref creation; its account needs repository administration rights to create protected
mobile tags, plus Actions write and pull-request write permissions. Pull-request CI uses no
distribution secrets.

The repository-level release token is intentionally available to dev release preparation. Maintainers
who can merge workflow/script changes into dev are therefore trusted with its production capabilities.
The dev → main promotion check validates release source, but does not isolate this credential from dev
code. This is the accepted operating model; the main-only signing environment protects signing inputs,
not the repository-level token.

| Environment secret | Value |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | Base64 of the original JKS release keystore |
| `ANDROID_KEYSTORE_PASSWORD` | Keystore password |
| `ANDROID_KEY_PASSWORD` | Private-key password |
| `ANDROID_KEY_ALIAS` | `yourtj-release` |
| `IOS_DISTRIBUTION_P12_BASE64` | Base64 of the Apple Distribution certificate **and private key**, exported as a macOS-compatible PKCS#12 file |
| `IOS_P12_PASSWORD` | PKCS#12 export password |
| `IOS_PROFILE_BASE64` | Base64 of the App Store provisioning profile for `tj.yourtj.forumApp` |
| `ASC_PRIVATE_KEY_BASE64` | Base64 of the App Store Connect `.p8` API key |
| `ASC_KEY_ID` | API key ID |
| `ASC_ISSUER_ID` | API issuer ID |
| `IOS_REVIEW_JSON` | JSON with `firstName`, `lastName`, `email`, `phone`, `username`, `password` for Apple review only |

Use `gh secret set NAME --repo YourTongji/YourTJ-Hub --env mobile-release` with values piped
through stdin. Do not pass secret values on the command line, commit private files or print
base64 into workflow logs. Base64 is encoding, not encryption. `gh secret list --env mobile-release`
can verify names and update times, not retrieve stored values.

Keep an independent encrypted backup of signing assets and passwords. Android updates require the
original signing key; generating a replacement is not a rotation strategy for installed APKs.
The public expected certificate SHA-256 is in `apps/mobile/release-config.json`. iOS distribution
certificates and profiles expire: renew the profile with the matching certificate and update both
secrets as needed. The build validates expiry, bundle/team and the matching private-key identity.
Review credentials belong only in the environment secret and Apple's review fields, never public
release notes, metadata or screenshots. Changing the demo account requires testing its login and
updating the secret; a visual captcha may still be required by the production login policy.

## Launcher artwork

Native icon assets are generated from the Web vector mark using the checked-in
[launcher configuration and wrapper](../../apps/mobile/packages/forum_app/assets/launcher/README.md).
The focused Flutter checks cover all declared iOS sizes/opacity, Android densities, adaptive safe
area and monochrome alpha masks. A source digest detects changes to Web artwork that require
regeneration. A store icon update requires a new IPA/build; already submitted binaries are immutable.

## Android APK and in-app update

Flutter 3.44.9 produces three signed APKs. Their **actual** Android version codes are base build `N`
plus `1000` (armeabi-v7a), `2000` (arm64-v8a) or `4000` (x86_64). Asset names contain that actual
code: `YourTJ-X.Y.Z+CODE-ABI.apk`. iOS uses the unmodified base `N`. The publisher verifies package,
version, ABI and signing certificate before writing `SHA256SUMS.txt` or uploading. Changing Flutter's
split-code algorithm requires updating the validator; a mismatch fails the release. Certificate
validation accepts both numbered signer output and Build Tools 37 scheme labels. Repeated identical
certificates across schemes represent one identity; conflicting certificates, public-key digests and
source-stamp-only output do not satisfy the release certificate check.

APK assets are first uploaded to a draft GitHub release. GitHub-computed SHA-256 digests must match
local files before it becomes public. The publisher resolves drafts through `gh release view` and
queries their database ID, since the REST tag lookup may return 404 for a draft. Existing asset names with different bytes are never overwritten.
The release is published with `latest=false`, so server downloads keep their separate latest marker.

On Android, startup/resume checks at most once every six hours; About exposes a manual check. The
user can defer, ignore a version or cancel a download. Only stable `mobile-vX.Y.Z` releases with a
newer compatible APK and a GitHub digest are considered (the most recent 100 repository releases).
Offline/rate-limit failures stay silent for automatic checks and produce retry feedback manually.

The **authoritative metadata request goes directly to api.github.com over HTTPS**. The download
client probes GitHub, ghfast.top, ghproxy.net and gh-proxy.com concurrently using a bounded Range
request, then tries the fastest responsive source first. Public proxies are best-effort third-party
services; there is no availability guarantee or claimed mainland latency measurement. When GitHub
metadata cannot be reached, the client does not accept unsigned metadata from a proxy. Mirror URLs
are maintained in `lib/src/updates/android_release.dart` and changing them requires an app release.

Mirrors receive only public APK URLs, without forum tokens, cookies or user IDs. Downloads have size,
time and SHA-256 checks; failure/cancellation deletes partial files and corrupt mirrors fall back.
Before opening the installer, Android also checks package name, increasing installed version code,
and equality with the installed app's signing certificate. Files are served only from the app-owned
updates cache through a non-exported FileProvider. The system prompts for installation permission
and installation confirmation. This updates the complete APK; it is not silent installation or a
Dart hot-code update. Debug-signed installations cannot update to the distribution key in place.

## iOS upload and review

The workflow pins and checksum-verifies ASC CLI 5.0.0, then signs using an ephemeral keychain.
Provisioning overrides apply only to the Runner release target, not Pods/SwiftPM dependencies.
Private inputs, temporary profiles and keychain changes are cleaned up even after build failure.
The exported IPA and dSYMs are retained as workflow artifacts for 30 days; private keys are excluded.

App Store Connect app ID is `6809457637`, team `4HJTS3G3T2`, bundle `tj.yourtj.forumApp`. The job
finds the exact version/build before uploading, waits for processing, submits to the existing
**YourTJ TestFlight** external group, updates store localization/screenshots and review credentials,
attaches the same build, validates and submits for App Store review with release after approval.
The publisher recognizes ASC's absent-beta-review exit status on a first submission and normalizes
single-resource lookup collections; ambiguous collections still fail instead of selecting arbitrarily.
It does not add testers or publish a TestFlight public link. Store metadata says “选课社区” and
accurately discloses posts, comments and messaging. Privacy labels, age rating, availability,
agreements and pricing are managed in App Store Connect; source changes affecting disclosures
require updating those forms before release.

`ITSAppUsesNonExemptEncryption=false` reflects the current authentication/TLS/system-storage use.
Reassess the declaration when adding non-exempt encryption. An Apple account capability checkbox
alone does not implement Sign in with Apple or configure notification delivery credentials.

## Failure and recovery

- **Signing failure:** check certificate/profile expiry, team/bundle and JKS alias/password. Never
  fall back to debug signing. Local validation: `publish_android.py --verify-only` with the same
  version/tag environment, or `build_ios.py` with file paths/password supplied via private environment.
- **Partial Android publication:** download the signed APK artifact from the failed run into
  `apps/mobile/packages/forum_app/build/app/outputs/flutter-apk/` and rerun `publish_android.py`
  with `MOBILE_VERSION`, `MOBILE_BUILD_NUMBER`, `RELEASE_TAG`, `ANDROID_HOME` and authenticated `gh`.
  Do not rebuild and overwrite existing assets; signed ZIP bytes may differ even for identical source.
  Verify the artifact belongs to the same tag/commit first.
- **Uncertain Apple upload:** rerun to query the exact build. Existing uploads are not duplicated.
  An incomplete reservation without `uploadedDate` requires inspection with `asc builds uploads list`;
  resolve that failed upload in ASC before resuming. Inspect its state with
  `asc builds uploads view --id UPLOAD_ID` and files with
  `asc builds uploads files list --upload UPLOAD_ID`. Only an abandoned `AWAITING_UPLOAD` reservation
  with no uploaded files may be removed using `asc builds uploads delete --id UPLOAD_ID --confirm`;
  first confirm no uploader is active. Never delete a processing upload or existing build. Invalid
  processing fails immediately.
- **Publisher repair after a tag exists:** the tag and retained signed artifacts remain immutable.
  With `recover_ios=true` and an existing tag, the workflow skips Android and all iOS build/signing
  steps. It uses publisher tools from the dispatch's reviewed main commit, store metadata from the
  original tagged source, and the exact recorded version/build already uploaded to Apple. A missing
  build fails before any upload; only ASC credentials are installed. Leave `testflight_only=false`
  to resume both submission channels, or set it to true to preserve a pending App Store version.
  A normal rebuild still uses tagged source and can produce different signed ZIP bytes. Use retained
  artifacts for Android recovery; never move a tag or overwrite an existing APK to repair tooling.
- **Apple validation/rejection:** correct the reported store fields or app behavior. A binary change
  needs a new version/build release; metadata-only corrections can resume against the existing build.
  Already waiting/in-review/approved submissions using the same build are preserved.
- **Bad public Android release:** mark that release as a prerelease or remove it from public
  distribution, then publish a higher version/build with the fix. Never replace an existing APK
  in place or lower an installed version code.

See [the release decision](../decisions/0014-mobile-release-distribution.md) and
[GitHub environment documentation](https://docs.github.com/en/actions/deployment/targeting-different-environments/managing-environments-for-deployment).

## Native push activation and verification

`Partial`: native authorization/registration, provider routing and release validation are implemented.
APNs/JPush credentials, vendor console configuration and signed-device delivery must be verified for
the deployed environment. CI compiles the iOS bridge and all six Android OEM adapters with build-only identifiers. These APKs are never distributed. A passing SDK build is not a delivery test. The provider decision is
[0019](../decisions/0019-native-push-providers.md).

iOS uses APNs directly; no Firebase project or Firebase Dart defines are required. Enable **Push
Notifications** on Apple Developer's `tj.yourtj.forumApp` App ID. Create an APNs authentication key
under **Certificates, Identifiers & Profiles → Keys**, grant production access for distribution,
and retain its Key ID and downloaded `.p8` securely. App Store Connect upload keys are a different
service and cannot substitute for an APNs key. Regenerate the App Store provisioning profile and
replace `mobile-release/IOS_PROFILE_BASE64`; the signer rejects profiles without
`aps-environment=production`, and checks the exported IPA entitlement before upload. Xcode's export uses the distribution profile to select the production
entitlement; debug development signatures use the sandbox service.

Provision the APNs `.p8` under the production instance's persistent storage directory, readable only
by the application UID. Set the production environment secrets `APNS_KEY_PATH` (the **container**
path), `APNS_KEY_ID`, `APNS_TEAM_ID`, `APNS_BUNDLE_ID`, `APNS_ENVIRONMENT=production`. The deployment
workflow and **Apply / instance config** (when targeting `main`) pass these to the configuration
renderer; config-only applies preserve the same provider settings. Both entry points keep dev
delivery disabled. A path setting does not upload the file;
provision it before deployment. Keep old key files during credential rotation so deployment rollback
can restore the previous configuration. Never copy the private key to dev or into an IPA.

Android uses JPush 6.2.1 / JCore 5.5.2, with pinned OEM adapters selected by the build configuration.
Create a JPush Android app for `tj.yourtj.forum_app`. Obtain its AppKey and Master Secret. Configure
manufacturer services in JPush's **Push settings → Integration settings** using each vendor's
application credentials, registered package, signing certificate fingerprints, notification category
and quotas. Available adapters are Huawei, Xiaomi, OPPO, vivo, Honor and Meizu. Huawei Android/HMS
support does not imply native HarmonyOS NEXT support. OEM channels may require developer verification
or application review; do not claim they are active merely because their adapter is in the APK.

Set `mobile-release/ANDROID_PUSH_JSON` to client identifiers only, for example:

```json
{
  "JPUSH_APPKEY": "<24-character JPush AppKey>",
  "VENDORS": ["honor", "xiaomi"],
  "HONOR_APPID": "<Honor App ID>",
  "XIAOMI_APPID": "<Xiaomi App ID>",
  "XIAOMI_APPKEY": "<Xiaomi client AppKey>"
}
```

Other client keys are `OPPO_APPID/OPPO_APPKEY/OPPO_APPSECRET`, `VIVO_APPID/VIVO_APPKEY`, and
`MEIZU_APPID/MEIZU_APPKEY`. With `huawei` selected, also set
`mobile-release/HUAWEI_AGCONNECT_JSON` to that app's `agconnect-services.json`. The preparation script
writes ignored `android/push.properties` and Huawei configuration files. It rejects missing OEM
parameters, unknown fields and provider server secrets; signed releases require at least one OEM
adapter. Ordinary debug builds without this configuration show push as unavailable.

Set **production** secrets `JPUSH_APP_KEY` and `JPUSH_MASTER_SECRET` for the server. Do not place the
JPush Master Secret or manufacturer server credentials in `ANDROID_PUSH_JSON`. The production
workflow renders `[push.jpush]`. Dev intentionally receives no native push secrets because its
snapshot contains production device registrations. Stopping a deployment channel does not change
existing user consent; enabling it again requires the client to register on resume or retry.

Before distribution, update the published privacy policy and store disclosures for JPush and enabled
OEM SDKs: device push identifiers, device/system/network information and notification title/body are
processed to deliver notifications. The native settings screen explains this before opt-in and links
to the processor policy. Do not include account passwords or forum session tokens in push payloads.

Validate on a physical iPhone using the exact TestFlight build and on each enabled manufacturer's
Android phone without Google services:

1. Sign in and enable push in Settings. The OS permission prompt appears; declining shows a settings
   recovery action. Allowing adds the app to system notification settings.
2. Confirm authenticated `GET /api/forum/push/config` enables the matching provider, and
   `POST /api/forum/push/device/register` succeeds with `provider=apns` or `jpush`. An empty token or
   failed API call must not display enabled. Never paste tokens into public logs.
3. Generate an ordinary notification from a separate test account. Check foreground presentation,
   background/locked delivery, and tap navigation to the corresponding topic/profile/notification page.
4. For Android, test the OEM offline channel after process removal, and inspect the provider delivery
   receipt. Do not equate swipe-away/process removal with the OS's explicit Force stop action.
5. Disable push, then log out/switch accounts. Verify device unbinding and no previous-account routing.
   Re-enable, deny permissions in system settings, return to the app, and verify state recovery.
6. Check production startup channel status and provider error logs. Configuration absence, registration
   failure and provider rejection are separate diagnoses. GitHub/Apple release success alone proves
   none of these delivery conditions.

Public provider setup references: [Apple APNs keys](https://developer.apple.com/help/account/keys/create-a-private-key),
[JPush integration settings](https://docs.jiguang.cn/jpush/console/push_setting/integration_set),
[OEM parameter applications](https://docs.jiguang.cn/jpush/client/Android/android_3rd_param).

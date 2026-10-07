# Mobile releases

> Doc type: operations runbook
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-03

`Current`: YourTJ is publicly distributed through the iPhone/iPad App Store and signed Android APKs,
linked from [the official download page](https://yourtj.de/#download). Public channel status and direct
links are maintained in [Distribution and updates](../product/mobile-experience.md#distribution-and-updates).

Each new candidate still requires the reviewed release workflow on `main`, configured
`mobile-release` environment secrets and successful platform execution. Apple review and the Android
system installer remain independent gates; a completed upload does not mean distribution approval.
GPL source/license delivery and iOS distribution checks follow the
[licensing guide](../development/licensing.md#许可兼容与分发), including the confirmed standard EULA
configuration and the remaining GPL compatibility verification.
`Partial`: native push delivery and signed upgrade/device journeys need candidate-specific evidence;
public availability does not establish the full physical-device acceptance matrix.

## Version and activation

Use [Release / Prepare and the portable release CLI](releases.md). Source must already be in main;
release preparation does not promote dev. Choose Android, iOS or both. iOS defaults to TestFlight;
App Store needs an explicitly approved destination or an existing-build promotion request.

Mobile retains `mobile-vX.Y.Z` and a shared monotonically increasing base build number. Android/iOS
can publish independently and skip versions. The approved source SHA controls the build and tag;
the Release PR head controls notes and targets. Final-head human review is mandatory. Android
What's New comes only from `android.zh-CN.md`; iOS store text comes from `ios.zh-Hans.txt`, and
TestFlight uses separate English testing notes. Static store description/screenshots remain under
`apps/mobile/store/`; `metadata.json` What's New is not a publishing fallback.

The user-facing app renders structured formal release notes with its own typography and sections;
it does not display the App Store text file. iOS About history shows only App Store releases, without
a TestFlight tab. Store-page What's New is a separate plain-text rendering. TestFlight's separate
plain-text testing instructions are displayed by Apple; see [Apple's test-information guide](https://developer.apple.com/help/app-store-connect/test-a-beta-version/provide-test-information/).
The app does not show a second automatic beta-notes prompt. Manual update checks from a TestFlight
build open the existing TestFlight App Store product-page fallback; they do not open a YourTJ beta
deep link. Missing App Store history is never filled from TestFlight.

New release requests use reviewed structured changelog facts and evidence references to render each
channel's notes. The status site's `/mobile/releases.json` catalog appears only after a successful
channel receipt is bound to the merged candidate's source SHA and content digest. App Store submission
or review is not public availability; TestFlight requires `APPROVED`. Catalog coverage is tracked
per channel, so old unstructured releases do not claim complete note history. See the
[catalog contract](../../apps/status/api/openapi.yaml) and [release pipeline decision](../decisions/0059-receipt-backed-mobile-release-catalog.md).
`completeFromBuild` is an inclusive minimum installed-build threshold for claiming a complete
upgrade range, not a claim that every build from that number has a structured entry. When no older
unstructured public build is known, it equals the first structured build; installations below that
floor have unknown prior history and the client must not claim completeness.
Release requests allow at most 100 entries per changelog group, the client's per-release limit; a
build whose merged channel entries exceed it is left out of the catalog. The catalog keeps the newest
structured builds that fit both the client's 300-release limit and the 1 MiB limit shared by the
status proxy and the client, measured on the exact published UTF-8 bytes; publishing rechecks the
size before upload. Coverage lists only retained builds, so the newest dropped public build becomes
the floor. A required disclosure keeps its
TestFlight channel as a required entry beside the verbatim testing note; the client shows the required
copy once, ahead of ordinary notes.

Recovery is channel-specific and uses original signed artifacts or an exact already uploaded Apple
build. Uncertain uploads are queried before retransmission. A pending other App Store version is
reported as a blocker, never withdrawn automatically. See the [recovery rules](releases.md#recovery-and-completion).
Apple agreements, reviews and physical-device/system-installer checks remain independent gates.

## Signing environment and secrets

GitHub Settings → Environments → **mobile-release** allows branch `main` and tag `mobile-v*`.
The `mobile-release-tags` ruleset restricts creation, updates and deletion of those tags to repository
administrators. Keep those policies together: environment tag matching alone does not prove a tag
contains a reviewed workflow. The repository `RELEASE_TOKEN` is used only for annotated tag/ref creation. Grant Contents write;
its identity must be allowed to create protected mobile tags by the tag ruleset. It needs neither
Actions write nor Pull requests write. Pull-request CI uses no distribution secrets.

Release request PRs use a scoped GitHub App installation token so normal CI is triggered. The model
child receives no GitHub token. The protected main controller uses `RELEASE_TOKEN` only for immutable
tag creation; signing keys are passed only to selected platform steps. Pull-request CI has no
signing/Apple keys. Keep main-only environment restrictions and tag rules together.

| Environment secret | Value |
|---|---|
| `ANDROID_KEYSTORE_BASE64` | Base64 of the original JKS release keystore |
| `ANDROID_KEYSTORE_PASSWORD` | Keystore password |
| `ANDROID_KEY_PASSWORD` | Private-key password |
| `ANDROID_KEY_ALIAS` | `yourtj-release` |
| `HUAWEI_AGCONNECT_JSON` | Huawei AGConnect Android client configuration for the optional Huawei adapter; package must be `tj.yourtj.forum_app` |
| `FCM_GOOGLE_SERVICES_JSON` | Firebase Android client configuration for the optional FCM adapter; package must be `tj.yourtj.forum_app` |
| `IOS_DISTRIBUTION_P12_BASE64` | Base64 of the Apple Distribution certificate **and private key**, exported as a macOS-compatible PKCS#12 file |
| `IOS_P12_PASSWORD` | PKCS#12 export password |
| `IOS_PROFILE_BASE64` | Base64 of the App Store provisioning profile for `tj.yourtj.forumApp`, with production Push Notifications and App Group `group.tj.yourtj.forumApp.widgets` |
| `IOS_WIDGET_PROFILE_BASE64` | Base64 of the App Store provisioning profile for `tj.yourtj.forumApp.ScheduleWidgets`, with App Group `group.tj.yourtj.forumApp.widgets` |
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
code: `YourTJ-X.Y.Z+CODE-ABI.apk`. iOS uses the unmodified base `N`.
The Android app exposes that base `N` from its build configuration for catalog matching. The
download candidate subtracts its selected APK ABI's fixed offset; base builds above 999 retain
their full identity. APK update/install comparisons still use actual Android version codes.
The publisher verifies package, version, ABI and signing certificate before writing `SHA256SUMS.txt`
or uploading. Changing Flutter's
split-code algorithm requires updating the validator; a mismatch fails the release. Certificate
validation accepts both numbered signer output and Build Tools 37 scheme labels. Repeated identical
certificates across schemes represent one identity; conflicting certificates, public-key digests and
source-stamp-only output do not satisfy the release certificate check.

APK assets are first uploaded to a draft GitHub release. GitHub-computed SHA-256 digests must match
local files before it becomes public. The publisher resolves drafts through `gh release view` and
queries their database ID, since the REST tag lookup may return 404 for a draft. Existing asset names with different bytes are never overwritten.
The release is published with `latest=false`, so server downloads keep their separate latest marker.

`Current`: after publishing the original APKs, the Android job refreshes an independent
[`mobile-latest` download channel](https://github.com/YourTongji/YourTJ-Hub/releases/tag/mobile-latest):

| Device architecture | Fixed APK download |
|---|---|
| ARM64, most current Android phones | [YourTJ-arm64-v8a.apk](https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-latest/YourTJ-arm64-v8a.apk) |
| ARM 32-bit | [YourTJ-armeabi-v7a.apk](https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-latest/YourTJ-armeabi-v7a.apk) |
| x86 64-bit | [YourTJ-x86_64.apk](https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-latest/YourTJ-x86_64.apk) |
| Website download probe | [YourTJ-download-probe.png](https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-latest/YourTJ-download-probe.png) |

The publisher selects the highest stable mobile version among the most recent 100 releases, downloads
all three source APKs and verifies their GitHub digests, sizes and matching ABI build numbers before
changing the aliases. It uploads the APKs unchanged under fixed names, stages the checked-in probe and
verifies GitHub's resulting digests. `SHA256SUMS.txt` records the APK aliases and probe. The channel
is marked pre-release only to keep it
out of the repository-wide Latest and the in-app updater; its APKs are stable release copies.
The source marker prevents older recovery from replacing a newer channel, and matching files are
skipped on retry. Canonical `mobile-vX.Y.Z` assets are never overwritten.

The channel also publishes a fixed 128–256 KiB PNG probe at
`https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-latest/YourTJ-download-probe.png`.
The website may load this same asset from GitHub and from a static allowlist of public mirrors using
`new Image()` requests in parallel, with a 3.5–4.5 second timeout per route. Keep the download link
pointing to GitHub until probing finishes; if every route fails, or results are close, use GitHub.
Do not resolve a mobile version through the GitHub API or download an APK for this measurement. The
probe measures only a short network transfer; it is not version metadata or an APK integrity signal.
The Android app's separate release-update mirror probing continues to use its existing APK flow.

Alias updates replace files individually. A URL can briefly return 404, and checksums may lag while
publication is running; use the original version linked in the channel notes for a consistent set.
The alias tag is not moved, so use the versioned source tag from those notes rather than the channel's
automatically generated source archives. These trade-offs are recorded in
[the fixed-download decision](../decisions/0050-android-stable-download-links.md).

To verify or repair only this download channel, with authenticated `gh` and no mobile release running:

```bash
python3 scripts/mobile-release/publish_android_latest.py --verify-only
python3 scripts/mobile-release/publish_android_latest.py
```

This needs no signing inputs and never rebuilds or publishes a new application version. An immutable
or unrecognized release at `mobile-latest` fails safely instead of changing its protections.

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
The archive installs and validates separate App Store profiles for Runner and the ScheduleWidgets
extension, then verifies the extension bundle and shared App Group entitlement inside the exported IPA.

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
  through the reviewed Recover workflow; it supplies the frozen identity, approved Android notes and authenticated `gh`.
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
- **Publisher repair after a tag exists:** use Release / Recover with the original candidate and
  approved channel. Source/tag/notes and signed artifacts remain immutable. The trusted reviewed
  controller is recorded in each receipt. Apple is queried before selecting an existing build or
  restoring the original IPA. Android-only recovery cannot change Apple state. Missing original
  artifacts are an explicit recovery failure, not permission to rebuild the same identity.
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
the deployed environment. [Native CI](../development/testing.md#ci-mapping) compiles the changed
platform on native/dependency changes, or both platforms on manual dispatch, using build-only
identifiers for the iOS bridge and all seven Android push adapters. These APKs are never distributed.
A passing SDK build is not a delivery test. The provider decision is
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

Android uses JPush 6.2.1 / JCore 5.5.2. Create a JPush Android app for `tj.yourtj.forum_app` and
obtain its AppKey and Master Secret. The AppKey is required in the client build; the Master Secret is
server-only. OEM offline adapters are optional. Without them, the JPush channel depends on the app's
long connection and cannot guarantee delivery after Android stops the process. To add an adapter,
configure that manufacturer's service in JPush **Push settings → Integration settings** using its
application credentials, registered package, signing certificate fingerprints, notification category
and quotas. Available adapters are Huawei, Xiaomi, OPPO, vivo, Honor, Meizu and FCM. FCM also needs a
Firebase Android app for `tj.yourtj.forum_app`, its `google-services.json` client configuration, and
the FCM service-account JSON uploaded directly in JPush Console. Keep the service-account private key
out of GitHub and the APK; `FCM_GOOGLE_SERVICES_JSON` is the separate Firebase client configuration.
Huawei Android/HMS support does not imply native HarmonyOS NEXT support. OEM channels may require
developer verification or application review; do not claim they are active merely because their
adapter is in the APK.

Set `mobile-release/ANDROID_PUSH_JSON` to client identifiers only. A JPush-only release without OEM
adapters needs no vendor credentials:

```json
{
  "JPUSH_APPKEY": "<24-character JPush AppKey>",
  "VENDORS": []
}
```

Other client keys are `OPPO_APPID/OPPO_APPKEY/OPPO_APPSECRET`, `VIVO_APPID/VIVO_APPKEY`, and
`MEIZU_APPID/MEIZU_APPKEY`. With `huawei` selected, also set
`mobile-release/HUAWEI_AGCONNECT_JSON` to that app's `agconnect-services.json`. With `fcm` selected,
set the `mobile-release/FCM_GOOGLE_SERVICES_JSON` secret to the matching Firebase Android app's
`google-services.json`. The preparation script writes ignored `android/push.properties` and selected
provider configuration files after checking their package names. It rejects missing parameters for
selected adapters, unknown fields and provider server secrets. Signed releases require a valid JPush
AppKey but do not require an offline adapter. Ordinary debug builds without a `push.properties` AppKey
show push as unavailable. Local dev APKs can use the same `prepare_push.py` inputs before running
`apps/mobile/scripts/build_dev_apk.sh`; generated JSON files are ignored by Git.

FCM is an optional transport inside JPush; it does not bypass a disabled server JPush channel.
Keep Firebase Messaging auto initialization and Analytics collection disabled in the Android manifest:
JPush requests the FCM token only after OS notification permission is granted and the existing
consent-aware registration flow initializes the SDK.
The settings screen continues to report a disabled server channel accurately on both Android and iOS.
See [Firebase startup controls](https://firebase.google.com/docs/cloud-messaging/android/get-started#prevent-auto-initialization).

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

1. On Android, install fresh or upgrade from a build without the one-shot permission marker. For an
   upgraded install, verify a previously explicit push opt-out stays off and does not trigger a
   prompt; with no saved preference, sign in and verify the OS prompt appears only after the main
   screen is stable, never on the login route or
   during OAuth return. Allowing reaches the existing registration path; declining shows
   `permissionDenied`, does not register a device, and does not prompt again on restart/resume. Settings
   retains system-settings recovery and explicit retry. If the app process exits while the OS prompt
   is open, verify that the next launch repeats the permission check/request. On iOS, verify the
   existing one-time request and that an interrupted request leaves the shared marker unset until
   the native permission call returns.
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
[OEM parameter applications](https://docs.jiguang.cn/jpush/client/Android/android_3rd_param),
[JPush Android vendor-channel integration](https://docs.jiguang.cn/jpush/client/Android/android_3rd_guide).


## Release verification and privacy disclosures

`Current`: normal Android and iOS releases both depend on the same `verify` job against the
reserved source SHA. It runs Flutter analysis/tests and release-tool tests before signing or
publishing. Publisher-only iOS recovery explicitly skips this build gate because it reuses an
already uploaded immutable build; it still runs publisher-tool checks. A successful local subset
does not establish CI or physical-device acceptance.

Runner, ScheduleWidgets and the bundled home_widget SDK contain privacy manifests. SwiftPM and
CocoaPods both include the SDK resource. The exported IPA validator checks those actual bundles
for the App Group UserDefaults reason; a source-only declaration does not satisfy this gate.
Runner also declares the first-party account, user content, message, device and campus data
categories. The local-only Widget/SDK do not collect data off device.

`Current`: the [App privacy supplement](../../apps/gooseforum/app/models/defaultconfig/pageconfig/app_privacy.md)
is embedded in the forum binary and appended to enabled `/privacy` pages, including persisted
custom policies. Rendering replaces an existing App supplement section with the embedded current
version, preserving surrounding custom policy sections and avoiding duplicates. The App also bundles
a four-language visit-statistics disclosure under About, independent of the server policy toggle or
network availability.
It covers campus processing, device snapshots/Widget display, selected-message reporting,
Android push processors and automatic first-party visit analytics. Publish this server before distributing the corresponding App.

`Partial`: App Store Connect privacy declarations still require verification against the actual
production SDK selection, retention and analytics configuration. Source privacy manifests and the
public policy are not substitutes for the App Store Connect form.

App Store privacy labels must account for account identifiers/contact information, private
messages, uploaded images, other user content, interaction state, device push identifiers and
school-authorized data. They support App Functionality and are linked to the account where
applicable. The reviewed source implements no cross-company advertising tracking. WebView/public
website analytics must be assessed from the actual Umami configuration before declaring labels;
do not infer “no data collected” from the Widget manifest.

`Current`: iOS retains password, Tongji, Google and GitHub and adds native Apple login.
Users connect Apple to their existing account in settings first; no email/name scopes or automatic
email-based account merging are used. Native Apple authorization must be tested with the actual
candidate. The [Apple login decision](../decisions/0041-native-apple-login-and-revocation.md)
describes credential retention and revocation.

Enable Sign in with Apple for `tj.yourtj.forumApp`, and create a dedicated P-256 Sign in with Apple
key associated only with that primary App ID. Native authorization does not require a web Services ID
or email-relay source. Set `APPLE_CLIENT_ID` (the bundle ID), `APPLE_TEAM_ID`, `APPLE_KEY_ID` and
`APPLE_PRIVATE_KEY_BASE64` (base64 of the `.p8` key) in the GitHub **production** environment used by
main deployment. Feed values via stdin; never paste private key contents into chat, logs or commits.
The renderer writes `[apple]` server configuration. All four values must be valid before public
`appleReady`/`appleOAuthReady` become true. Dev credentials remain empty because its database is a
production snapshot; production grants must not be redeemed or revoked there. A copied Apple binding
cannot be disconnected or closed on dev without a separate isolated Apple test configuration.

Distribution profiles and exported IPA entitlements must contain
`com.apple.developer.applesignin = ["Default"]`; the signing validator rejects missing capabilities.
Unlink/account closure calls Apple's revocation endpoint before committing local deletion. A provider
outage preserves the encrypted grant and active account for retry. Preserve the forum signing key
used to encrypt existing grants; rotating it without re-encryption prevents revocation.

A new server containing block enforcement, private-message reporting and optional
`clientMessageId` support must be deployed before releasing the matching mobile binary.
`Current`: automatic chat image loading trusts the configured API origin. Deployments returning
image URLs on a separate CDN must also supply `--dart-define=YOURTJ_CHAT_IMAGE_ORIGINS=https://cdn.example.com`
(comma-separated exact origins, including scheme and port; no paths or wildcards). Redirects must
remain within these origins. Unlisted image URLs display as links instead of loading automatically.
Signed APK upgrade, external-browser OAuth return, APNs/JPush/OEM delivery and Widget behavior
require recorded physical-device evidence for the actual candidate version/build.

## Native visitor statistics

`Current`: release builds already supply `YOURTJ_API_BASE_URL=https://f.yourtj.de`; only that exact
production origin on native Android/iOS enables the transport. No additional secret or analytics SDK
is needed. The public website ID and collection URL are fixed in `analytics/visitor_analytics.dart`;
never put Umami account credentials into Dart defines or the App. Public-page statistics start
without user action for guests and signed-in users. There is no settings switch; new installations
and upgrades ignore the legacy `visitor_analytics_opt_in` value, including a saved `false`. Release
notes must disclose the automatic collection policy when distributing this change.

Umami accepts page views at `https://umi.yourtj.de/api/send` with explicit `browser: yourtj-app`,
`os: iOS|Android OS`, `device: mobile|tablet` and `tag: yourtj-app`. The custom User-Agent includes
OS and device family so the current Umami bot filter accepts native requests; its fixed protocol
version is not the App release version. Queries/IDs/content are stripped locally. A named custom
event would not contribute to the device page-view report, so `payload.name` must remain absent.

Verify the candidate on a fresh installation and an upgrade with a previously disabled preference:
open a public page without configuring statistics and inspect the joint Umami report for `yourtj-app`.
Verify private pages send nothing and backgrounding stops sends. Local automated tests use a fake
transport and never inject test visitors into the production site. Historical WebView
rows cannot be relabelled as App. The status collector refreshes device reports every five minutes
when its server-only report credentials are configured.

`Partial`: before each App update, verify the App Store Connect privacy form against the actual
Umami configuration and retention policy. The source manifest includes analytics use of product
interaction/other data and IP-derived coarse location, with no advertising tracking. The embedded
privacy supplement discloses automatic collection without a switch, receive-side IP and regional
derivation. Deploy the updated server privacy supplement alongside the App so persisted site policies
also show the current disclosure. Leaving the App does not delete retained server aggregates.

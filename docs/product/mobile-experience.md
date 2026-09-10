# Mobile experience

> Doc type: product spec
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-08

The Flutter app combines the forum, course catalog, scheduler and Wiki. Ordinary browsing and
writing use native pages. Management uses the same first-party workspaces and permission checks as
Web inside an authenticated in-app browser. The navigation and management boundary are recorded in
[0012](../decisions/0012-unified-mobile-reading-navigation.md).

## Navigation and reading

- `Current`: Home announcements render optional titles and HTML bodies, including the legacy
  single-HTML payload. A small bell sits in a separate leading column, with title and body aligned
  to the same inset as Web. They grow with their contents and text size; empty announcements take no
  space. Multiple announcements rotate with numbered manual controls; assistive navigation and
  reduced motion disable automatic rotation. Refresh replaces the active announcement safely.
- `Current`: mobile body text uses 17 logical pixels with system text scaling. Feed cards use
  compact vertical padding and one timestamp; embedded Markdown uses smaller paragraph margins
  so short replies do not acquire a large empty footer.

- `Current`: four persistent destinations — Home, Campus, Notifications and Messages — use icon-only
  navigation with accessible labels. Search is a pushed page, reachable from Home. Campus links to
  the native course catalog, scheduler and Wiki; returning preserves the selected destination.
  About links to native friend links, sponsors, terms and privacy pages using the site’s published
  configuration; disabled policies remain hidden.
- `Current`: Home cards retain both images for two-image topics. A portrait single image sits beside
  the text; a landscape image appears below the text with aspect-preserving fit. Portrait galleries
  show up to three columns; two landscape images share a row; larger landscape galleries overlap
  up to three previews with a total count. Tapping opens the full gallery with zoom.
- `Current`: Home topic cards expose compact authenticated like and bookmark shortcuts beside the
  reply/view metrics. A successful action updates its selected icon immediately; failed actions
  preserve the previous state and show the localized error. Home summaries batch-load the viewer's
  like/bookmark state; absent state (anonymous, unavailable or older servers) suppresses the
  shortcuts. Selected states survive offscreen card recycling, and returning from detail refreshes
  them. In-flight reads cannot overwrite newer successful actions. Metrics and actions wrap at
  narrow widths and enlarged text sizes. Like totals are not part of the home summary.
- `Current`: simple-content topics show an uncropped, swipeable image gallery above the body. The
  same gallery is used in the publishing preview.
- `Current`: root headers, filter rails and bottom navigation overlay the reading viewport. They
  hide after 48 logical pixels downward and return after 12 pixels upward, with 200 ms transitions.
  Hidden headers are clipped at the system safe-area edge; the reading viewport stays stable.
  Reaching the top, changing destination or opening the account drawer restores the controls.
  Reduced motion removes the transition; keyboard/modal interaction keeps controls visible.
  Editors and scheduler grids are pushed pages outside this behavior.
- `Current`: Home, Campus and Notifications have a stable compose button; Messages has a new-chat
  button. Topic pages keep reply and floor controls in the bottom dock. Pull-to-refresh and
  reselecting the active root destination provide refresh and return-to-top without changing icons.
  Refreshable pages accept a short pull from the top on release, including empty lists; the gesture
  uses finger travel so tall screens and iOS rubber-band damping do not demand a longer pull.
  Home, Campus and Notifications show the refresh indicator below their overlaid navigation.
  Small or retracted pulls do not refresh, and an ongoing refresh cannot be started twice.
- `Current`: the floor slider loads the selected server window on release. Earliest/latest
  shortcuts, reply links, and earlier/later pagination navigate the actual reply stream. Returning
  to the first post from a middle window reloads that window before offering refresh; stale
  pagination responses are discarded after a floor or session change.

- `Current`: topic and reply authors open their public profiles. Owners can edit/delete their
  content; replies support likes, bookmarks, sharing and paginated revision history. Moderation
  actions follow server capabilities. Removed content has an explicit placeholder; posting
  restrictions suppress reply controls, while anonymous users can proceed to login.
- `Current`: reply edits preserve unsaved text on failed saves and confirm before discarding.
  Server-required reply captchas can be refreshed without losing the draft. Session changes
  invalidate pending edits and destructive confirmations; share failures remain visible in-app.

- `Current`: embedded reply Markdown adds no device safe-area spacing. Dates and reply actions
  share a compact footer, wrapping on narrow screens or large text. Reply references use Web's
  subtle background and left rule, an author/avatar/floor header, and a four-line preview with
  expand/collapse controls only when the rendered text overflows.
- `Current`: notification headings resolve the same template keys and event types as Web in the
  selected language, with actor names and topic/content previews. Legacy literal headings take precedence when no template key is present; content previews take
  precedence over topic titles. Protocol-key filtering applies only to heading fields, preserving
  user content and badge names that begin with the same prefix.

## Language and presentation

- `Current`: native launcher icons use Web's YourTJ cat mark. iOS includes opaque device and
  App Store sizes; Android includes legacy densities, adaptive masks and a themed monochrome layer.
  The launcher artwork is generated independently of the in-app horizontal wordmark.

- `Current`: the native app supports the same four languages as Web: Simplified Chinese, English,
  Japanese and German. Login and Settings expose an immediate language picker with a follow-system
  option; unsupported system languages fall back to Chinese. The device preference survives restart
  and is independent of public profile language. Switching preserves the current page, session and
  unsaved input.
- `Current`: API requests send the selected language without recreating the authenticated client.
  Notification templates and server message translations reuse Web's catalogs in all four languages;
  authenticated management workspaces inherit the choice through the first-party language cookie.
  User-written content and server-defined badge names remain in their original language.
- `Current`: profile and settings use Web's Lucide line icons, with subtle semantic color tiles for
  activity and account controls. Social links use all six Web provider marks and brand colors.

- `Current`: search uses one filled capsule field across messages, new conversations, the course
  catalog, global search, Wiki and scheduler. Search fields provide a localized clear action and keyboard
  submission where applicable. Clearing global search resets results, scope and pagination, and
  invalidates pending requests; account and publishing forms retain their separate form styling.
- `Current`: topic read-only view/reply counts sit above independent reply, like, bookmark and watch
  actions. Active actions use Web's semantic tints, actions wrap on narrow screens, and the reply
  heading has no decorative discussion icon. Topic subscriptions use topic-specific labels; reply
  commands have no toggle semantics. The dock switches to an accessible icon-only reply action when
  its label cannot fit, including long translations and enlarged text. SVG icons inherit their enclosing button foreground
  unless a semantic or provider color is explicitly set.

## Publishing

- `Current`: publishing uses an unframed title and writing canvas. Article formatting tools remain
  folded in a bottom accessory bar above the software keyboard; expanding them preserves the editor
  selection. Image and draft actions remain in the accessory bar; the empty simple gallery is a compact
  selection tile. Rich and simple body text use the same mobile reading scale. Preview hides the accessory bar.

- `Current`: reply composers use one rounded surface with a borderless, growing two-line input.
  Image, hide-keyboard, collapse and send actions share the bottom row. The reply target is a
  lightweight text row; attachment previews and server-required captcha controls appear only when needed.

- `Current`: publishing and reply composers have a localized hide-keyboard button that preserves
  unsent text. Dragging the publishing page or topic stream also dismisses the keyboard; opening
  the publishing preview removes editor focus. Rich-text formatting remains available while editing.

- `Current`: the type selector keeps Web's moment/question/article values. Moments and questions
  use a simple gallery plus text; articles use an inline rich editor backed by Markdown. Article
  formatting tools are folded by default. Existing topics retain their type when edited.
- `Current`: simple galleries support up to nine uploaded images, reordering and removal. Images
  survive switching to the article editor. Switching back extracts images into the gallery and
  plain text into the body; conversion is rejected when more than nine images would be lost.
- `Current`: Next opens the preview/classification step. Up to three existing categories can be
  selected below the rendered image/title/body preview. Publish writes only after this step; saving a draft retains the server's title,
  body and classification requirements. Moments can derive their title from the first text line.
- `Current`: publishing limits, captcha requests and other API failures use the Web error catalog
  in the selected language, including server-provided parameters.
- `Current`: unsaved changes prompt before leaving. A server-required captcha is shown in the
  composer and can be refreshed without discarding content.
- `Planned`: text-to-image cards and offline draft autosave. No UI claims these features exist.

## Campus and sign-in

- `Current`: Campus previews real reviewed courses and links to the course catalog, scheduler and
  Wiki. It does not display an official personal calendar or claim an enrollment integration.
- `Current`: the scheduler opens in course selection. Plan preview remains a local planning
  grid, with week filters, conflicts, custom blocks and existing plan operations. Web and mobile
  warn about time conflicts before a teaching class is selected, while keeping the add action
  non-blocking. A prominent tip opens
  the full [Web scheduler](https://f.yourtj.de/schedule) in the external browser without transferring
  the native credential. Plans are not official enrollment results.
- `Current`: signed-in plans cloud-sync with the Web scheduler (`GET/PUT/DELETE /api/pk/plans`,
  issue #537): local changes upload after a 3s debounce, entering the scheduler reconciles against
  the cloud snapshot (empty cloud auto-uploads local; conflicting edits show a one-time
  use-cloud / keep-local dialog), and the server's `updatedAt` clock is the only sync authority.
  Uploads carry the observed server revision; HTTP 409 triggers another read and a conflict
  dialog. Initial read failures and unresolved conflicts block writes. Pending local changes
  survive page exit and transient failures, and the sync clock advances only after local
  persistence succeeds. Switching accounts requires choosing the cloud copy or explicitly
  keeping the retained local plans, including when the new account has no cloud snapshot.
  Signed-out use stays purely local with zero requests; account closure deletes the cloud copy.
- `Current`: course details retain offering-specific five-star reviews and existing review fields;
  bookmark and write-review actions stay in a bottom dock. Scores share a baseline with their
  five-point denominator. The signed-in user’s own reviews (including anonymous reviews) appear
  first across pagination; edit/delete controls remain on those rows.
- `Current`: Profile includes a private My course reviews entry for paginated management across
  courses, including anonymous reviews. Each visible review can be edited, deleted or opened at
  its offering and review position. Hidden reviews remain listed for deletion, with no edit or
  public-detail action; deleted reviews are omitted. Course detail and management share the same
  editor and a rounded delete confirmation with the target review excerpt and explicit cancel.
- `Current`: shared transient feedback appears in dismissible top banners above sheets, below
  the system safe area. Course review failures show localized server reasons and preserve the
  draft; success and error messages use the same surface with distinct semantic icons.
- `Current`: Wiki search uses the existing page-grouped search contract, debounces input, ignores
  stale results and opens paragraph anchors. Search unavailability has retry feedback. Reading
  keeps directory, Wiki search and GitHub edit actions in a bottom dock; GitHub remains the content
  source of truth.
- `Current`: sign-in offers account/password, Google and GitHub. Password captcha and TOTP remain
  supported. Google availability follows the published Web configuration. Social buttons use the
  existing OIDC code/PKCE exchange with an allowlisted provider hint, not a new credential flow.

## Registration

- `Current`: registration loads the current Web login configuration before submission. Restricted
  email domains use a prefix field and domain selector; unrestricted sites accept the full address.
  Password confirmation is checked locally. Only published terms/privacy policies are linked and
  require explicit agreement. Configuration failures preserve the form and offer retry.

## Profile and privacy

- `Current`: avatar and cover uploads open a native drag/pinch crop preview with an accessible
  zoom slider and reset action. Avatars export at 300×300; covers at 1600×320 with the central
  mobile area marked. Camera orientation is normalized before cropping. Failed uploads retain
  the selection for retry; covers can also be removed with confirmation.
- `Current`: OAuth connections show native account identity and provider availability. Binding
  opens the existing site settings in the system browser, where the user signs into the matching
  account; returning refreshes the native binding list. Unbinding remains native, including an
  existing Google connection when new Google sign-in is disabled. This browser flow does not
  transfer the native session and may require a separate Web login.
- `Current`: account settings support username changes and the twelve built-in avatars. Server
  validation remains visible in the username form so rejected names can be corrected and retried.
- `Current`: profile editing includes nickname, bio, signature, website name/URL, profile language
  and the six Web social providers. Saving preserves unedited fields and unknown social providers;
  website/social destinations accept HTTP(S), and social usernames expand to provider URLs.
  Public profiles display website/social links with the corresponding provider marks and open
  them in the system browser. Worn badges appear on the avatar independently of the badge list;
  administrator identity has a localized role label. Returning
  from settings refreshes profile identity and media immediately.

- `Current`: the root avatar opens an account drawer with profile, bookmarks, a folded content
  management group (drafts, content and recycle bin), settings and permission-gated workspaces.
  The profile overflow retains these infrequent entries. Account controls are outside the public profile.
- `Current`: activity entries distinguish signup, post, like, follow and comment with matching
  icons and localized captions in bordered cards with a content preview and compact timestamp. Stream changes retain the profile header collapse, limiting deep offsets to the start of the
  new stream so loading, empty states and retry actions stay visible. Empty badge lists use
  badge-specific feedback.
- `Current`: profile bios trim boundary whitespace; signatures use a separate feather mark and subtle
  underline. Avatar overlap participates in layout so it leaves no translated blank space. The role
  label stays beside the name; earned badges appear as bordered title/description cards with colored
  hexagons and their server-provided SVGs. The selected badge remains attached to the avatar.
- `Current`: users with follow permission retain the follow button for already-followed accounts,
  including administrators. It displays the followed state and toggles to unfollow, prevents duplicate
  in-flight requests and restores the previous state when a request fails.
- `Current`: only the active profile tab displays its label; all tabs retain accessible names.
  Activity, topics, likes, own bookmarks, follows/followers and badges fetch their corresponding
  server streams. Cursor pagination uses the server's next URL within the same user's profile.
- `Current`: content management and recycle bin provide topic/reply filters, cursor loading,
  multi-selection, restore and deletion. Restore/permanent-delete affordances follow the server's
  capabilities; confirmation/password requirements and partial batch failures remain authoritative.
- `Current`: privacy settings link to content management and account closure. Closure offers
  anonymized-history and best-effort content-deletion modes, requires the current password and
  clears the native session on success. Retention and authorization rules match Web.

## Management workspaces

- `Partial`: the complete Web admin console and moderation workspace are reachable in the App.
  The native wrapper and authenticated handoff pass local iOS and Android login/draft/console
  journeys; all 28 administrative modules fit both viewports, including the link editor.
  The journey creates and deletes a friend link through the real administrative form.
  Both independent course workspaces also pass authenticated handoff and viewport checks on iOS
  and Android.
  Android export sharing and a JSON file selected through the system picker also pass a local
  device journey; the import is not submitted by that test. On iOS the export reaches the native
  share sheet, but dismissal/file selection has not completed under the available simulator UI
  automation. Remaining administrative forms and that iOS file round-trip still need device
  validation before claiming complete mobile parity.
- `Current`: course management and course-review moderation have separate entries in Profile and
  the course catalog. Only CourseManager or Admin can discover and enter them; forum moderator
  status alone does not grant access.
- The embedded browser accepts only the configured first-party origin. Production requires HTTPS;
  cleartext is permitted only for local development hosts. Outside links open in the system browser
  without the native Bearer header. No bearer is placed in a URL or injected into JavaScript.
- The handoff accepts an explicit Bearer credential, verifies the existing session and workspace
  permission, sets an HttpOnly SameSite=Lax cookie, then redirects to an allowlisted workspace.
  Cookie-only requests cannot establish a browser session. All responses are `no-store`; existing
  revocation, role, writable-account and CSRF checks remain active.
- Android file inputs use the system file selector; iOS uses WebKit's picker. Export navigation is
  restricted to the exact same-origin admin export endpoint. Native downloads do not follow
  redirects, require an attachment response, and share the actual JSON/CSV filename. Temporary
  files are removed after sharing. The browser's cookies/storage/cache are cleared on exit.

## Distribution and updates

- `Partial`: Android checks GitHub mobile releases at startup/resume with a six-hour limit and a
  manual About action. Update prompts support defer, ignore, progress and cancellation. Public APK
  mirrors are ranked with bounded probes; SHA-256, package and signing-certificate checks precede
  the system installer. Unit tests and signed native builds cover the implemented paths; the first
  GitHub-hosted release and an installed-to-updated device journey remain distribution validation.
- `Partial`: iOS uses TestFlight and App Store distribution through the same versioned release job.
  Apple processing/review is independent of CI. The app does not offer APK-style updates on iOS.
  Signing, metadata, failure recovery and environment secrets are documented in the
  [mobile release runbook](../operations/mobile-releases.md).

## Verification boundaries

The source, contract and focused Flutter/Go tests define the implemented behavior. Figma is the
editable visual counterpart, not an alternative API or permission model. The maintained design is
[06 Mobile · Unified](https://www.figma.com/design/eLF6vFbmdwDQXec1IyuA4X/YourTJ_Mob_App_Design?node-id=284-302). Native device behavior,
Linux-rendered goldens, distribution and additional locales have independent verification gates;
local widget tests do not imply those gates passed.

## System notifications

- `Partial`: iOS uses direct APNs; Android uses JPush with selected OEM offline adapters and does not
  require Google Play services. Provider credentials, signing profiles and physical-device delivery
  remain deployment requirements; app-local notification lists are independent of system delivery.
- `Current`: Settings retains the push entry with a provider-processing disclosure and shows missing
  build configuration, unavailable server channels, permission denial and registration failure.
  Explicit enable requests system permission. Resume checks existing authorization without repeatedly
  prompting; a non-empty token and successful API registration are required to display enabled.
  Enable taps during startup/resume or stop are queued; a later disable or account change cancels
  queued consent. Failed unbinding is retained and retried on resume while push stays disabled,
  using the owning account; signing in to a different account does not acknowledge that cleanup.
- `Current`: notifications use the existing event copy and only navigate to supported in-app topic,
  profile and notification routes. Logout stops native delivery and attempts server unbinding;
  the next account requires fresh consent. Optional JPush analytics/location collection is disabled.
- See [activation and device validation](../operations/mobile-releases.md#native-push-activation-and-verification)
  for credentials, supported OEMs and delivery limitations.

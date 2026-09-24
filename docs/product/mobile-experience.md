# Mobile experience

> Doc type: product spec
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-20

The Flutter app combines the forum, course catalog, scheduler and Wiki. Ordinary browsing and
writing use native pages. Management uses the same first-party workspaces and permission checks as
Web inside an authenticated in-app browser. The navigation and management boundary are recorded in
[0012](../decisions/0012-unified-mobile-reading-navigation.md).

## Navigation and reading

- `Current`: paginated feeds, search, notifications, profiles, content management, own course
  reviews and post history automatically fetch near the list end. Requests are serialized;
  errors and responses without cursor/item progress retain an explicit retry control instead
  of starting a retry loop. Short pages continue filling the viewport while data advances.
- `Current`: topic bodies and replies link only server-resolved mention occurrences to native
  user profiles. The payload carries numeric identities and UTF-16 source ranges; unknown users,
  escaped text, code, existing links and math remain unchanged. Hidden/deleted bodies expose no
  mention metadata, and persisted Markdown stays unchanged.

- `Current`: a bare HTTP(S) URL in its own Markdown paragraph resolves through the server batch API and
  becomes a compact native preview only when typed metadata is ready; failure keeps the ordinary link,
  and each document stops after five previews. Below 640px, cover images use a 56px cropped thumbnail;
  wider cards preserve the complete cover within a 168px rail, matching Web. Cards and ordinary Markdown links share internal routing
  and external confirmation. The confirmation shows the hostname and selectable full URL, supports
  system back, and scopes optional session trust to the Public Suffix List registrable domain. The card
  is covered at 320 logical pixels, dark mode and 2.0 text scale.
- `Current`: Home announcements render optional titles and HTML bodies, including the legacy
  single-HTML payload. A small bell sits in a separate leading column, with title and body aligned
  to the same inset as Web. They grow with their contents and text size; empty announcements take no
  space. Multiple announcements rotate automatically and expose capsule indicators plus previous/
  next controls when expanded. The banner can collapse to a single-line ticker; the collapsed state
  is shared across the latest, popular and trending tabs. Assistive navigation and reduced motion
  disable automatic rotation. Refresh replaces the active announcement safely.
- `Current`: feed body text uses 17 logical pixels; Markdown reading and publishing body text use
  18 pixels with a 1.55 line height and system text scaling. Code uses 16 pixels and tables use
  17 pixels; headings keep a distinct hierarchy and follow the active theme. The first post supports
  text selection. Feed cards use
  compact vertical padding and one timestamp; embedded Markdown uses smaller paragraph margins
  so short replies do not acquire a large empty footer. Notification rows, conversation rows and
  chat bubbles share the feed's type scale (16 px titles, 15 px secondary text, 13 px timestamps),
  so the messaging surfaces read at the same size as the home feed.
  Conversation dates move below the preview when they would crowd the sender name, including at
  enlarged text sizes. Empty notification content respects the overlaid header and navigation insets.

- `Current`: pushed pages use platform-native transitions on iOS — the system
  Cupertino page transition with the interactive edge-swipe back gesture, so
  secondary pages (topic, course, Wiki, settings) can be swiped closed from the
  left edge. Android keeps the web-mirrored fade/rise transition. Horizontal
  scroll rails keep working; the back gesture only claims the narrow left-edge
  band.
- `Current`: four persistent destinations — Home, Campus, Notifications and Messages — use icon-only
  navigation with accessible labels. Search is a pushed page, reachable from Home. Campus links to
  the native course catalog, scheduler and Wiki; returning preserves the selected destination.
  About links to native friend links, sponsors, terms and privacy pages using the site’s published
  configuration; disabled policies remain hidden.
- `Current`: Home cards retain both images for two-image topics. A portrait single image sits beside
  the text; a landscape image appears below the text with aspect-preserving fit. Portrait galleries
  show up to three columns; two landscape images share a row; larger landscape galleries overlap
  up to three previews with a total count. Tapping the feed card opens the topic; the full gallery
  with zoom is available from inside the topic view.
- `Current`: topic bodies, Markdown and Wiki reading surfaces open the shared image lightbox. It
  supports swipe navigation, pinch and double-tap zoom, actual-size viewing, long-press save and
  system sharing; feed previews deliberately keep their card navigation and do not open the lightbox.
- `Current`: Home topic cards expose compact authenticated like and bookmark shortcuts beside the
  reply/view metrics, and the like metric shows the topic's total like count. Actions switch
  their selected icon and the like count immediately (likes adjust the shown total by one)
  before the request resolves; failures restore the previous state and count and show the
  localized error. Home summaries
  batch-load the viewer's like/bookmark state; absent state (anonymous, unavailable or older
  servers) suppresses the shortcuts. Selected states survive offscreen card recycling, and
  returning from detail refreshes them. In-flight reads cannot overwrite pending or newer successful actions. Likes and bookmarks
  settle independently; switching accounts discards all pending interaction state and reloads the feed.
  Metrics and actions wrap at narrow widths and enlarged text sizes.
- `Current`: simple-content topics show an uncropped, swipeable image gallery above the body. The
  same gallery is used in the publishing preview.
- `Current`: the Home filter rail lists the site's sidebar categories as tappable pills in a second
  row. Pills navigate to their category page; the row collapses when the server publishes no
  categories, keeping the original single-row rail height.
- `Current`: root headers, filter rails and bottom navigation overlay the reading viewport. They
  hide after 48 logical pixels downward and return after 12 pixels upward, with 200 ms transitions.
  Hidden headers are clipped at the system safe-area edge; the reading viewport stays stable.
  Reaching the top, changing destination or opening the account drawer restores the controls.
  Reduced motion removes the transition; keyboard/modal interaction keeps controls visible.
  Editors and scheduler grids are pushed pages outside this behavior.
- `Current`: Home, Campus and Notifications have a compose button that first expands three smaller
  choices: moment, article and question. Their Lucide icons and purple, amber and emerald tints match
  Web. Choosing one opens the corresponding editor; tapping outside, the close button or system back
  dismisses the menu. The menu respects reduced motion and scrolls on short screens.
  Messages has a direct new-chat button. Topic pages keep reply and floor controls in the bottom dock.
  Pull-to-refresh and
  reselecting the active root destination provide refresh and return-to-top without changing icons.
  Refreshable pages accept a short pull from the top on release, including empty lists; the gesture
  uses finger travel so tall screens and iOS rubber-band damping do not demand a longer pull.
  Home, Campus and Notifications show the refresh indicator below their overlaid navigation.
  Small or retracted pulls do not refresh, and an ongoing refresh cannot be started twice.
- `Current`: the floor slider loads the selected server window on release. Earliest/latest
  shortcuts, reply links, and earlier/later pagination navigate the actual reply stream. Returning
  to the first post from a middle window reloads that window before offering refresh; stale
  pagination responses are discarded after a floor or session change.
- `Current`: replies offer a compact sort capsule beside the reply count — oldest first, newest
  first, author only. Oldest and newest flip the loaded window locally without refetching; in
  newest-first order the list footer loads earlier floors and the top control loads newer ones.
  Author-only filters the loaded window to the topic author and automatically scans the remaining
  stream — later windows first, then earlier ones — for at most five windows per automatic scan.
  Loading more continues the search. While windows remain, an empty filtered view invites further
  loading; it only reports no author replies once both directions are exhausted. Switching back
  restores every loaded floor. Sort controls wrap with narrow screens and enlarged text.

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

- `Current`: Home and global search keep the current list after a failed refresh and show a light
  failure notice. Pagination errors remain beside an explicit retry action; retry continues the
  same query/page without discarding prior items. New queries and account changes invalidate old responses.
- `Current`: outgoing chat messages appear immediately as sending bubbles. Failures retain their text
  and expose manual retry without replacing a newer input. Acknowledged bubbles stay visible until
  matched by server history. Existing conversations load their initial server history before enabling
  send; users can keep typing while waiting and retry a failed history load. Offline cached messages
  do not establish this sending boundary. The session-local outbox survives leaving a conversation
  and is cleared at the account/session boundary; it is not persisted across app termination. Only one request for
  each bubble can run at once. The API has no message idempotency key, so ambiguous network failures
  cannot guarantee exactly-once delivery when manually retried.

## Language and presentation

- `Current`: bottom sheets size to short content and constrain long, scrollable content to the
  available viewport. Device safe areas are consumed once: the title starts at the panel's own
  padding, and the panel background extends behind the bottom home indicator. Scheduler pickers,
  course filters, account pickers, Wiki contents, language selection and publishing tools share
  this behavior. Input sheets and confirmation dialogs avoid the software keyboard. Review and
  reply forms allow the whole form to scroll when enlarged text and the keyboard leave too little
  space for the editor and actions; drafts survive resizing. Reply editing still confirms discard
  and prevents dismissal by dragging or tapping outside.
- `Current`: shared form inputs use 16-pixel text. Buttons have a minimum height of 44–56
  pixels by size and grow for wrapped or enlarged labels; disabled actions remain visibly muted.
  Interactive category chips have at least 44-pixel targets. Home, notification and settings tabs
  grow with system text size, and the overlay's content inset uses the same measured height.
- `Current`: empty and retry states share a soft icon surface, readable explanation and optional
  next action, with scrolling on short screens. Empty notifications link back to Home; empty drafts
  open the three-type compose menu; empty conversations retain their new-message action. List
  footers distinguish reaching the end from an empty result.
- `Current`: native launcher icons use Web's YourTJ cat mark. iOS includes opaque device and
  App Store sizes; Android includes legacy densities, adaptive masks and a themed monochrome layer.
  Home reuses the same blue YourTJ cat artwork in a compact square with an accessible brand name.
  Its white background preserves the original colours in both themes; the mark stays centered in
  the header.

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
- `Current`: global search starts with guidance and direct course, scheduler and Wiki destinations.
  Result scope buttons stay available during loading, empty results and failures, and scroll
  horizontally at larger text sizes. Switching scope or retrying uses the last submitted keyword;
  typing a different keyword does not search it until submission. Each result section identifies its
  type and shows displayed rows separately from matching totals; unqueried scopes are not labelled
  as zero, and the all-scope view does not treat the topic total as an aggregate total.
  Users, topics and categories build one row at a time near the viewport. Only topics paginate;
  appending a page retains the other groups, and a failed page keeps the current rows with a retry.
  Course and Wiki search actions carry the current input into the matching native page.
  Recent searches keep up to ten distinct queries per site and account (with a separate guest list),
  in device preferences only; users can clear them. Storage failure does not block searching.
- `Current`: topic view/reply metrics remain below the body; reply, like, bookmark and watch actions
  share one bottom dock. The AppBar shows a generic topic label until the body title scrolls out of
  view, then shows that title. Actions use Web's semantic tints and localized accessible labels;
  the like action includes its count. The reply heading has no decorative discussion icon.
  Topic subscriptions use topic-specific labels; reply commands have no toggle semantics. The dock switches to an accessible icon-only reply action when
  its label cannot fit, including long translations and enlarged text. SVG icons inherit their enclosing button foreground
  unless a semantic or provider color is explicitly set.

## Publishing

- `Current`: publishing uses a type-coloured icon, contextual writing hint and a two-step
  edit/preview indicator above an unframed, multiline title and writing canvas. Classification sits
  in a rounded panel below the preview. All three types share these controls and spacing. Article
  formatting tools remain
  folded in a bottom accessory bar above the software keyboard; expanding them preserves the editor
  selection. The heading tool applies heading 2 with a tap and opens a level sheet on long press that
  offers heading 1–3 (matching the Markdown round-trip); the current level is checked and re-picking
  it clears the heading. The accessory bar holds the draft action and, for articles only, the image
  tool; moments and questions pick images from the compact gallery tile above the body. Rich and
  simple body text use the same mobile reading scale. Preview hides the accessory bar.

- `Current`: reply composers use one rounded surface with a borderless, growing two-line input.
  Image, hide-keyboard, collapse and send actions share the bottom row. The reply target is a
  lightweight text row; attachment previews and server-required captcha controls appear only when needed.
  Typing `@` opens a user suggestion sheet above the software keyboard (reply target, topic author
  and participants first, then debounced server search) with Web-identical token and ranking
  semantics; selecting a candidate inserts plain `@username ` at the caret.

- `Current`: publishing and reply composers have a localized hide-keyboard button that preserves
  unsent text. Dragging the publishing page or topic stream also dismisses the keyboard; opening
  the publishing preview removes editor focus. Rich-text formatting remains available while editing.

- `Current`: the type selector keeps Web's moment/question/article values. Moments and questions
  use a simple gallery plus text; articles use an inline rich editor backed by Markdown. Article
  formatting tools are folded by default. Existing topics retain their type when edited.
- `Current`: simple galleries support up to nine uploaded images, reordering and removal. Images
  survive switching to the article editor. Switching back extracts images into the gallery and
  plain text into the body; conversion is rejected when more than nine images would be lost.
- `Current`: article body images support long-press dragging to any
  paragraph: the image lands below the paragraph it is dropped on, the move
  is a single undo step, and long document drags auto-scroll at the editor
  edges.
- `Current`: Next opens the preview/classification step. The step shows one publish action in the
  AppBar, with the draft action beside it as an icon button. If a long translation or enlarged text
  cannot fit, the next/publish action also uses a labelled icon button; up to three existing
  categories can be selected
  below the rendered image/title/body preview. Publish writes only after this step. The draft action
  saves incomplete forms locally; complete forms can be saved as server drafts, subject to the
  server's title, body and classification requirements. Moments can derive their title from the first text line.
- `Current`: publishing limits, captcha requests and other API failures use the Web error catalog
  in the selected language, including server-provided parameters.
- `Current`: changed editors debounce local recovery saves by 700 ms and flush when leaving or
  the app becomes inactive. Title, Markdown/simple text, type, category IDs and uploaded image URLs
  survive reopening, including when the page metadata request fails. Save progress, success and
  retryable storage failure are visible. Local recovery has one slot per creation entry type and one
  per edited topic; switching type retains the entry's slot. A restored editor still obtains current
  server metadata before publishing.
- `Current`: drafts show separate local and server sections. Local snapshots use app-private device
  preferences scoped by API origin and numeric account ID, with no token or background cloud upload.
  Logging out hides them; logging back into the same account restores access. Explicit discard or
  successful server acknowledgement removes the matching recovery snapshot; local deletion is
  confirmed. Account closure attempts to clear that account's local drafts and searches. Serialized
  writes order deletion after pending saves. Storage failure is reported when saving; OS termination
  before the debounce/flush completes can lose the newest unsaved input.
- `Current`: changed editors offer continue, discard, or save to this device and leave. A server-required
  captcha can be refreshed without discarding content.
- `Planned`: text-to-image cards. No UI claims this feature exists.

## Campus and sign-in

- `Current`: Campus opens a native private overview with the school teaching week, time-aware
  greeting, today's courses and then recent notices. Weekly timetable, academic records and charts,
  calendars, notice bodies and identity management use the existing campus API. Today’s timetable
  uses the server-resolved Shanghai teaching date, including holidays,
  makeup source weeks and explanatory notices; it refreshes across school-local midnight.
  The export-only adjustment switch does not disable this display. GPA is loaded only
  on the academic tab. The timetable shares the planner renderer without its editing or storage.
  The Campus bottom destination opens this page directly. Course reviews, the scheduler and Wiki
  have visible shortcuts at the top of its home view, also available to guests, unbound users and
  when school services fail. Pushed tools return to the Campus destination. Explore campus retains
  public course previews; shortcuts are shared with search discovery. See [campus semantics](campus.md) for binding, privacy and provider limits.
- `Current`: school authorization uses the current native forum session in a restricted WebView.
  The initial Bearer header goes only to the first-party session handoff; school navigation receives
  no native credential. The server callback returns to a native confirmation, including resuming
  a notice after a permission update. Leaving the campus tab drops its private view and cancels
  requests. Selected overview datasets have a five-minute foreground memory cache, reusable only
  after fresh binding-status verification; grades and notice bodies remain page-local.
  Backgrounding, session/site changes and identity invalidation clear the cache. Pull-to-refresh
  keeps same-identity content visible while loading; failures show errors instead of stale results.
  School-local date rollover invalidates teaching-day data. Nothing enters persistent/offline storage.
  See [campus retention rules](campus.md).
- `Partial`: native school login on a physical device is not end-to-end verified. Automated tests
  cover navigation policy, session handoff, confirmation, stale responses and native rendering.
- `Current`: the scheduler opens in course selection. Plan preview remains a local planning
  grid, with week filters, conflicts, custom blocks and existing plan operations. Web and mobile
  warn about time conflicts before a teaching class is selected, while keeping the add action
  non-blocking. Completed term/grade/major selection collapses into an editable summary; course,
  credit, hour and conflict counts wrap in a compact row. A small Web action opens
  the full [Web scheduler](https://f.yourtj.de/schedule) in the external browser without transferring
  the native credential. Plans are not official enrollment results.
- `Current`: signed-in plans use the same per-plan revision and three-way merge rules as Web
  (`GET/PUT/DELETE /api/pk/plan-items`). Independent course changes and custom-event fields merge
  automatically; only conflicting values require a choice. A remotely deleted plan with local edits
  can be kept as a device-only recovery draft and restored under a new ID, outside cloud quota until
  restoration. Current plan, major selection and week view are device-local. Each account retains its
  own cache and merge bases; guest content needs explicit adoption. Changes debounce for 3 seconds,
  dirty network failures back off up to 60 seconds, foreground/network restoration flush pending
  edits, and focus reads are throttled to 30 seconds. Clean state has no polling timer.
  Existing cloud snapshots migrate intact on first use; legacy clients receive 410 afterward.
  Account closure erases cloud content and prevents in-flight requests from recreating it.
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
- `Current`: sign-in offers account/password, Google, GitHub and Tongji when the published options
  allow it. Password captcha and TOTP remain
  supported. The login captcha stays folded until the password field is first interacted with;
  the first password focus/input warms the challenge, and a blank outside tap or genuine secure-IME
  dismissal reveals it without taking focus from another explicit control. That reveal is latched through transient Android
  focus rebounds, and a prefetch failure stays silent until the visible retry path is used. On
  Android, auth-field pointer-down creates a short-lived target token; if the secure keyboard
  reclaims the password focus during that token's settling window, the app makes at most two
  bounded attempts to return focus to the explicitly tapped field and then stops. A focused field
  also has a finite view-insets-based IME show watchdog. Dismissing an already-visible secure
  keyboard releases password focus and is honored as user intent; a transient hidden IME during an
  explicit password-to-username/captcha handoff remains recoverable. Blank-space and button taps
  create no focus target and do not start a focus battle. Captcha pixels are left unchanged in light mode
  and use the Web-equivalent dark-mode transform. Google availability follows the published Web
  configuration. On Android, Google/GitHub/Tongji all use one RFC 8252 external-system-browser
  flow with manual PKCE/state/nonce and the native MainActivity callback bridge; the Android path
  intentionally bypasses flutter_appauth/AppAuth/CustomTabs. No OAuth provider uses a WebView for
  Android login. Non-Android platforms retain AppAuth. `Partial`: the new Android path awaits a
  physical-device APK test; the exact native crash stack remains unproven without logcat.

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
- `Current`: Settings' nickname and bio entries edit only their named field. Avatar upload and the
  twelve presets share one source picker. Website/social links have their own entry; full profile
  editing retains signature and profile language.
- `Current`: profile editing includes nickname, bio, signature, website name/URL, profile language
  and the six Web social providers. Saving preserves unedited fields and unknown social providers;
  website/social destinations accept HTTP(S), and social usernames expand to provider URLs.
  Public profiles display website/social links with the corresponding provider marks and open
  them in the system browser. Worn badges appear on the avatar independently of the badge list;
  administrator identity has a localized role label. Returning
  from settings refreshes profile identity and media immediately.

- `Current`: the root avatar opens an account drawer with a generous left inset, larger line icons,
  nickname and account handle. Following/follower counts come from the user's card and open the
  matching profile streams. Unavailable counts show a placeholder with retry instead of zero; opening
  the drawer refreshes the card, and account changes discard previous identity data. Profile,
  bookmarks, drafts, my content, recycle bin and my course reviews are direct entries. Settings and
  permission-gated workspaces remain available. The profile overflow retains its infrequent entries.
  Account controls are outside the public profile.
- `Current`: activity entries distinguish signup, post, like, follow and comment with matching
  icons and localized captions in bordered cards with a content preview and compact timestamp. Stream changes retain the profile header collapse, limiting deep offsets to the start of the
  new stream so loading, empty states and retry actions stay visible. Empty badge lists use
  badge-specific feedback.
- `Current`: profile bios trim boundary whitespace; signatures use a separate feather mark and subtle
  underline. Avatar overlap participates in layout so it leaves no translated blank space. The role
  label stays beside the name; earned badges appear as bordered title/description cards with colored
  hexagons and their server-provided SVGs. The selected badge remains attached to the avatar.
  Settings allow selecting and ordering zero to five owned, enabled badges for the profile header.
  An explicit empty selection hides that row; existing accounts default to their first five badges.
  This selection does not change the avatar badge or the complete earned badge collection.
  Profile body text uses 16 pixels; statistics prioritize the values and wrap into fewer columns on
  narrow screens or at large text sizes. Settings groups use rounded inset surfaces, multiline row
  labels and consistent trailing arrows; avatar upload copy describes image selection and cropping.
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
  On iOS the first launch after login requests the system permission once; granting it counts as
  push consent and enables delivery without visiting Settings. Android keeps explicit opt-in —
  the JPush SDK is never initialized before the user enables push. Explicit enable requests system
  permission. Resume checks existing authorization without repeatedly prompting; a non-empty token
  and successful API registration are required to display enabled.
  Enable taps during startup/resume or stop are queued; a later disable or account change cancels
  queued consent. Failed unbinding is retained and retried on resume while push stays disabled,
  using the owning account; signing in to a different account does not acknowledge that cleanup.
- `Current`: notifications use the existing event copy and only navigate to supported in-app topic,
  profile and notification routes. Logout stops native delivery and attempts server unbinding;
  the next account requires fresh consent. Optional JPush analytics/location collection is disabled.
- See [activation and device validation](../operations/mobile-releases.md#native-push-activation-and-verification)
  for credentials, supported OEMs and delivery limitations.

## Tongji sign-in

`Current`: native login and registration show “Tongji SSO” when the public login options declare
campus configuration ready. The entry explains automatic activated registration and links published
policies. The backend handles the school callback and resumes the same manual PKCE/nonce exchange
used by the Android external-browser path; the App stores only its forum session, never a school
access/refresh token. Existing bindings sign in to the same forum account; new users receive a
private student-ID@tongji.edu.cn email without a separate activation step. All four UI languages are
supported. Tongji shares the exact MainActivity-owned `yourtj://callback` bridge with Google and
GitHub; AppAuth's Android receiver does not claim it, and no WebView is used for this OAuth login.
`Partial`: physical-device school sign-in has not been validated with the new APK.

## Private user notes

`Current`: User profiles provide a private-note editor with retry and clear behavior. Names in topic
lists, replies, profile connections, search, conversations, notifications, mention candidates and
revision history use `note(username)` for the current viewer. Notes are fetched through the shared
core contract and remain only in a session-scoped memory provider; changing account invalidates
pending responses and never reuses notes from the offline forum cache. Limits and account-erasure
semantics are defined in [Identity and access](identity-and-access.md#private-user-notes).

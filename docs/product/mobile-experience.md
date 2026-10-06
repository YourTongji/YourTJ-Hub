# Mobile experience

> Doc type: product spec
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-10-03

The Flutter app combines the forum, course catalog, scheduler and Wiki. Ordinary browsing and
writing use native pages. Management uses the same first-party workspaces and permission checks as
Web inside an authenticated in-app browser. The navigation and management boundary are recorded in
[0012](../decisions/0012-unified-mobile-reading-navigation.md).

The released app is available from the [YourTJ download page](https://yourtj.de/#download):
iPhone/iPad through the App Store and Android through public APK downloads. See
[distribution and updates](#distribution-and-updates) for channel status and remaining device evidence.

The [interaction and layout standard](mobile-design-system.md) defines the shared visual and
behavioral acceptance rules. Its `Planned` requirements are tracked separately from the implemented
behaviors below; [state and cache boundaries](../architecture/mobile-state-and-cache.md) describe the
corresponding planned ownership and lifecycle contracts.

## Launch experience

`Current`: Android and iOS display a static YourTJ brand mark in their native launch surface.
Android selects light/dark launch resources using the saved app appearance; Flutter continues with
the resolved app theme, a transparent animated mark and the caption “未济非终，皆有可能”. The
animation yields to the interactive app within 1.2 seconds and is skipped for reduced motion.
Session and Home requests run beneath the launch surface. Home mounts its base sort tabs before
server navigation arrives, then fills the existing skeleton. Initial cached/network rows prefetch
at most four author avatars with a 240-millisecond ceiling using the same image keys as the visible
avatars. Avatar decode sizes snap to a ladder (24/40/48/64/96 logical pixels; larger avatars round up
to the next 16-pixel step), so the same author rendered at nearby sizes — feed card, detail header,
reply row — reuses one in-memory image entry per URL instead of downloading and decoding it again on
each page switch. Request and account guards still apply after prefetching. Startup system-bar styling is local to the launch surface; the normal
route-aware fallback resumes when the launch surface leaves.

## Navigation and reading

`Current`: Home negotiates feed capability v2, keeps explicit Latest separate from the default root,
restarts expired recommendation snapshots and reports foreground visible rows/detail dwell. Attribution
traces stay in memory and are stripped from offline pages. Recommendation reasons use four localized
labels. See [feed ranking and measurement](feed-ranking.md) for the shared behavior and retention.


`Current`: root layout uses the available window width. Below 600 logical pixels it retains bottom
destinations; at 600 and above it uses a persistent, scrollable 72-pixel navigation rail. Forum,
notification and conversation lists occupy a centered column up to 720 pixels wide. Campus can use
1120 pixels for its timetable and tools. Wide layouts reclaim the bottom-navigation inset, keep
compose actions inside the content column and anchor their menu to that column. Resizing preserves
the retained branch navigator, inputs and reading position; opening a keyboard does not change the
width breakpoint.

`Current`: the rail shares destination icons and unread state with the bottom bar. Each action
exposes its name, selected state and activation in one semantic node. Persistent navigation is
ordered after the active route in the accessibility tree so iOS does not hide it behind that route.


- `Current`: Home offers a server-defined Following sort. It requires sign-in and shows only
  currently followed authors' public forum topics, newest creation time first with descending
  topic ID for ties. Pagination uses an opaque cursor in `nextUrl`; edits, replies and pinning
  do not reorder it. After following or unfollowing from a profile, pull to refresh Following
  to replace retained rows and start from the newest matching topics. Continuation requests
  already exclude unfollowed authors, while newly followed content above the cursor appears
  on refresh. An empty follow list stays empty; guests are directed to sign-in.

- `Current`: paginated feeds, search, notifications, profiles, content management, own course
  reviews and post history automatically fetch near the list end. Requests are serialized;
  errors and responses without cursor/item progress retain an explicit retry control instead
  of starting a retry loop. Short pages continue filling the viewport while data advances.
- `Current`: each visited Home sort retains its own loaded topics, pagination cursor, scroll
  position, loading and retry state. Switching sorts keeps the filter rail available during
  loading; late responses update only the sort that requested them. Returning to a visited sort
  resumes it without refetching. Hidden sorts pause automatic pagination until selected again.
  Session changes discard all retained feeds.
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
  next controls when expanded. The banner starts as an expandable single-line ticker; its state
  is shared across the latest, popular and trending tabs. Assistive navigation and reduced motion
  disable automatic rotation. Refresh replaces the active announcement safely.
- `Current`: feed body text uses 17 logical pixels. Post Markdown, server-rendered Wiki HTML and
  course-review HTML all derive their reading typography from one shared rich-content profile rather
  than from per-surface hard-coded sizes: body text keeps the 17-pixel design baseline, headings are
  relative ratios (H1–H4 ≈ 1.45/1.30/1.18/1.08 × body at the design size; as the painted body grows
  with the system font scale and reading size the levels move closer, down to 40% of the extra size
  at 2×, and windows under 360 logical pixels tighten them one more step), inline code and code blocks are one step
  smaller, tables inherit the body size, and course reviews use the same profile at a compact
  ~15.5-pixel baseline. The first post supports text selection.
- `Current`: Settings → Appearance → Text size opens a live preview page with two layered sliders
  in 10% steps. App text (90%–130%) scales every text in the app, rich content included; reading
  text (80%–140%) then adjusts only rich content bodies — posts, Wiki and course reviews — on top of
  it. The preview is a small app screen built from the real components: tab bar, feed row, a post
  with a Markdown body (heading, bold text, list), action row with a button, and bottom navigation.
  It updates, together with the app behind it, while either thumb is dragged. Dragging app text
  outlines the whole preview; dragging reading text outlines the post body, dims everything else and
  scrolls the body into view, and the highlight fades shortly after release. The control panel stays
  at the default size so the slider never moves away from the finger; Reset to default restores both
  values and is disabled at the default.
- `Current`: 100% is a device-adapted default rather than the raw design size. Design sizes are iOS
  points; Android starts at 16/17 of them, matching its 16sp reading body, and windows whose
  shortest side is under 360 logical pixels take one more 5% step. Rotation never changes text size.
  The default, both preferences and the system font scale multiply; the system scaler applies last
  instead of being replaced or clamped. Repository code never pins `TextScaler.noScaling` or a fixed
  text scale factor.
- `Current`: fenced code and server-rendered `<pre>` blocks render through one shared code block with
  syntax highlighting (light and dark themes), a language label, a copy action, and horizontal
  scrolling inside the block itself; unknown languages or a failed highlight fall back to plain
  monospace text. Wide Markdown and server-HTML tables keep natural column widths and scroll inside
  their own region, so the page itself never scrolls sideways and an outer card cannot swallow the
  gesture. Wiki bodies keep consuming the server `rendered_html` (heading ids, table of contents
  anchors and relative URLs unchanged), and course reviews render the API's `contentHtml`, so the
  server stays the only place that normalizes legacy review headings. Feed cards use
  compact vertical padding and one timestamp. Author, time and category labels share one metadata
  row. Long author names ellipsize, and the category group scrolls horizontally when space is tight,
  retaining separate touch targets.
  Metadata is vertically centered in its 44-pixel targets, aligned near the avatar top without
  overlapping the title below. Titles use a compact 1.35 line height with a 2-pixel gap before the
  excerpt. A 2-pixel gap leads into the action row, followed by a 4-pixel bottom inset, keeping
  consecutive posts close while preserving separate touch targets.
  Embedded Markdown uses smaller paragraph margins
  so short replies do not acquire a large empty footer. Notification rows, conversation rows and
  chat bubbles use 16 px body text; conversation and notification rows use 16 px titles, 15 px secondary text and 13 px timestamps,
  so the messaging surfaces read at the same size as the home feed.
  Conversation dates move below the preview when they would crowd the sender name, including at
  enlarged text sizes. Empty notification content respects the overlaid header and navigation insets.

- `Current`: pushed pages use platform-native transitions on iOS — the system
  Cupertino page transition with the interactive edge-swipe back gesture, so
  secondary pages (topic, course, Wiki, settings) can be swiped closed from the
  left edge. Android keeps the shared mobile fade/rise transition. Horizontal
  scroll rails keep working; the back gesture only claims the narrow left-edge
  band.
- `Current`: four persistent destinations — Home, Campus, Notifications and Messages — use an
  icon-only bottom bar on compact windows, with the selected Filled icon, indicator and unread state
  kept together at a fixed compact height, so labels never add height or grow with text size. Each
  destination keeps its localized name for screen readers and shows it as a long-press tooltip.
  The iOS bottom bar uses a restrained Flutter blur and translucent surface over
  the existing theme; Android, high-contrast mode, reduced motion and accessible-navigation mode
  use the opaque theme surface. This is a Flutter material treatment and does not adopt a native
  iOS Liquid Glass API. Search is a pushed
  page, reachable from Home. Campus links to the native course catalog, scheduler and Wiki;
  returning preserves the selected destination.
  About links to native friend links, sponsors, terms and privacy pages using the site’s published
  configuration; disabled policies remain hidden.
- `Current`: Home cards retain both images for two-image topics. A portrait single image sits beside
  the text; a landscape image appears below the text with aspect-preserving fit. Portrait galleries
  show up to three columns; two landscape images share a row; larger landscape galleries overlap
  up to three previews with a total count. Tapping the author avatar or name opens the native
  profile; tapping text opens the topic. Tapping a preview opens the shared lightbox at that image
  with every topic image available, including images beyond the feed preview limit. Author
  targets and previews support keyboard activation; previews announce their localized image
  position, and both author targets have at least 44-by-44 logical-pixel touch areas.
- `Current`: topic bodies, Markdown and Wiki reading surfaces open the shared image viewer. It
  supports swipe navigation, pinch and double-tap zoom, actual-size viewing, long-press save and
  system sharing. Multi-image viewers show a horizontally scrolling thumbnail rail with a centered
  focus; selecting a thumbnail changes the image and resets zoom and actual-size mode. Distant
  selections jump directly; adjacent selections use the shared media cadence. Tapping
  toggles the viewer controls and rail together; a vertical drag dismisses at minimum scale, moving
  and scaling the image while the background fades. At minimum scale, horizontal paging uses the
  image page view's horizontal drag recognizer and cumulative pointer displacement after device
  touch slop; velocity is reserved for release physics, so a quick new swipe can interrupt the
  previous page settle immediately. Vertical dismissal starts only after at least 1.5 touch-slops
  of travel with 1.5:1 vertical-to-horizontal dominance. A later, clearly horizontal movement can
  hand a provisional dismiss drag back to paging. A new touch can interrupt a slide-return animation; clear
  horizontal intent returns any partial vertical offset to rest and pages immediately. Dismissal
  accepts a light drag (one tenth of the viewport height) or a quick vertical flick (420 logical
  pixels per second) on release. A cancelled drag or a second finger joining the gesture returns
  the image to rest without dismissing the viewer, even beyond that distance threshold. Zoomed
  images keep gestures for image navigation and panning.
  Paging and dismiss gestures stay out of the way while an image is zoomed. Changing reduced motion
  while viewing keeps the current image and settles active zoom or return animations. Home feed
  previews use the same viewer and image actions.
- `Current`: Home topic cards expose compact authenticated like and bookmark shortcuts beside the
  reply/view metrics. A single heart action includes the topic's total like count; both actions
  retain a minimum 44-by-44 logical-pixel touch target while their icons animate. Actions switch
  their selected icon and the like count immediately (likes adjust the shown total by one)
  before the request resolves; failures restore the previous state and count and show the
  localized error. Home summaries
  batch-load the viewer's like/bookmark state; absent state (anonymous, unavailable or older
  servers) suppresses the shortcuts. Selected states survive offscreen card recycling, and
  returning from detail refreshes them. Loaded Home sorts share interaction updates. In-flight
  reads cannot overwrite pending or newer successful actions or newer state returned from detail.
  Likes and bookmarks
  settle independently; switching accounts discards all pending interaction state and reloads the feed.
  Metrics and actions wrap at narrow widths and enlarged text sizes.
- `Current`: simple-content topics show an uncropped, swipeable image gallery above the body, with
  an ambient blurred image backdrop, a compact count badge and page indicator. The same gallery is
  used in the publishing preview and opens the shared full-screen viewer from any image.
- `Current`: Home displays categories in a horizontal row below the feed sorts. Category pills
  filter the existing stream in place, with a highlighted selection and an All categories action.
  The display menu contains list/card preferences; unavailable categories take no space.
- `Current`: list mode uses a compact title/type row, optional excerpt, and a shared category/metadata
  band. Categories retain separate 44-pixel targets and a horizontal rail; long metadata moves the
  rail onto a new line rather than reducing text size. Dividers do not add a blank footer. Question,
  moment and article labels are localized, and enlarged text allows the title/excerpt to grow.
  Home's unfiltered stream, in both list and card mode, folds pinned topics into one initially
  collapsed announcement-style line: pin mark, "Pinned" badge, the first pinned title and, with
  several pins, their count. A single pin opens directly; several unfold into one-line titles led by
  a small author avatar (with unread dot and relative time) inside the same strip, each opening its
  topic. At large text the badge and time yield to the title. Category streams and Following retain
  the server's ordering.
  Expanding pins neither reloads the stream nor changes its pagination cursor.
- `Current`: Home sort, Campus section and notification filter rails share a scrollable tab bar.
  The page-swipe recognizer feeds the shared tab controller's live drag offset, so the selected
  underline follows a held slow swipe and stretches evenly toward the adjacent tab. After release,
  the extended segment contracts with a logarithmic ease-out curve. Home, Campus and Notifications
  move both content panes with the same drag or tap transition while retaining each visited page's
  elements and scroll state. A pane drag reports the touch slop it travelled before winning the
  gesture arena, so the page stays under the finger; release settles with a spring that keeps the
  swipe's velocity, and a fling back toward the origin cancels the switch. A pending pane uses the
  transparent animated YourTJ mark until its data is ready.
  Profile stream tabs use the same drag progress; the bar reveals selected tabs outside its viewport
  and honors reduced-motion settings.
- `Current`: root headers, filter rails and bottom navigation overlay the reading viewport and
  follow the finger: they slide out over 64 logical pixels of downward scroll, any upward scroll
  pulls them back, and a partial state settles in the last direction within 180 ms once scrolling
  stops. Near the top they stay attached to the content; edge bounce is ignored. Content insets
  never change, so the feed does not jump. Hidden headers are clipped at the system safe-area edge;
  the compose button follows the bottom bar. Header and bottom-bar borders are one physical pixel.
  While controls are hidden, off-screen tab panes scroll their reserved header band away, so a pane
  swiped or tapped into shows content rather than a blank band; the pane on screen never moves and
  returning controls restore those offsets. Reaching the top or changing destination restores the
  controls; the account drawer overlays the page and leaves their state unchanged.
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
- `Current`: a submitted reply is acknowledged with a full window around its new floor, and the
  created reply itself is scrolled into view. When that window adjoins the loaded floors the reply
  is merged by server ID: loaded replies stay, the earlier cursor keeps the loaded window's top
  while the later one extends to the new window's tail, and no floor is duplicated. A distant new
  floor replaces the window (web `revealCreatedPost` merges more permissively; a linear cursor pair
  cannot represent the gap) and still reveals the reply. A reply held for review is filtered out of
  every window for non-moderators, so the app keeps the loaded floors and cursors instead of
  clearing them; the reply appears once approved. Deep-link windows — such as a notification
  pointing at one reply — keep both continuation controls on the anchored floors.
- `Current`: replies offer a compact sort capsule beside the reply count — oldest first, newest
  first, author only. Entering newest-first fetches one bounded tail window through the existing
  post-window endpoint — including on deep-linked topics — and refreshing the default window does
  the same, so both the first page and deep links reach the latest floor once newest-first is
  chosen; the list footer loads earlier floors and the top control retries a failed tail request or
  loads newer floors. Oldest-first keeps the loaded window locally.
  Author-only filters the loaded window to the topic author and automatically scans the remaining
  stream — later windows first, then earlier ones — for at most five windows per automatic scan.
  Loading more continues the search. While windows remain, an empty filtered view invites further
  loading; it only reports no author replies once both directions are exhausted. Switching back
  restores every loaded floor. Sort controls wrap with narrow screens and enlarged text.

- `Current`: tapping a topic-author or non-anonymous reply avatar opens a top-docked profile
  preview; tapping the name still opens the public profile directly. The preview matches Web's
  identity, status, bio, signature, up to five badges, four statistics, up to eight public links,
  join date and follow/profile/message actions. The message action opens that user's conversation;
  block/unblock stays synchronized with the shared block list. The compact card stays near the
  previous full-height layout, slightly inset from the available safe area; overflowing content
  scrolls internally with a visible scrollbar. The action controls, badge, statistics, date and
  social-link rows stay single-line on narrow screens, with a horizontal scroll cue when needed.
  The nickname sits directly below the overlapping avatar. Signature stays
  directly below the bio with the profile's feather and wave treatment; social links use compact
  44-pixel targets. The private-note edit action sits beside the handle, clear of status badges;
  the trailing more menu switches between block and unblock from the shared block list, and a
  successful change refreshes that state for all consumers. Profile and more actions share a
  theme-derived, translucent outline; the more action uses a circular 44-pixel target. The full
  5:1 cover stays sharp as a horizontal banner and softly fades at its
  lower edge into a full-card, cover-derived blurred color field. A theme-tinted veil keeps text
  legible without flattening the lower card to a solid fill; light-mode admin and online badges
  use darker semantic foregrounds on a translucent, lightly rimmed base surface for contrast across
  the blur boundary.
  The ReIcon close control honors reduced motion. Profile data is cached for 60 seconds
  within the active session, then
  refreshed in the background.
  Late reads cannot replace a newer read or follow action. Closed accounts show the closure notice,
  and legacy per-content anonymous reply avatars do not open a profile. Persistent anonymous personas
  open their independent `/a/:publicUid` profile. Profile pages, relationship lists and previews
  share one optimistic follow state per user; failure restores it and the next profile-card read
  refreshes follower counts.
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
- `Current`: profiles use a 3:1 cover (112–200 logical pixels tall), an overlapping avatar,
  trailing edit/follow/message actions, distinct name and handle, readable bio and inline statistics.
  Following and follower counts open the corresponding lists. Extra account tools, including course
  reviews, remain in the profile menu. Profile and connection headers identify the viewed person.
  Connection rows show a 48-pixel avatar, name, handle and up to three bio lines, with a pill follow
  action. Narrow layouts and enlarged text move the action below the bio. Follow actions serialize
  per person, update immediately and roll back on failure; late reads cannot overwrite local actions.
  Reads started after a settled mutation reconcile remote changes across retained connection tabs.
  Session changes clear pending relationship state. Self rows and old-server rows without relationship
  state omit the action; guests are directed to sign-in and return to the viewed list before following,
  without automatically replaying the action.
- `Current`: notification entries use a small event glyph (pink heart for likes), the actor's
  avatar, a bold actor name within the localized action, inline time and a muted three-line preview.
  Avatar URLs are resolved in a server batch; likes without a stored preview use the visible reply excerpt.
  Actor avatars open the profile independently of the notification's read action. Unread dots,
  acknowledged-read updates and failure retries remain available.
- `Current`: notification headings resolve the same template keys and event types as Web in the
  selected language, with actor names and topic/content previews. Legacy literal headings take precedence when no template key is present; content previews take
  precedence over topic titles. Protocol-key filtering applies only to heading fields, preserving
  user content and badge names that begin with the same prefix.

- `Current`: Home and global search keep the current list after a failed refresh and show a light
  failure notice. Pagination errors remain beside an explicit retry action; retry continues the
  same query/page without discarding prior items. New queries and account changes invalidate old responses.
- `Current`: Notifications also retain rows after a failed refresh. Filter changes and account
  generations reject older refresh/pagination responses. Pagination deduplicates IDs and pauses with
  explicit retry after failure or a response without progress. A failed refresh preserves that pagination
  error and pause; a successful refresh or explicit retry resumes loading. Single/all-read actions are serialized,
  display pending state and surface failures; rows remain unread until acknowledged. Confirmed reads
  cannot be reverted by an earlier fetch. Single-read failures retain a row-level retry action until
  a successful action or refreshed server state confirms the read; the
  unread filter removes acknowledged rows and continues pagination when its visible page is drained.
- `Current`: outgoing chat messages appear immediately as sending bubbles. Failures retain their text
  and expose manual retry without replacing a newer input. Acknowledged bubbles stay visible until
  matched by server history. Existing conversations load their initial server history before enabling
  send; users can keep typing while waiting and retry a failed history load. Offline cached messages
  do not establish this sending boundary. The session-local outbox survives leaving a conversation
  and is cleared at the account/session boundary; it is not persisted across app termination. Only one request for
  each bubble can run at once. Retries reuse the same client message ID, so an ambiguous network
  failure does not create another stored message when the same outbox entry is retried.
- `Current`: unsent private-message text and caret/selection are kept per peer in app-private device
  secure storage, scoped by API origin and numeric account ID. Conversation rows show a localized draft
  preview, including new peers without a server conversation; list search also matches draft text.
  Unresolved new-peer rows remain visible but cannot open until the server conversation list succeeds;
  a resolved existing conversation still waits for its initial history before enabling send.
  Input remains editable during sending. A successful acknowledgement clears only the submitted
  revision, while newer input and failed sends remain available. A failed send rehydrates the
  submitted composer snapshot (text, sticker tokens, newlines and caret) under its submitted revision
  unless the user kept typing (whitespace included) while the request was in flight, or a newer
  message was already acknowledged; the restored draft therefore still clears on acknowledgement and
  a re-send reuses its outbox bubble and client message ID instead of delivering the content twice.
  Saving debounces for 500 ms and flushes on leaving or app inactivity;
  failures keep the current text in session memory with visible retry. No message is sent by autosave.
  Signing out hides drafts and invalidates pending saves; the same account/site can restore them on
  its next session; accepting a same-site login recreates the draft registry for the new identity.
  Account closure attempts to remove that account's local writing. Drafts contain no credential and
  the app does not upload or synchronize them. On iOS, a dedicated Keychain service uses
  `AfterFirstUnlockThisDeviceOnly` with synchronization disabled: items cannot migrate to another
  device, although same-device backup restoration is permitted. Android keeps namespaced draft keys
  in the existing secure-storage file and excludes that file, its wrapped-key preferences and the
  legacy Flutter preferences file from cloud backup and device transfer. This also excludes other Flutter preferences (such as
  theme/language) and secure credentials in those files from system migration.
  Legacy plaintext chat records are copied for all stored accounts and read back before removal;
  only the active account's records are exposed. A failed migration retains the original and shows
  retry, while a secure deletion marker prevents stale legacy text from resurrecting. Previously
  created OS backups cannot be retroactively erased by the app. Android's plugin enumerates the
  shared encrypted store before account filtering; unreadable ciphertext, including an unrelated
  record, can prevent draft restore/save until the storage error is resolved. The app retains the
  current text and legacy copies with a retry message; it never resets the secure store or deletes
  unrelated credentials to recover. See
  [Android backup rules](https://developer.android.com/identity/data/autobackup) and
  [Apple device-bound Keychain behavior](https://developer.apple.com/documentation/security/ksecattraccessibleafterfirstunlockthisdeviceonly).
  OS termination before a successful save can lose the latest edits; the outbox's separate session-only
  retention and ambiguous-retry limitation remain.
- `Current`: chat text, including sending, acknowledged and failed outbox bubbles, supports native
  selection/copy and underlined HTTP(S) links using the shared
  internal-routing/external-confirmation policy. Holding a message bubble opens the action menu
  (reply, whole-message copy, saving resolved stickers to the collection, and report for received
  messages) without interrupting drag-scroll or drag text selection; a plain tap does not open it.
  Messages containing only resolved stickers and
  whitespace render without a colored bubble or bubble padding, for both incoming messages and
  outgoing/history/outbox messages. History stickers open the same message menu, which stays
  reachable while a sticker image is loading or unavailable. Outbox stickers retain their own
  collection and failed-image retry actions until the message appears in history.
  Time and delivery state remain available.
  Mixed text and unknown or disabled sticker tokens retain the normal bubble. Inline stickers
  remain supported; chat text is not
  interpreted as Markdown or HTML. The selection menu also offers whole-message copy, preserving
  sticker tokens that partial native text selection omits; the action menu offers the same copy.
  Choosing reply shows a dismissible sender/excerpt preview above the composer until the message is
  sent, the preview is cancelled, or the session changes. The quote names its author
  (`@username`, or a localized self label when the page payload carries no viewer username); a
  failed send re-attaches the quote only while its draft and reply selection are unchanged, so
  sending again retries the same entry instead of posting an unquoted duplicate. A successful
  bubble retry clears that draft's quote without discarding a newer reply selection. Activating a
  quote reference reveals the lazy-loaded target with the normal short easing; reduced motion jumps
  directly after the target is mounted and settles the highlight through the shared motion policy.
  The emoji accessory replaces the current
  selection and leaves the caret after insertion. Replacing the draft with text that has no valid
  selection resets insertion to the end. Opening it dismisses the software keyboard and keeps focus
  inside the composer for hardware shortcuts; the keyboard control restores
  focus. The accessory only moves focus and bounds the input's rendered height; the field stays
  multiline, so mobile return keeps inserting newlines with the panel open, after a sticker is
  inserted and after one is deleted, without leaving the page. Its bounded scrollable grid has
  touch-sized controls, localized labels and system-back/Escape
  dismissal. Hardware Ctrl/Cmd+Enter sends. Disabling the composer
  also disables emoji edits. Platform IME transitions still require physical-device verification.
- `Current`: the private-conversation header avatar and each incoming message avatar open the peer's
  profile (`/u/{peerId}`). Both keep their 36/32-pixel artwork and its top-left anchor and reserve a
  44 × 44 hit area around it. The header title keeps its exact position; an incoming message bubble
  starts 4 logical pixels further right with 4 fewer logical pixels of available width, while the
  outgoing side stays unchanged, so scrolling, text selection, link taps and the report action are
  unaffected. Each avatar is its own
  screen-reader node, labeled with the localized "view profile" action and the peer's display name;
  the message text stays a separate node and is not announced as a button. The outgoing (own) avatar
  stays display-only, and the conversation-list avatar keeps opening its conversation rather than a
  profile. The web conversation exposes the same peer entry point on the header name instead and
  keeps every avatar display-only.
  `Partial`: physical-device tap feel and screen-reader verbosity on device still need validation.
- `Current`: native conversations acknowledge only incoming, unread server message IDs whose actual
  bubbles are at least 50% visible for a stable 350 ms in the message viewport. For a bubble taller
  than the viewport, visibility uses the viewport height. The keyboard-clipped viewport, current
  route and ancestor navigator routes, active tab, foreground lifecycle and session epoch all gate
  measurement. List prebuilding, opening a conversation, fetching messages and intermediate positions
  during a jump do not establish read state. Batches contain at most 100 IDs with one request in
  flight; stale callbacks cannot update the next session. A transient failure has one automatic retry
  and an explicit retry, preserving unread state. Unsupported servers show a compatibility message
  and never fall back to the whole-conversation read endpoint.
- `Current`: new incoming messages preserve the user's history position and expose an accessible
  lower-right jump-to-latest button. Jumping only acknowledges bubbles actually visible after layout;
  unseen history remains unread. Loading older pages preserves the visible bubble anchor across lazy
  relayout, and newer fetches retain the older-history cursor. `Partial`: physical-device visibility
  thresholds, keyboard overlays and lifecycle behavior still require device validation.
- `Current`: the authenticated Flutter shell keeps one chat/notification/unread event connection only
  while foregrounded. The server sends an immediate resync instruction and owner-scoped change hints;
  the app reloads actual messages, notification lists and unread badges through REST. Reconnects and
  resumed sessions reconcile again, and a failed or unsupported stream uses foreground polling until
  delivery recovers. Account changes cancel the previous connection and discard stale unread responses.
  Background push delivery is not provided by this stream.

## Motion and continuity

`Current`: native mobile motion uses one semantic policy: 120 ms press feedback,
160 ms selection, 180 ms content, 220 ms layout and 280 ms sheet/media transitions.
Root headers, navigation and compose controls move together while the reading viewport stays
stable. Tabs, announcement expansion, form focus, theme/logo changes and local disclosures
consume that policy. Android pushed pages use a 220 ms fade with a six-logical-pixel rise and
180 ms return; iOS retains Cupertino navigation and its cancellable edge-swipe gesture.
Native drawers, scrolling, refresh and direct image gestures retain platform behavior.

`Current`: feedback banners enter and leave softly; a replacement crossfades in the same overlay
and cancels an older pending dismissal. A dismissed banner stops accepting input before removal.
Feed and topic-dock toggles use the same single, at-most-six-percent icon pulse on explicit
activation. Passive data updates do not replay it. Actions and network requests start immediately,
and optimistic state and failure rollback remain owned by their existing feature state.

`Current`: reduced motion removes custom transition durations, announcement expansion and rotation,
icon pulses, programmatic image paging/zoom and reading-position movement. Shared indeterminate
progress becomes a static glyph with loading semantics and no fabricated completion percentage; real upload
progress remains determinate. Enabling reduced motion during a pulse or banner exit settles it;
changing the preference keeps route-local drafts, focus and reading state mounted. Android push/pop
durations become zero; lazy course-review deep links jump and wait for layout before locating the row.
Short return-to-top movements animate; jumps beyond three viewport heights go directly to the target.
Media wrap-around also jumps directly instead of sweeping through intervening images.

`Partial`: widget coverage verifies cadence, bounded feedback, interruption, reduced motion and
iOS back gestures. Physical-device frame timing and subjective motion acceptance remain runtime
validation; simulator/debug execution does not establish production frame-rate guarantees.

## Language and presentation

- `Current`: bottom sheets size to short content and constrain long, scrollable content to the
  available viewport, with a maximum width of 640 pixels, 24-pixel top corners and a shared drag
  handle when dragging is enabled. Device safe areas are consumed once: the title starts at the panel's own
  padding, and the panel background extends behind the bottom home indicator. Scheduler pickers,
  course filters, account pickers, Wiki contents, language selection and publishing tools share
  this behavior. Input sheets and confirmation dialogs avoid the software keyboard. Review and
  reply forms allow the whole form to scroll when enlarged text and the keyboard leave too little
  space for the editor and actions; drafts survive resizing. Reply editing still confirms discard
  and prevents dismissal by dragging or tapping outside.
- `Current`: shared form inputs use 16-pixel text. Button backgrounds use minimum heights of
  32/40/44/48 pixels by size, within touch targets of at least 48 pixels. Both grow for wrapped or
  enlarged labels; disabled actions remain visibly muted. Category chips have a compact 24-pixel
  fill at normal text size and at least 44-pixel targets when interactive. Home keeps category
  discovery directly above the stream and list/card preferences in the display menu. Home and notification tabs
  grow with system text size, and the overlay's content inset uses the same measured height.
- `Current`: empty and retry states share a soft icon surface, readable explanation and optional
  next action, with scrolling on short screens. Empty all-notifications link back to Home; empty unread notifications explain that everything is read and offer the All tab; empty drafts
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
- `Current`: navigation and settings use consistent 24-pixel line symbols. Navigation shows a soft selected
  background; settings use neutral symbols without decorative colored tiles. Social links use all six Web provider marks and brand colors.

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
  Topic subscriptions use topic-specific labels; reply commands have no toggle semantics. The dock
  keeps the floor number at its natural width and switches to an accessible icon-only reply action
  when that action's label cannot fit, including long translations and enlarged text, reserving no
  room for that action when a locked topic or restricted category omits it; the number itself only
  ellipsizes when it no longer fits beside the dock's remaining controls. SVG icons inherit their
  enclosing button foreground
  unless a semantic or provider color is explicitly set. Comment timestamps occupy a separate line;
  their action strip uses the full body width and starts at its leading edge. Actions retain 44-pixel
  targets, 20-pixel glyphs and aligned counts, wrapping together when enlarged text needs space.

## Input and component surfaces

`Current`: email changes, TOTP setup/enable/disable and sticker renaming validate and await the
server inside their editor. Failed writes keep the entered values with a local error for retry;
success closes the editor. Pending writes prevent duplicate submission, and changing accounts
removes the previous account’s private form state and ignores its late responses.

`Current`: native components are implemented by `ui_kit` on Flutter primitives. Ordinary account,
security, profile, course and schedule fields use a quiet filled surface with 16-pixel corners and a
52-pixel minimum height. Floating labels stay inside the filled surface. Focus and validation add a
thin continuous outline without a glow; field labels,
errors, native selection, autofill, password managers and IME actions remain available. Multiline
notes grow within their available space. Search uses a capsule and preserves focus on clear.

`Current`: private messages use a separate circular attachment action, a filled 24-pixel rounded
input with an inset sticker action, and a send icon. Reply entry starts at one line, grows up to
four lines, and keeps target context and a separate flat attachment/sticker toolbar. Publishing
retains its open writing canvas; edit tools use the same outline symbols. Keyboard and sticker
panels are mutually exclusive and preserve draft text and selection. Unsupported voice actions
are not displayed. Chat bubbles use 20-pixel corners and are bounded by the conversation pane.

`Current`: primary and secondary buttons use pill shapes with state-specific colors and a separate
48-pixel hit target. Icon actions retain at least 44 pixels, except the compact profile social links,
whose touch areas are 24 pixels wide and 32 pixels high. Menus, segmented controls and choice
labels grow with system text size; selected states expose semantics. Dialogs use one scrollable
surface with 24-pixel corners, while inline alerts use a quiet 16-pixel surface. Avatars, loading,
badges, dividers and selectors share the same semantic palette without third-party default skins.

## Publishing

`Current`: opening a publishing field or moving its caret alone does not create unsaved work.
While the software keyboard is visible, the edit step hides its introductory guide and empty photo
placeholder, retaining the title, body, save status and writing toolbar. Title focus and controller
identity survive this layout change. The header keeps a small outer margin for its primary action.


- `Current`: publishing uses a type-coloured icon, contextual writing hint and a two-step
  edit/preview indicator above an unframed writing canvas. New moments start with the body and
  an optional Add title action; opening that field alone does not create unsaved work. Existing
  nonempty titles remain visible in topic edits and restored drafts. Type switches retain title
  input; articles and questions keep their required title field. Moment previews show only a
  manually entered title, while title-free publishing/server drafts use the existing body-derived
  API summary. Classification sits
  in a rounded panel below the preview. All three types share these controls and spacing. Article
  formatting tools remain
  folded in a bottom accessory bar above the software keyboard; expanding them preserves the editor
  selection and active body focus, including while local save status changes. Format buttons reflect
  the current selection visually and announce their label, enabled state and format toggle together;
  undo and redo are disabled when
  their respective history is empty. The heading tool applies heading 2 with a tap and opens a level
  sheet on long press that
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
  the publishing preview removes editor focus. Returning to article editing retains the live document,
  selection and undo history, restores the previous scroll position, and resumes body focus only if
  the body was focused before preview. Using the hide-keyboard action before preview keeps it dismissed
  on return. Rich-text
  formatting remains available while editing.

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
- `Current`: publishing can select up to nine photos per batch; simple galleries retain the
  nine-photo total limit. The foreground queue uploads in selection order, pauses at a failed photo
  for retry or removal, and ignores the result of a removed photo. Successful URLs are immediately
  included in local recovery; gallery ordering/removal and article insertion positions remain part
  of the draft. Article insertions track intervening text edits at the original selection.
  Pending photos visibly block leaving, manual draft submission, publishing and type changes.
  Temporary picker files are retained only for the current editor: app termination requires selecting
  unuploaded photos again, and the UI distinguishes this from saved text and uploaded photos.
  Backgrounding starts no further queued upload; an already-started request may finish. Resuming
  continues the current queue, while session/site invalidation rejects its results and later requests.
- `Current`: Next opens the preview/classification step. The step shows one publish action in the
  AppBar, with the draft action beside it as an icon button. If a long translation or enlarged text
  cannot fit, the next/publish action also uses a labelled icon button; up to three existing
  categories can be selected
  below the rendered image/title/body preview. Publish writes only after this step. The draft action
  saves incomplete forms locally; complete forms can be saved as server drafts, subject to the
  server's title, body and classification requirements. Moments can derive their title from the first text line.
- `Current`: publishing limits, captcha requests and other API failures use the Web error catalog
  in the selected language, including server-provided parameters.
- `Current`: server-required captcha challenges on publishing and replies explain that newer accounts
  may be asked to complete a captcha after frequent posting, and that the requirement lifts as
  activity subsides or the account meets the site age condition.
- `Current`: changed editors debounce local recovery saves by 700 ms and flush when leaving or
  the app becomes inactive. Title, Markdown/simple text, type, category IDs and uploaded image URLs
  survive reopening, including when the page metadata request fails. Save progress, success and
  retryable storage failure are visible. Each new composition has an independent identity;
  changing its content type keeps that identity. Cloud-draft edits, published-topic edits and replies
  use distinct identities. Earlier v1 recovery slots remain listed and can be explicitly reopened.
  Opening a cloud draft by its server ID while offline also finds its latest device recovery copy,
  before the server can confirm whether the topic is published or still a draft.
  Starting another composition never replaces a previous one. A restored editor still obtains current
  server metadata before publishing.
- `Current`: the drafts page presents device and cloud sections in one scroll surface, with a new
  composition action, content previews, recovery kind, content type and last-edit time. Continue editing
  reopens the same writing identity; replies reopen their topic. Returning from new or resumed writing
  refreshes the list. Title/text search and all/device/cloud/reply
  filters operate on device copies and the currently loaded cloud list; the cloud endpoint returns at most
  100 drafts and only its title/description are searchable here. A full 100-item window shows its limit
  and explains that deletion refreshes it to reveal subsequent drafts. Counts describe displayed copies, so a
  device recovery copy and its cloud draft count separately. Empty matches offer a filter reset. Local
  loading is distinct from an empty list; failed local or cloud refreshes retain displayed content with
  an inline retry. Local snapshots use app-private device
  preferences scoped by API origin and numeric account ID, with no token or background cloud upload.
  Logging out hides them; logging back into the same account restores access. Explicit discard or
  successful server acknowledgement removes the matching recovery snapshot; local deletion is
  confirmed with an explicit explanation that cloud content stays unchanged. Device and cloud rows both
  show labeled delete actions beside Continue editing. Cloud deletion uses the existing topic content
  lifecycle: confirmation moves the exact server ID to the recycle bin and preserves every local
  recovery copy, including copies associated with that ID. Only a matching successful per-item result
  removes the row; network, missing-result and item failures retain it. The server deletion-rate guard
  can request password confirmation using the same protected dialog as content management; cancellation
  keeps the draft. Success offers a direct recycle-bin
  action and refreshes the cloud window. In-flight deletion disables repeat actions, rejects stale
  list reads and rejects confirmation/results after account or site changes. The latest local deletion
  can be undone from a persistent action while the drafts page stays
  open; another deletion replaces that undo and leaving the page ends it. Restoration keeps the original
  identity and metadata, never overwrites an existing copy, and remains retryable on storage failure.
  Account/site changes clear search and undo state and reject queued stale restoration. Account closure
  attempts to clear that account's local drafts and searches. Serialized writes order deletion and
  restoration after pending saves. Storage failure is reported when saving; OS termination
  before the debounce/flush completes can lose the newest unsaved input.
- `Current`: changed editors offer continue, discard, or save to this device and leave. A server-required
  captcha can be refreshed without discarding content.
- `Current`: one reply recovery copy per topic preserves text, its reply target and uploaded image URL.
  Selecting another target replaces only the generated mention prefix, keeping the body. Collapsing,
  changing floors, leaving the topic and app inactivity preserve the reply; storage failure keeps the
  editor available with retry. Leaving after a storage failure offers continued editing or an explicit
  unsaved exit that preserves the previously saved copy. Restored replies rebuild local mention
  suggestions. An acknowledged send clears only unchanged submitted text; edits made
  while sending remain recoverable. The returned post ID opens its anchored reply window after success,
  resets obsolete pagination and updates the reply count used when returning to the feed.
  Reading a topic without editing creates no draft. Session invalidation prevents queued writing from
  crossing the account boundary; cache clearing does not delete writing recovery copies.

## Campus and sign-in

- `Current`: Campus opens a native private overview with the school teaching week, time-aware
  greeting, today's courses and then recent notices. Weekly timetable, academic records and charts,
  calendars, notice bodies and identity management use the existing campus API. Today’s timetable
  uses the server-resolved Shanghai teaching date, including holidays,
  makeup source weeks and explanatory notices; the first Campus entry after school-local midnight
  automatically refreshes a previous-day snapshot.
  The export-only adjustment switch does not disable this display. GPA is loaded only
  on the academic tab. The timetable shares the planner renderer without its editing or storage.
  The Campus bottom destination opens this page directly. Course reviews, the scheduler and Wiki
  have visible shortcuts at the top of its home view, also available to guests, unbound users and
  when school services fail. Pushed tools return to the Campus destination. Explore campus retains
  public course previews; shortcuts are shared with search discovery. See [campus semantics](campus.md) for binding, privacy and provider limits.
- `Current`: while the app stays in the foreground, Campus remembers its selected section,
  independent academic/notice search text and scroll positions, and selected timetable week when
  switching sections, bottom destinations or returning from a pushed page. Scroll restoration waits
  for the selected section's data and clamps to the available content; a fresh section settles at
  the top immediately, and manual scrolling cancels pending restoration. These choices stay only in
  page memory and survive app backgrounding; they are not serialized. Session/account/site changes,
  binding changes (including the first binding after an observed unbound state) and authorization
  loss still clear them. Backgrounding unmounts private response content, cancels its requests and
  clears the controller and foreground memory cache while the campus header and tab bar remain in
  place. On return, binding status is verified and current data is fetched again; the on-device
  snapshot remains stored for cold-start recovery but is skipped during this foreground refresh.
  A data-free, tab-shaped skeleton mirrors each section's real layout, retains fixed headings and
  controls, and uses a shared theme-aware shimmer unless reduced motion is enabled. Grades and
  notice bodies are not retained by the navigation state or added to the device snapshot.
- `Current`: school authorization uses the current native forum session in a restricted WebView.
  The initial Bearer header goes only to the first-party session handoff; school navigation receives
  no native credential. The server callback returns to a native confirmation, including resuming
  a notice after a permission update. Leaving the campus tab drops its private response content and
  cancels requests. Selected overview datasets have a five-minute foreground memory cache, reusable
  only after fresh binding-status verification; grades and notice bodies remain page-local.
  Backgrounding clears the foreground memory layer and controller state. A Drift device snapshot atomically retains only
  profile, calendar, timetable and server-adjusted today data, scoped by API origin, numeric forum
  account and binding revision. Grades, exams, campus messages/bodies and credentials are excluded.
  The private campus workspace places a compact refresh icon to the right of the snapshot time,
  preserving a 44-pixel touch target; stale/offline notices remain below the same-row metadata.
  Automatic and manual refreshes coalesce. On entry, a snapshot from an earlier Shanghai date triggers
  a complete four-dataset refresh after binding verification, alongside the selected section's reads.
  A successful snapshot's commit date prevents repeated daily refreshes, including after app restart;
  failures retain the previous snapshot and can retry on a later entry or manual refresh. Same-day
  restored snapshot tabs do not refetch the four persisted datasets when the foreground cache expires.
  Ordinary block failures keep usable same-day content visible; invalid teaching rules suppress old
  course results. Missing data retains an explicit refresh action. The existing fresh-read flow after
  backgrounding remains independent of daily snapshot reuse.
  Settings can clear only campus memory, device snapshots and desktop data, preserve drafts/plans and
  school binding, report partial failure and retry. Pending refreshes cannot refill a cleared cache.
  Snapshot storage is bounded to 1 MiB per document and four scopes; reads discard data older than 30 days.
  Pull-to-refresh keeps the last successful same-identity snapshot when the network fails; logout,
  unbind/rebind, account/site changes and explicit identity invalidation clear both snapshot and Widget
  data. The visible minute clock does not poll the network. School-local date rollover invalidates the
  in-app teaching-day response; Widgets advance within their last verified eight-day local window and
  request a refresh when a future day is unknown. See [campus retention rules](campus.md).
- `Current`: Android and iOS provide native home-screen course Widgets from the official campus
  snapshot, independently from the scheduler store. Display, local time advancement, privacy,
  settings and physical-device validation boundaries are owned by the
  [campus Widget specification](campus.md#home-screen-widgets).
- `Partial`: native school login on a physical device is not end-to-end verified. Automated tests
  cover navigation policy, session handoff, confirmation, stale responses and native rendering.
- `Current`: the scheduler opens in course selection. Plan preview remains a local planning
  grid, with week filters, conflicts, custom blocks and existing plan operations. Web and mobile
  warn about time conflicts before a teaching class is selected, while keeping the add action
  non-blocking. Completed term/grade/major selection collapses into an editable summary; course,
  credit, hour and conflict counts wrap in a compact row. A small Web action opens
  the full [Web scheduler](https://f.yourtj.de/schedule) in the external browser without transferring
  the native credential. Plans are not official enrollment results.
- `Current`: planner and official timetable grids share a responsive seven-day layout with a fixed
  section/time rail during horizontal scrolling. Larger screens expand the columns; narrow screens
  keep readable column widths and explain sideways scrolling. Spanning course blocks show title,
  room, teachers and week range; single-section and stacked blocks prioritize title, room and week
  parity, with complete details in their accessible labels. Course colors retain stable slots, while
  soft borders, an accent line and separate conflict icons follow the Web hierarchy. Row heights and
  column widths follow accessibility text scaling, including nonlinear scaling of small text. Course
  details and selectable empty cells support keyboard activation and labeled screen-reader actions;
  unconfigured empty cells and custom placeholders do not present inert buttons. The week selector
  has a minimum 48dp action height.
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
- `Current`: the course catalog debounces keyword search and captures filters for each request
  generation, so late responses and pages cannot replace a newer search. Short lists load the next
  page automatically while visible. Paging errors keep existing courses and offer explicit retry;
  duplicate pages stop automatic loading until retried. Pull-to-refresh retains results and shows
  an inline retry on failure. Department, term and campus pickers search both values and displayed
  labels, retain selections across search terms, and provide clear-selection controls; teachers
  remain free-text multi-value filters. Filter options have separate loading/error feedback, and
  search plus all filters can be reset together. Sheets accommodate the keyboard and large text,
  with a persistent Done action. Session/site invalidation clears the old catalog, permissions and filters, then loads the new
  session’s catalog; queued searches and late results cannot cross identities. These interactions use the existing
  course API and SSR filter options; search service failures remain errors rather than empty results.
- `Current`: each catalog row shows the most recent term first. When more terms exist, the first term,
  a `+N` counter and a chevron live in **one** disclosure chip (the term keeps the same muted colour as
  a single-term row, so colour never implies selection); expanding swaps the counter for the localized
  collapse action and an up chevron and lays the remaining terms out in an inner 6dp wrap, so wrapping
  happens per group rather than one stray chip per line. Info chips (course code, terms) share one
  spec — `type.meta` text with 4dp vertical padding, a 6% `baseContent` fill that stays visible on
  white and dark surfaces, and the `--gf-radius-selector` (8) radius — while the metric row keeps a
  12dp group gap against the 6dp intra-term gap (2×). Term chips keep at least a 44dp target and only
  change the row's local disclosure state. Every metric row is laid out at the 44dp target height
  (visible chips centred) whether or not it has a disclosure chip, so single-term and multi-term
  cards are the same height; title and teacher lines trim their outer leading, and on device the
  visible gaps are even (about 13dp top, title→teacher, teacher→chips and bottom) at 100%, 130% and
  200% text scale.
- `Current`: course filter chips are flat 32dp pills (the `--gf-radius-selector` radius, not stadium
  pills) inside a 44dp target; the visible pill carries the label, an optional selected-count badge, a
  chevron-down affordance for the multi-select pickers and a check for the reviews-only toggle. The row
  scrolls horizontally with a 12dp edge fade hinting at more chips; selection updates the query
  immediately and uses a short color transition that settles immediately when reduced motion is enabled.
  While any search or filter is active, a 32dp square × chip (same radius, labelled “reset search and
  filters”) leads the same row instead of a separate text button row, so the list never shifts down.
- `Current`: course details retain offering-specific five-star reviews and existing review fields;
  bookmark and write-review actions stay in a bottom bar that reserves its own layout space, so the
  scrollable review list never renders (or taps) beneath it. Scores share a baseline with their
  five-point denominator. The rating summary draws the average as a 104dp progress ring whose arc
  carries the Web ring's linear `warning → primary` gradient (bottom-right to top-left, the same
  geometry as the rotated SVG), which has no angular seam at the arc start, with a primary end dot
  ringed in the card background; the
  5→1 distribution bars use the same warning hue with a brightness ladder that is brightest for 5★
  (0.95) and dimmest for 1★ (0.24). The signed-in user’s own reviews (including anonymous reviews) appear
  first across pagination. Every review card shows a 40-pixel avatar: member reviews use the server
  `avatarUrl`, anonymous and legacy reviews (and a member avatar that fails to load) use the shared
  generated face seeded from the public author label plus the review id, so one review keeps the same
  face as the Web card. Server-provided relative avatar paths are resolved against the API origin
  before loading, for the detail card, My course reviews and the share card alike. Review metadata
  is one wrapping run per card: term, class, instructors and the short review date (relative inside a
  week, `M月D日` after, year only across years); class code, campus and faculty stay in the page header
  and offering list instead of repeating on every card. Review bodies use the shared
  Markdown renderer, and the editor uses the app’s rich Markdown surface with six Web-matched
  templates rendered at the compact reading profile.
  The editor toolbar can pick an image from the gallery, upload it through the shared
  `/file/img-upload` pipeline and insert the resulting Markdown image at the caret, so a review
  body can carry the same image content as Web without a second upload implementation; while a
  pick or upload is pending, Publish/Save stays disabled so the image cannot be dropped. The write
  sheet pairs the offering selector and the five 48dp rating targets on one row (selector at the
  start, rating at the end) and keeps the publishing identity on the next full-width row: the
  current user’s avatar and nickname with “posting as …”, or the anonymous placeholder with a
  separate “post anonymously” title and “identity stays hidden” hint, next to the anonymity
  switch; the copy stays complete at 320dp and 200% text scale, wraps the selector/rating pair when
  they cannot fit, and collapses to a single scrollable row when the keyboard leaves too little height.
- `Current`: the review action bar is one row of at least 44-pixel hit targets — helpful count,
  dislike count and image sharing — whose visible pill is only 32dp tall (13pt label, 16pt icon,
  12dp side padding) so the card stays reading-dense; the row scrolls horizontally on narrow
  screens or at large text instead of wrapping. Edit and delete live in the card’s top-right overflow menu, which also carries reporting
  for other authors’ reviews; a guest sees no report entry, and reactions send guests to sign-in.
  Reports require an explicit reason and limit supplemental notes to 300 characters; a failed
  submission keeps the entered values visible. The server switches a reaction in one transaction
  (removing the opposite state and writing the selected one) and serializes concurrent changes per
  review. A reaction toggle keeps the list mounted with no loading state or re-read and applies the
  mutual-exclusion rule locally in the same frame: it clears an already-selected opposite side
  first (idempotent) and then writes the target state, so older server builds still yield
  exclusion. On failure Flutter shows a localized error; if the opposite side was already cleared
  but the target write failed it rolls back to neither side selected (matching the server),
  otherwise it restores the previous counts.
- `Current`: course reviews and forum posts/replies can open a shared image preview with themes,
  fixed-width Markdown cards, save and system-share actions. Cards render at 375 logical pixels,
  3× capture scale, and a fixed text scale; compact Markdown styles keep long posts within bounds.
  The five palettes include paper, dark and three pastel themes; pastels are mobile-only and derive
  their surfaces from GF color tokens. Long captures are tiled at 4096 physical pixels and
  capped at 48 MiB of RGBA output (about 12.6 megapixels at the fixed width); CPU stitching and PNG
  encoding run in an isolate. Network images settle or use a stable placeholder after 20 seconds.
  Review cards keep the public author label rules for member, anonymous and legacy reviews and are
  a single white card (20dp radius, hairline border, two soft shadows tinted from the text colour)
  floating on a deeper canvas (`base300` on light palettes, `base200` on dark): the course name
  (22pt) with “teacher · faculty · code”, a stat bar that reads in one line (this review's score with
  its stars | course rating | review count, hairline-divided), the server-normalized `contentHtml`
  at a 16/1.7 reading profile with a narrow heading scale (19/18/17/16), and after a hairline the
  signature (avatar, author, “offering · date”). Every text may wrap; nothing is ellipsized. Small
  journal-style details live only in the outer margin and on the card edge, never over text: a
  translucent striped washi tape across the top-left corner in the palette accent, two four-point
  sparkles in the star colour, a halftone dot grid and a thin ring peeking from behind the card;
  there are no gradients, glows or colour bars. Unlit stars are outlined so all five slots stay
  visible; dark palettes use a black shadow and a lighter stat-bar fill. Body images use the reading
  card's centred, bordered shell. The theme picker shows each palette as a 32dp swatch split
  diagonally into the card colour and the palette accent (pastel surfaces alone are almost white;
  Paper uses a slate accent so it stays distinct from Blue),
  with a primary ring and label emphasis on the selected one. Secondary text and the stat bar are contrast-checked across all
  five palettes; the preview is the exact exported card.
- `Partial`: automated tests cover PNG capture and tiled stitching; native save/share behavior and
  visual layout still need simulator and device acceptance.
- `Current`: Profile includes a private My course reviews entry for paginated management across
  courses, including anonymous reviews. Management rows reuse the detail card’s avatar and top-right
  overflow menu (open course / edit / delete) instead of a wrapping action row. Hidden reviews remain
  listed for deletion, with no edit or public-detail action; deleted reviews are omitted. Course
  detail and management share the same editor and a rounded delete confirmation with the target
  review excerpt and explicit cancel.
- `Current`: shared transient feedback appears in dismissible top banners above sheets, below
  the system safe area. Course review failures show localized server reasons and preserve the
  draft; success and error messages use the same surface with distinct semantic icons.
- `Current`: Wiki search uses the existing page-grouped search contract, debounces input, ignores
  stale results and opens paragraph anchors. Search unavailability has retry feedback. Reading
  keeps directory, Wiki search and GitHub edit actions in a bottom dock; GitHub remains the content
  source of truth.
- `Current`: Wiki body links open native Wiki pages and the Wiki overview only for the configured
  site origin (scheme, host and port). External links, including other sites' `/wiki/` paths, retain
  their destination and use the shared external-link confirmation. Same-site repository attachments
  under `/wiki/_assets/` open their actual URL in the system browser/app; launch failure keeps the
  reading page and shows a localized error. Encoded page/file paths, query strings and fragments are
  preserved, while page-local anchors continue scrolling inside the document.
- `Current`: sign-in offers account/password, Google, GitHub and Tongji when the published options
  allow it. The provider buttons stay folded behind one labeled control below the password form; it
  opens a short draggable bottom sheet that lists every published provider with the same icon, label
  and native flow as before. The login tab shows the complete card without scrolling on a 390-by-844
  phone with safe-area insets in all four languages, including the password captcha once it is
  revealed. A short form area, such as a small phone or the space left by the keyboard, drops the
  brand lockup and subtitle so the form and its sign-in methods fit instead. The register tab still
  scrolls: before any captcha challenge its control row sits just below the fold (measured at 390 by
  844: 39 px in English, 54 px in German, 5 px in Chinese and Japanese). The Tongji notice and its
  policy links live with the Tongji entry, and unconfigured providers stay hidden. Opening moves
  focus into the sheet; dismissing it by barrier, drag, back or Escape returns focus to the control,
  while choosing a provider closes the sheet and starts that flow on the page. Native credential
  fields expose username/password/new-password autofill, email and one-time-code hints and explicit
  keyboard actions; password-manager saving is requested only after accepting the native session.
  Password fields stay obscured by default and each carries a state-labelled reveal toggle that
  keyboard traversal can reach; showing or hiding leaves the text, caret and focus untouched, and
  screen readers announce it as one labelled button.
  Back, language and appearance controls stay outside the scrollable form, so long errors,
  enlarged text and the keyboard cannot cover their touch targets.
  Narrow layouts and larger text stack the captcha image above its input. Password captcha and TOTP remain
  supported. The login captcha stays folded until the password field is first interacted with;
  the first password focus/input warms the challenge. Blank taps and keyboard dismissal reveal it
  without reopening the keyboard. Only the password keyboard Next action moves focus automatically
  into the captcha; explicit field taps keep their target. Registration Next advances one field at a time. That reveal is latched through transient Android
  focus rebounds, and a prefetch failure stays silent until the visible retry path is used.
  The captcha image itself is the Web-equivalent refresh control: a tap requests a fresh challenge,
  keeps an already-focused captcha field and its keyboard, and covers the image with a progress
  state; the image is inert and submission is refused while the request is in flight, and the
  previously typed code is cleared only once the new challenge arrives. A failed refresh keeps the
  old image, the still-valid typed code and the retry target, reporting a retryable error instead of
  a blank frame. Once the server requires a captcha, an empty code is rejected locally with a
  localized message instead of spending a login attempt. On
  Android, auth-field pointer-down or keyboard Next creates a short-lived target token; if the secure keyboard
  reclaims the password focus during that token's settling window, the app makes at most two
  bounded attempts to return focus to the explicitly tapped field and then stops. A focused field
  also has a finite view-insets-based IME show watchdog. Dismissing an already-visible
  keyboard cancels recovery and is honored as user intent; a transient hidden IME during an
  explicit password-to-username/captcha handoff remains recoverable. Blank-space and button taps
  create no focus target and do not start a focus battle. Captcha pixels are left unchanged in light mode
  and use the Web-equivalent dark-mode transform. Google availability follows the published Web
  configuration. On Android, Google/GitHub/Tongji all use one RFC 8252 external-system-browser
  flow with manual PKCE/state/nonce and the native MainActivity callback bridge; the Android path
  intentionally bypasses flutter_appauth/AppAuth/CustomTabs. No OAuth provider uses a WebView for
  Android login. Non-Android platforms retain AppAuth. `Partial`: the new Android path awaits a
  physical-device APK test; the exact native crash stack remains unproven without logcat.
- `Current`: native routes that require a session lead guests to sign-in before constructing the
  private page. Login retains the original native location, including topic reply position, composer
  context and chat recipient, using an explicit route/query allowlist. External, recursive and
  malformed return targets fall back to Home. Successful login replaces the old navigation stack and
  restores only that context; detail pages sit above a fresh Home so Back remains available, while
  shell destinations open their own branch. Users still explicitly submit posts, follow users or send
  messages. Keyboard submission shares the button's busy guard for login, TOTP, registration and
  password recovery. Device settings remain public: guests can change language and appearance without
  fetching account details or sessions. The category index and account sections retain their
  sign-in destination alongside appearance, language and desktop-widget preferences. A session change
  removes dialogs, menus and sheets owned by the previous session from the root and shell navigators, completing pending confirmations as cancelled;
  new-session overlays remain open. `Partial`: native password-manager prompts and physical-device
  keyboard behavior still require device validation; widget tests cover route boundaries, four
  languages, narrow viewports and 200% text.

## Registration

- `Current`: registration loads the current Web login configuration before submission. Restricted
  email domains use a prefix field and domain selector; unrestricted sites accept the full address.
  Password confirmation is checked locally. Only published terms/privacy policies are linked and
  require explicit agreement. Configuration failures preserve the form and offer retry.
- `Current`: the restricted email field accepts either a prefix or a full address at a published
  domain, matching domains without case sensitivity. A full address selects its own domain instead
  of appending another suffix; malformed or unlisted addresses stay on the form with a localized
  error. Registration API failures use the shared server-message catalog, including daily quotas
  and retry-login instructions. Occupied usernames/emails and creation failures retain the same
  generic registration error, preserving the server's account-enumeration boundary.
- `Current`: when email verification is enabled, successful registration tells the user to check
  their inbox; when it is disabled, the ordinary registration confirmation remains. Email changes
  tell the user to check the new address for its activation link in both immediate and staged modes.

## Persistent forum anonymous identity

`Current`: native settings offer the same complete THUOCL candidates, ten words per batch and ten
batches per Shanghai day as Web. Candidates wrap without truncation and require a confirmation
dialog explaining the one-year name lock and restricted audit boundary. An ambiguous draw retains
its request key for retry. Self-disable/enable keeps the persona UID, avatar and slot; a governance
restriction cannot be cleared from settings. Private state follows the current session epoch.

`Current`: native topic and reply composers show the current name/avatar and a main/persona choice,
retained in local writing recovery and failed/captcha submissions. Editing a published topic retains
its author. A continuation in an owned anonymous topic defaults to persona. Persona names and
notification actors open the native `/a/:publicUid` profile with only visible topic/reply history
and counts; author-only reply filtering compares public identities. The common avatar component
loads the server's fixed SVG. These are display identities and never receive a native user preview
or private-message action. Scoped moderators can open anonymous governance from topic/reply menus;
only explicit permission 7 exposes the audited reveal action. Reveal results stay in the open dialog
and are discarded on close or session change.

`Partial`: native widget/contract evidence does not establish physical-device keyboard, screen-reader
or production push delivery behavior. The shared business and privacy rules are owned by
[anonymous identity](anonymous-identity.md).

## Profile and privacy

- `Current`: visit statistics run automatically for guests and signed-in users in official Android/iOS
  release builds against `https://f.yourtj.de`, with no settings switch or consent preference. Fresh
  installations and upgrades use the same policy, regardless of any legacy saved analytics choice.
  Page views go to the existing first-party Umami website. Debug/profile,
  Web, desktop and alternate-server builds do not send. Public home/search/topic/category/course/Wiki
  navigation uses fixed `/app/*` categories; original IDs, slugs, search terms and fragments never leave
  the app. Campus, schedule, chat, notifications, profiles, login, settings, writing and administration
  routes are excluded. The payload supplies `browser: yourtj-app`, OS family and phone/tablet type;
  no account, session credential, hardware/ad identifier, content or persistent visitor ID is supplied.
- `Current`: analytics uses its own unauthenticated transport, three-second network timeouts, no
  redirects, at most three pending views, no disk queue and no retry. Rebuilds and brief foreground
  resumes do not repeat the same view; reopening after thirty minutes can start another view.
  Backgrounding cancels transport and drops pending events; disposing the collector also clears its
  memory-only Umami cache token. Normal browsing never waits for analytics or preference storage.
  Already delivered events remain subject to the server retention policy.
  About → App visit statistics provides a bundled, localized disclosure without login or network
  access, including when the administrator disables the site privacy policy. This read-only page
  explains automatic collection, upgrade behavior, data scope and retention; it has no switch.
  The automatic collection decision is recorded in
  [0051](../decisions/0051-mobile-automatic-visitor-statistics.md).
- `Partial`: Umami receives network IP/UA and may derive country/region/city. Its salted visitor
  calculation is approximate: installations sharing IP, OS and device family can collapse, and Web/App
  visits are not joined to an account. Aggregates cover reporting production installations, not a
  count of App accounts; offline clients, older builds and failed requests can be absent.
  Existing server retention and network-log policies still apply; no historical App identity is inferred
  or backfilled. The public status page exposes only coarse device/OS/client aggregates.

- `Current`: activity, topics, liked posts and bookmarks use flat avatar-led content rows with fine
  separators. Activity actor, action symbol/label and time share a compact wrapping header; excerpts
  and the separate like/bookmark controls align beneath it, with inset separators. Activity actions
  group at the start of the body column with an 8-pixel gap and wrap on narrow layouts, using the
  same icon sizes and state colors as Home. Liked posts and
  bookmarks share the same spacing and readable preview treatment. Topics, likes and bookmarks show
  the content author's name, title, excerpt and a compact first-image thumbnail when available.
  The Liked posts tab means likes given; the profile statistic still counts likes received.
  Anonymous replies and older servers without author enrichment use an unlinked neutral avatar.
  Bookmark replies and activity URLs with a post number open that floor. Each profile stream keeps
  its own scroll position when switching between different row heights.

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
- `Current`: profile editing presents the cover and overlapping avatar in the same crop and
  proportions as the public profile, with directly editable name, bio, signature and links below.
  Camera controls open the image picker/cropper; selected images and cover removal remain local
  drafts until Save. Cancel/back offers Keep editing or Discard when anything has changed.
  The existing independent APIs save text, cover and avatar in order. A partial failure identifies
  completed steps, retains remaining drafts and resumes without repeating acknowledged writes;
  leaving after partial success refreshes the saved profile. The public profile opens the editor
  directly in one route and returns there on cancel or save; opening from settings returns to settings. An account
  change closes active profile/crop editors and rejects old-session callbacks. The avatar picker
  retains twelve presets.
- `Current`: profile editing includes nickname, bio, signature, website name/URL, profile language
  and the six Web social providers. Saving preserves unedited fields and unknown social providers;
  website/social destinations accept HTTP(S), and social usernames expand to provider URLs.
  Public profiles display website/social links as icon-only controls and open them in the system
  browser. Provider marks use compact, equally sized 24-by-32-pixel touch targets alongside profile dates,
  with their names available to screen readers and long-press tooltips. Worn badges appear on the avatar independently of the badge list;
  administrator identity has a localized role label. Returning
  from settings refreshes profile identity and media immediately.

- `Current`: public profile covers extend behind the status bar and navigation. Back, overflow
  and notifications use circular frosted controls with 44-pixel targets. Scrolling reveals an opaque
  title bar with the name and topic count left-aligned beside the back control, and keeps content
  tabs below it; a status-area scrim protects white system indicators during collapse. The application supplies a theme-aware status-bar fallback, so returning from
  an immersive cover to a plain feed restores legible system indicators. Cover, avatar and profile actions share one header layer so the avatar stays
  fully visible. The compact action band keeps the display name eight pixels below the avatar
  ring. Other users' profiles show the same top-right overflow control; its menu offers block or
  unblock for that profile, while the own profile keeps its existing account actions. Account ID,
  bio and statistics use tighter related-content spacing. The band grows for
  wrapped actions and larger text. Pull-to-refresh starts below the safe area and toolbar. The editor crop preview
  uses the same available width and system inset as the public cover.

- `Current`: the profile's role and display badges use icon-only circular medallions with a fine rim,
  subtle highlight and inset face. Names remain available to screen readers and hover/long-press
  tooltips. Tapping opens a dismissible, scrollable information sheet with the name, description and,
  when provided, award date and distinct award reason. The badge gallery and badge selection controls
  share the same artwork treatment. Gallery cards arrange into columns according to available width
  and text size; large text reduces the column count and card height grows with the title. The dark
  theme retains a softly lit inner face so fixed-color server artwork stays legible.

- `Current`: the root avatar opens an account drawer with aligned 24-pixel outline icons, compact
  56-pixel minimum rows and a clear nickname/account-handle hierarchy. The full-height, square-edged
  panel slides over the leading side at 84% of the viewport width, capped at 400 pixels. Its contents
  scroll within the safe area. On tabbed root pages, opening drags track the finger across the leading
  55% only while the first tab is selected. On later tabs, horizontal drags stay with tab navigation
  and never open the drawer. Pages without swipe tabs retain the leading-side drawer gesture. Once
  open, a drag toward the leading edge closes it. An accepted drawer drag keeps following the finger
  when it reverses past its starting point. The outside shade exposes a localized close action to
  screen readers. Vertical scrolling and nested horizontal controls keep their gestures.
  Following/follower counts open the matching native connection lists. Unavailable
  counts show a placeholder with retry instead of zero. Opening the drawer refreshes the card, and
  account changes discard previous identity data. Profile, bookmarks, drafts, my content, recycle bin,
  course reviews, settings, community information and permission-gated workspaces remain available.
  A hairline separator aligned with the entry icons groups the account entries above the settings,
  community information and appearance entries, in both guest and signed-in states.
  The appearance shortcut opens System/Light/Dark choices; the open sheet follows theme changes
  immediately. The profile overflow retains its infrequent entries.
  Account controls are outside the public profile.
- `Current`: initial public-profile loading shares the resolved cover and avatar geometry, keeps
  the overlaid navigation available, and uses inline statistics plus the five-item content rail.
  Connection lists use their own two-tab/person-row skeleton without a cover. Stream placeholders
  follow the same compact avatar and reading column as loaded activity.
- `Current`: activity entries distinguish signup, post, like, follow and comment with matching
  icons and localized captions in flat rows with a content preview and compact timestamp.
  A first visit to a stream retains the collapsed profile header so loading, empty states and retry
  actions stay visible; revisiting restores that stream's loaded pages and scroll position. Empty
  badge lists use badge-specific feedback.
- `Current`: profile bios trim boundary whitespace; signatures use a mirrored feather and a wave
  below the complete text block, including wrapped lines.
  Admin and online chips sit beside the display name. The smaller handle sits below it.
  Joined date, available last-active time and public link icons appear in that order in one
  leading-aligned row, with consistent 8-pixel gaps between groups. Social icons follow the
  last-active label instead of being pushed to the opposite edge.
  Labels remain complete; overflow scrolls horizontally with a muted hairline cue.
  The website uses the filled globe-pointer symbol in black or white for the current theme.
  Avatar overlap participates in layout so it leaves no translated blank space. Earned badges use
  a compact, left-aligned row of shared circular medallions, retaining server-provided artwork.
  The 3-pixel gaps around the badge and statistics rows are visually balanced; no extra footer gap
  separates statistics from the profile tabs. The selected worn badge remains attached to the
  avatar independently.
  Settings combine checkboxes, display positions and drag handles in one badge list, selecting and
  ordering zero to five owned, enabled badges for the profile header.
  An explicit empty selection hides that row; existing accounts default to their first five badges.
  This selection does not change the avatar badge or the complete earned badge collection.
  Profile statistics keep all five values in one compact row; overflow scrolls horizontally with
  the same muted hairline cue.
  Settings groups use rounded inset surfaces, multiline row
  labels and consistent trailing arrows; avatar upload copy describes image selection and cropping.
- `Current`: Settings opens a scrollable category index, with device preferences separated from
  account settings. Appearance offers system, light and dark modes; language and site information
  remain available to guests without fetching account details or sessions. Theme choices apply
  immediately, survive restart and take precedence over asynchronous restoration; writes are
  serialized so the latest choice remains stored. Account categories preserve existing section
  links, open on a normal back stack and fetch only their required data. Failed refreshes retain
  loaded content, and session changes clear private settings before loading the next account.
  The category index and section headers support enlarged text, keyboard activation and localized
  accessible labels; content stays centered within 720 pixels on larger windows.
- `Current`: users with follow permission retain the follow button for already-followed accounts,
  including administrators. It displays the followed state and toggles to unfollow, prevents duplicate
  in-flight requests and restores the previous state when a request fails.
- `Current`: profile content tabs form a continuous pinned rail. The selected item expands its icon
  and localized label; during a held horizontal swipe, the old and incoming icons move with their
  labels as the segment widths interpolate, using a subtle scale and fade. Inactive items show icons
  with accessible names. The underline animates with the tab widths, respecting reduced motion.
  Activity, content, likes, own bookmarks and badges fetch
  their corresponding streams. The header and tabs stay visible while an unloaded stream displays
  skeleton rows. Each stream retains loaded pages and scroll position; leaving a stream cancels
  unfinished reads, which restart if needed on return. Inactive reads cannot replace the selected
  stream; refresh and account changes
  invalidate older responses. Failed refreshes and pagination keep already loaded rows. Pagination
  follows only relative server URLs for the same user and stream, deduplicating overlapping rows.
- `Current`: following and follower statistics are keyboard-accessible navigation controls with
  at least 48-pixel targets. They open a separate native two-tab connection list, identify the
  profile by its handle, and link each person to their public profile. Both lists use the existing
  public `/u/:id/following` and `/u/:id/followers` PagePayload endpoints and retain independent
  pagination and scroll state. Visitors can browse public connections; an own-profile entry without
  a signed-in user offers login. The account drawer's existing connection links use the same page.
  Pull-to-refresh reloads the selected list to reflect follow changes; cached lists are not live
  subscriptions. Profile and connection content is centered at a maximum width of 760 logical pixels;
  statistics wrap and tabs scroll horizontally with enlarged text.
- `Current`: content management and recycle bin provide topic/reply filters, cursor loading,
  multi-selection, restore and deletion. Restore/permanent-delete affordances follow the server's
  capabilities; confirmation/password requirements and partial batch failures remain authoritative.
- `Current`: privacy settings link to content management and account closure. Closure offers
  anonymized-history and best-effort content-deletion modes, requires the current password and
  clears the native session on success. Retention and authorization rules match Web.

The approved permission/privacy/failure boundaries for optional analytics are recorded in
[STATUS-ANALYTICS-926-927](https://github.com/YourTongji/YourTJ-Hub/pull/926#issuecomment-5874262017).

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

- `Current`: iPhone/iPad distribution is live on the
  [App Store](https://apps.apple.com/cn/app/yourtj/id6809457637), and Android has a public
  [ARM64 APK](https://github.com/YourTongji/YourTJ-Hub/releases/download/mobile-latest/YourTJ-arm64-v8a.apk)
  plus [other architectures](https://github.com/YourTongji/YourTJ-Hub/releases/tag/mobile-latest).
  These are the formal distribution channels linked from [yourtj.de](https://yourtj.de/#download).
  Public availability was checked on 2026-10-03; it is separate from candidate-specific device validation.
- `Partial`: Android checks GitHub mobile releases at startup/resume with a six-hour limit and a
  manual About action. Update prompts support defer, ignore, progress and cancellation. The prompt is
  a bottom sheet that tapping outside or dragging does not dismiss: notes are grouped as required,
  security, new, improved and fixed under labels smaller than the note titles, and no note level
  exceeds the sheet title. The header shows the installed-to-target version; notes beyond the
  required items and the first five expand in place. The primary action stays pinned; cancelling a
  download keeps the prompt open. About history lists versions on a timeline, newest first. Release notes match Android split-ABI version codes by their shared build number. Public APK
  mirrors are ranked with bounded probes; SHA-256, package and signing-certificate checks precede
  the system installer. Prompts show the installed-to-target release notes, including required actions,
  and the About page provides Android release history. Notes are optional cached display data;
  they do not affect APK verification or installation. Unit tests and signed native builds cover the
  implemented paths; an installed-to-updated physical-device journey remains separate acceptance evidence.
- `Current`: released iOS builds are distributed and updated through the App Store; TestFlight is
  the candidate testing channel. Apple processing/review for each new version is independent of CI.
  The app does not offer APK-style updates on iOS.
- `Partial`: App Store builds check Apple's public listing and offer a link when a newer version is
  available. The App Store update prompt and About history render structured formal release notes;
  iOS history has no channel picker and never substitutes TestFlight testing instructions when
  public notes are missing. Missing public history keeps its empty or incomplete-history state.
  TestFlight handles beta update notices and the separate plain-text testing instructions. The app
  does not duplicate them in an automatic beta prompt; a tester's manual update check opens the
  TestFlight App Store product page, where they can open or install TestFlight. This fallback does
  not deep-link into the YourTJ beta. Unknown distribution receipts do not trigger channel-specific
  update prompts. Formal update prompts use receipt-backed channel coverage; incomplete history
  falls back to the target release summary.
  Signing, metadata, failure recovery and environment secrets are documented in the
  [mobile release runbook](../operations/mobile-releases.md).

## Verification boundaries

The source, contract and focused Flutter/Go tests define the implemented behavior. Figma is the
editable visual counterpart, not an alternative API or permission model. The maintained design is
[06 Mobile · Unified](https://www.figma.com/design/eLF6vFbmdwDQXec1IyuA4X/YourTJ_Mob_App_Design?node-id=284-302). Native device behavior,
visual acceptance, distribution and additional locales require separate verification;
local widget tests do not imply those gates passed.

## System notifications

`Current`: session cleanup preserves an absent push preference when a guest completes their first
login, so it cannot manufacture an opt-out before the one-time Android permission request. Existing
consent is cleared at account boundaries; an explicit Settings opt-out remains effective even while
native cleanup is pending.

- `Partial`: iOS uses direct APNs; Android uses JPush with selected OEM offline adapters and does not
  require Google Play services. Provider credentials, signing profiles and physical-device delivery
  remain deployment requirements; app-local notification lists are independent of system delivery.
- `Current`: Settings retains the push entry with a provider-processing disclosure and shows missing
  build configuration, unavailable server channels, permission denial and registration failure.
  On iOS the first launch after login requests the system permission once; granting it counts as
  push consent and enables delivery without visiting Settings. Android requests system permission
  once after a valid login reaches the main screen and its first frame is stable; an upgraded install
  without the one-shot marker receives the same request unless the user had explicitly turned push
  off in Settings. A grant counts as push consent. A denial
  keeps the app preference enabled and shows `permissionDenied` without initializing JPush or
  registering a device. Below Android 13, the bridge only reads system notification authorization.
  Settings retains explicit retry and disable actions. Resume checks existing authorization without
  repeatedly prompting; a non-empty token and successful API registration are required to display
  enabled.
  Enable taps during startup/resume or stop are queued; a later disable or account change cancels
  queued consent. Failed unbinding is retained and retried on resume while push stays disabled,
  using the owning account; signing in to a different account does not acknowledge that cleanup.
- `Current`: notifications use the existing event copy and only navigate to supported in-app topic,
  profile and notification routes. Logout stops native delivery and attempts server unbinding;
  the next account requires fresh consent. Optional JPush analytics/location collection is disabled.
- See [activation and device validation](../operations/mobile-releases.md#native-push-activation-and-verification)
  for credentials, supported OEMs and delivery limitations.

## Tongji sign-in

`Current`: native login and registration show “Tongji SSO” in the more-methods sheet when the public
login options declare campus configuration ready. The entry explains automatic activated registration
and links published policies. The backend handles the school callback and resumes the same manual
PKCE/nonce exchange
used by the Android external-browser path; the App stores only its forum session, never a school
access/refresh token. Existing bindings sign in to the same forum account; new users receive a
private student-ID@tongji.edu.cn email without a separate activation step. All four UI languages are
supported. Tongji shares the exact MainActivity-owned `yourtj://callback` bridge with Google and
GitHub; AppAuth's Android receiver does not claim it, and no WebView is used for this OAuth login.
`Partial`: physical-device school sign-in has not been validated with the new APK.

## Private user notes

`Current`: User profiles provide a private-note editor with retry and clear behavior. Names in topic
lists, replies, profile connections, search, conversations, notifications, mention candidates and
revision history use `note(display name)` for the current viewer, the display name being the current
nickname falling back to the username. Notes are fetched through the shared
core contract and remain only in a session-scoped memory provider; changing account invalidates
pending responses and never reuses notes from the offline forum cache. Limits and account-erasure
semantics are defined in [Identity and access](identity-and-access.md#private-user-notes).

`Current`: personal profile topics and activity entries show the current viewer's like and bookmark
state and allow toggling each action. Topic and reply identifiers are kept separate. Successful
detail-page actions update all retained profile streams on return without collapsing pagination or
resetting scroll; pending writes and older reads cannot undo each other. Failed actions roll back,
a fresh refresh reconciles external changes, and account changes discard personal state. Older
servers that omit interaction fields retain read-only content previews.


## Sticker library

- `Current`: messages, replies and publishing share a Recent / My / Official sticker picker. Official
  packs come from the administrator's enabled library, with a source and license link. Bundled presets
  retain their manifest's source pack; upgrades restore the default grouping on previously seeded
  official entries while preserving custom groups. Recent use
  keeps up to 30 distinct items in the current account/site session. Picking inserts at the caret or replaces the
  current selection; it does not send or publish. Unicode emoji and image stickers remain distinct.
- `Current`: sticker pickers stay inline below the reply, message or publishing input. Opening one
  dismisses the typing keyboard without covering the editor; the keyboard button resumes typing at
  the current selection. A bounded live preview renders draft stickers as images and updates after
  insertion, editing or deletion, using the same Markdown/plain-text rules as the destination.
  Repeated picks keep the panel open and advance the caret. Image uploads and reply-target/image
  removal keep the next insertion aligned with the updated caret. Ordinary typing does not retry
  unavailable sticker resolution; token changes and explicit retry can resolve again.
  Back closes the panel before leaving the
  page; short windows keep the picker scrollable and reply input controls accessible.
  In private conversations, picker, preview and keyboard height changes keep the bottom of the
  current reading position above the input area. Reading history does not jump to the latest message;
  a conversation already at the bottom stays there. Closing restores the position within list bounds,
  and visibility-based read receipts wait for the resized viewport to settle.
- `Current`: adding a personal sticker offers the system photo library or file picker. Cancelling
  either picker leaves the library unchanged. Selected photos use the same authenticated upload and
  retry flow as files, without applying the post-photo resize/compression settings to stickers.
  The personal library supports image upload, collecting a shared sticker by long press (in a chat
  message, the bubble's action menu offers the same collection for its resolved stickers),
  private display names, reordering and removal. Each library row opens a single-sticker preview
  and has a labeled action menu for preview, rename and removal; disabled stickers remain removable.
  Upload guidance and library rows scroll together so large text does not crowd out the controls.
  Both single and selected removals require confirmation and preserve already-sent stickers. Failed
  removals keep the remaining entries available for retry. Account changes close pending library
  action sheets, library-launched previews and removal confirmations. It holds up to 200 stickers; images are limited to
  4 MiB and an account can create up to 1000 retained personal assets. Uploads use the authenticated
  file service. Failed requests retain the current input and expose retry. Concurrent collection
  writes are rejected explicitly so a skipped operation cannot report success.
- `Current`: personal membership is account-private. Shared personal token names can be resolved by
  recipients; the public directory lists official stickers only. Removing a library entry keeps
  shared posts and messages renderable. Account closure removes collection membership while shared
  assets retain their history references. [0038](../decisions/0038-personal-sticker-library.md) owns this
  storage and privacy decision.
- `Current`: native post/reply bodies and private messages display image stickers in a 128 × 128
  logical-pixel box, preserving aspect ratio and GIF animation. Consecutive stickers wrap within
  the content width. Picker and draft-preview thumbnails remain 56 × 56, and personal-library
  rows use 48 × 48, so the reading size does not crowd the input controls.
- `Current`: tapping a native inline sticker or a library row opens that sticker alone in the shared
  image viewer, with animated GIF playback, pinch/double-tap zoom and actual-size viewing. Stickers
  remain excluded from surrounding attachment galleries. Picker taps still insert without sending;
  a visible hint and each item’s long-press action expose a separate large preview on all three tabs.
  Previewing does not insert a token or change recent use. Chat long presses still open the message
  action menu, including when an image fails to load. Unknown or unavailable tokens retain a readable
  fallback. Library state is isolated by site and account.
- `Current`: disabled official stickers stay in personal management for ordering/removal and are
  unavailable for insertion; permanently deleted official stickers leave the collection. Older servers
  without personal-library endpoints show a compatibility message while official stickers remain usable.
- `Current`: Web post and reply sticker images are excluded from lightbox clicks and gallery
  collection; chat stickers are also non-interactive. Ordinary photos retain image preview, and
  images explicitly linked to another page retain their link destination. Sticker recognition uses
  renderer-owned provenance; an ordinary image's author-supplied alt text does not change its behavior.
- `Partial`: Web renders shared personal stickers, but personal-library management is a native App
  surface. Physical-device keyboard transitions and both-platform visual acceptance remain separate
  from widget and simulator coverage.


## Shared visual treatment

`Current`: Home shows animated feed sort tabs followed by a horizontally scrollable category row.
Category buttons use 12-pixel corners, a compact 36-pixel visual height and a minimum 48-pixel touch
target, growing with system text size. Labels remain on one line. The selected category uses a tinted
background, outline and check mark; the All categories button clears the filter. A newly selected
category scrolls into view, including selections made from a post badge. Selecting a button
or a Home post badge filters the existing Home stream without pushing a route. The display menu
contains the list/card choice, while categories remain directly available on Home.
Each category and its supported sort retain their own items, pagination and reading position; clearing
the filter restores the previous global sort and position. Category feeds reuse the category payload
and its latest/new ordering, while the unfiltered feed retains Following and other Home sorts.
Known sort labels follow the selected app language. Category and search results use the author-led
topic card, with the current category omitted from repeated card labels. Topic headers keep the
name, timestamp and categories on one row: long display names ellipsize before metadata can wrap.
Very narrow or enlarged-text layouts let the category group scroll horizontally; author and category
touch targets remain separate and at least 44 pixels. Full names remain available to accessibility
and through the author profile. Feed card previews use the topic gallery's blurred image fill and
theme-tinted veil behind the uncropped image.
Unread notifications have a filter-specific empty state and no unrelated publishing action.
Wiki recent items prioritize titles and update times; repository editing stays in the detail header.

`Current`: shared detail headers center an 18-pixel semibold title and use one 24-pixel back symbol.
Buttons use pill shapes, with visibly distinct disabled states in light and dark themes.
Navigation uses a restrained selected background, and settings use neutral symbols and inset groups.
Profile display badges combine selection, position and drag sorting in one list, with accessible
move actions. Saved edits retain their input until the server succeeds.


`Current`: course reviews keep changed input behind an explicit discard confirmation, block dismissal
while saving and retain the form on failure. Rating stars expose selected semantics and 48-pixel touch
targets. The write/edit sheet is editor-first: the title, a compact meta block (offering chip and
anonymous switch, then the rating stars), a pinned formatting toolbar and a pinned action row
(counter plus cancel and submit) surround a body that scrolls inside its own bounded region, so long
reviews never push the toolbar or the actions out of reach. The meta block shows every control without
a hidden horizontal scroll while the panel has room, and falls back to one scrollable row — stars and
anonymous switch first — only when the keyboard plus a large text scale squeeze the panel below 300
pixels. The sheet keeps the panel above the keyboard at every supported
text scale. Applying a template inserts into the existing controller in place — focus, selection and
undo history survive, the caret lands after a leading heading or label (typed text does not inherit the label's bold), and a non-empty body still asks before
being replaced. Templates convert line by line so the Quick template's bare `-` placeholders become
empty bullet items instead of swallowing the Pros/Cons labels above them. Cached AI summaries start collapsed, with refresh available inside the expanded section.
Settings show current device preferences, readable device/browser session names and platform-specific
push disclosure; account closure remains inside account settings rather than the main index.


## User safety and message retries

`Current`: profiles and conversation headers expose reversible user blocking; Settings → Account
lists the caller's blocks. Server enforcement stops new private messages and interaction
notifications in both directions. Public content, message history and already delivered notifications
remain available. Pending activation does not prevent managing a block.

`Current`: the message action menu exposes a report action for received messages, with explicit
disclosure of the selected message to administrators; the report entry is no longer rendered under
every received bubble. One message can be replied to from the same menu. Replies retain a
plain-text `> @sender: excerpt` quote line above the body, with a separate target message ID for
navigation; the excerpt is bounded to 120 characters and sticker tokens expand to their
readable preview label. Conversation previews drop that leading quote before the 255-character
preview bound, so the inbox shows the reply body rather than the quoted excerpt. Topic/post and message forms submit a fixed reason enum plus a separate
explanation. Only administrators can review or handle private-message evidence in the embedded
moderation workspace; global/category moderators cannot obtain it. The
[privacy boundary decision](../decisions/0040-user-blocks-and-private-message-reports.md) defines
retention, account cleanup and concurrency behavior.

`Current`: each session-local outbox entry has a random client message ID that remains stable on
retry, and binds the composer snapshot captured at submission to its draft revision. A failure
reinstalls that snapshot under the same revision only while the draft is unchanged or was emptied by
an unrelated user action; text typed in flight (including whitespace) and drafts a newer send already
acknowledged are never replaced, and the reinstalled draft still matches its pending entry. An
attempt a second callback already claimed is not treated as a failure. The server deduplicates the
same sender/key and rejects a changed peer/body/type. Keys are
retained with messages. Older clients without a key keep legacy send behavior. Restarting the app
does not restore an outbox entry's key; a newly composed message is a new send intent.

`Current`: private-message bubbles no longer show a timestamp under every message. A localized date separator
(today, yesterday, month/day within the current year, full year/month/day otherwise) opens each
device-local calendar day, and a bubble shows its local `HH:mm` only when it is the first message of
its day or its gap from the previous message exceeds five minutes. Own and peer messages follow the
same rule. Web renders the same separators and visibility rule in place of its former static
today-only label; the separator is exposed as a heading on both surfaces. Grouping is recomputed from
the loaded list, so prepending an older `beforeId` page or appending a live message updates the
boundary message without duplicate separators. Invalid legacy timestamps keep their original text
and start a new group instead of guessing a date.

`Current`: message timestamps, conversation-list times and message date separators convert
offset-bearing server timestamps to the device timezone. Date-only calendar values stay calendar
dates; legacy timestamps without an offset and with a time of day are interpreted as UTC wall-clock,
matching Web and the historical server format (issue #221), so both surfaces derive the same day
boundary and clock. UTC and explicit-offset representations of one instant display identically,
including day/year boundaries.

`Current`: private-message bodies retain their complete text and sticker tokens. Conversation-list
summaries are limited to 255 Unicode characters, including a truncation marker; a truncated summary
does not split a sticker token. Long messages therefore remain sendable on PostgreSQL. Send failures
show a localized message without disclosing database errors.

### Apple login on iOS

`Current`: configured iOS builds list Apple's system sign-in button in the more-methods sheet next to
the other providers, and only for login.
The entry keeps Apple's official `ASAuthorizationAppleIDButton`, whose frame and corner radius are the
only adjustable properties, so it renders as the same pill-shaped row as the browser-based providers.
Apple localizes that control and its authorization sheet from the languages the iOS bundle declares
(`CFBundleLocalizations`: English, Simplified Chinese, Japanese, German), not from Flutter's in-app
language switch — Apple exposes no locale override for the system control. A device or per-app
language among those four matches the app UI; outside them the bundle falls back to the development
region (English) while the Flutter UI falls back to Simplified Chinese
(`resolveAppLocale`), so the two disagree.
`Decision needed`: the two ways to close that gap — writing the app's `AppleLanguages` default
(which only applies after a relaunch) or changing the Flutter fallback language — are product
decisions, and neither has an owner yet.
An existing forum user connects Apple from account settings before using it to log in. Cancellation
leaves the login form available. Account switching retains the cache-clearing boundary before committing
the new session. Apple authorization revocation expires only the matching Apple-authenticated session.
Unlink and account deletion revoke the server grant. See [identity semantics](identity-and-access.md#native-apple-sign-in).
`Partial`: a candidate still requires physical iPhone authorization/return, revocation and account-deletion
acceptance; simulator compilation does not establish those results.

### 私信回复、多选与转发

`Current`: Flutter 私信输入框的「＋」提供图片与表情库入口。单张图片先在本地预览，
确认发送后通过现有文件服务上传，以图片消息发送；上传失败保留预览供重试，发送失败保留
会话级待发送气泡，重试沿用同一消息标识和已上传 URL。文字、图片与失败重试共用单次发送限制，
选图/上传期间也不启动另一条消息写入。图片独立发送，保留未发送的文字与
引用草稿；选择器取消或账号切换后的异步结果不会发送。历史消息、待发送气泡与转发记录
均直接显示可信来源的图片，不包裹气泡底色或内边距，并可点开全屏查看和保存。
自动加载仅允许当前论坛源与构建时明确配置的资源源，逐次重定向也受同一限制；其他地址保留为文字链接。
缩略图按 240×180 逻辑像素与设备像素比限制解码尺寸，保持原比例，全屏查看仍使用原图。
图片使用现有公共素材 URL 规则，不提供会话成员专属的文件访问控制。
上传返回的 URL、收件人及幂等消息标识先写入与私信草稿相同的设备安全存储，再尝试发送；
失败、离开页面或重启后，原账号重新打开会话可手动重试，成功后删除恢复记录，不自动补发。
账号切换时晚到的上传结果仅保留在原账号；显式清除用户数据会撤销晚到结果的恢复写入权限。
本地选图预览仍只在当前进程保留；应用在上传接口返回 URL 之前终止时，通用文件服务仍无法保证孤立文件回收。
会话列表中，完整的图片文件地址摘要显示为本地化的「[图片]」；普通网页链接和带说明的文字保持原文，草稿预览仍优先显示。

`Current`: Web 与移动端的发送气泡使用独立色对：深蓝底（`#2563EB`）与白字（`#FFFFFF`），
浅色和深色模式均保持约 5.17:1 的文字对比度。正文和合并转发卡片继承气泡文字色；
移动端正文中的可点击链接也继承该色，Web 私信中的 URL 仍按普通文本显示。
接收气泡与转发详情沿用中性表面，移动端纯贴纸与单张图片消息不带气泡底色。

`Current`: 消息气泡与纯贴纸支持向左滑动回复；短滑、右滑和垂直滚动不触发回复。
长按浮动菜单提供回复、复制、转发、多选，以及适用的收藏表情和举报入口。回复选定后聚焦输入框，
保留未发送草稿；多选模式暂停输入，返回键先退出多选，草稿与原引用保留。
回复正文仍保存为现有纯文本引用行；客户端把引用行呈现为有界引用块，旧消息也可直接显示。
图片引用保存稳定的 `[Image]` 标记，Flutter 在输入预览与引用块中按阅读者语言显示图片占位，避免把发送者语言固化进消息。
新回复另外持久化可空的目标消息 ID。点击有目标 ID 的引用块会定位原消息、短暂高亮；
已加载但在视口外的目标先滚动到对应列表位置，未加载目标按 ID 读取附近记录后再定位。
定位过程暂停分页和途经消息的已读确认，聊天区右上方贴边显示带箭头与文字的圆角
「返回刚才位置」按钮，点击以 280 ms 缓入缓出的滚动恢复跳转前的阅读位置，开启系统减少动态效果时直接定位；
返回动画途中暂停分页与已读确认，到达后高亮来源消息。按钮颜色适配浅色、深色主题，长文案可换行。
手动滚回会话底部时自动收起返回按钮；尚有后续消息待加载的历史分页边界不算会话底部，引用定位的自动滚动也不会收起按钮。
旧记录与旧客户端发送的引用仍显示原文本，不做不确定的内容匹配。
合并聊天记录只以本地化的“[聊天记录]”占位参与引用；外层记录气泡隐藏复制整条和收藏入口，详情条目长按可复制单条或收藏其中的贴纸。
消息与转发收件人均使用圆形多选控件；消息选择栏平滑展开/收起并淡入/淡出，勾选状态使用原生填色与勾号动画，
开启系统“减少动态效果”时省略选择栏过渡。头像统一使用圆形裁切，
包括会话列表、会话标题、收发气泡和转发收件人列表；加载失败使用同形占位。
收发消息行的头像均不带描边，图片完整占用 32 个逻辑像素，保持双方视觉尺寸一致。
会话右上角的三点菜单提供屏蔽/解除屏蔽入口，选择后仍需确认；加载失败时可在菜单内重试。

`Current`: 一次最多选择 50 条已发送消息、10 个收件人，收件人来自现有会话与建议联系人。
二级页面可搜索、选择逐条或合并方式并确认发送。逐条最多 10 条，超过时使用合并转发；
逐条保留内容，合并卡片展示摘要并可进入记录详情；详情复用聊天页的头像、气泡、贴纸与文本选择，
按发送者名称和原始时间排列，作为只读记录展示。新记录包含公开头像 URL，旧记录使用圆形占位；
头像不提供原用户主页或原会话入口，图片可用性遵循公开素材规则。会话列表使用本地化的记录标签与正文摘要。
合并记录包含原发送者显示名与时间，但这些信息不构成身份认证。收件人只能阅读转发的副本，不能访问
原会话。再次合并转发保留内部“聊天记录”卡片，可逐层点开和返回；卡片外层显示发送该卡片者的头像与时间，
内层保持各原发送者的头像与消息。快照最多 4 层、合计 50 个条目（含卡片）、64 KiB；
转发受双方屏蔽、内容审核和普通私信频率配额约束。缺少头像字段的旧快照仍显示占位，不按昵称猜配头像。

`Partial`: 每位收件人的投递独立确认，失败可沿用相同标识重试；已确认成功的收件人不会重复发送。
离开选择页后，可从原会话恢复未完成投递。此队列仅在当前账号的进程内保存；切换账号停止后续发送，
重启会丢失未确认队列，重新发起可能产生重复。取消未完成投递会提示已送达副本无法撤回。
已送达副本不会随源消息、显示名或源会话的后续变化而改变；贴纸展示仍遵循素材可用性。

详见 [私信转发决策](../decisions/0044-nested-private-message-history.md)。


## Device storage

- `Current`: Settings → Data and storage is available without signing in. It shows forum reading,
  synchronized chat, images/GIFs and campus/widget cache categories, the managed on-device total and
  recoverable user work separately. Category values are payload estimates; database and journal
  overhead contributes to the total. Unavailable storage is an error, not zero usage.
- `Current`: clear-cache confirmation names the selected categories and explicitly preserves login,
  drafts, unsent messages and schedule plans. Forum/chat/media are selected initially; campus is
  opt-in because its offline document and timetable widget are removed together. Clearing fences old
  requests before deletion. Partial failure remains visible with retry, including after app restart.
- `Current`: cloud content management and the recycle bin remain in the side drawer. The storage
  surface offers local drafts and plans; account settings owns blocked-user management.
- `Current`: local reset confirms the number of local drafts, unsent messages, unsynchronized
  plans and schedule recovery drafts before removing them, signing out and restoring preferences.
  Storage reports recovery drafts separately from legacy schedules awaiting an owner. It does not
  delete cloud content, close the account or remove school bindings. Interrupted reset retains an intent for retry;
  business routes remain unavailable until it finishes, so new work cannot enter a pending reset.
- `Current`: ordinary writing and schedule plans commit to a dedicated encrypted transaction store.
  Legacy schedules without a known site are held for explicit recovery into a confirmed identity;
  they are never automatically adopted by a matching numeric ID on another site.
- `Partial`: offline coverage is bounded reading recovery for home/topic/chat plus the existing campus
  allowlist. Search, notification, Wiki and course pages do not gain a blanket disk cache. The media
  upload queue is not a durable offline-send service. See the authoritative
  [storage boundaries and limits](../architecture/mobile-state-and-cache.md#cache-policy).

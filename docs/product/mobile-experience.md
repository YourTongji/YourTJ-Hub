# Mobile experience

> Doc type: product spec
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-25

The Flutter app combines the forum, course catalog, scheduler and Wiki. Ordinary browsing and
writing use native pages. Management uses the same first-party workspaces and permission checks as
Web inside an authenticated in-app browser. The navigation and management boundary are recorded in
[0012](../decisions/0012-unified-mobile-reading-navigation.md).

The [interaction and layout standard](mobile-design-system.md) defines the shared visual and
behavioral acceptance rules. Its `Planned` requirements are tracked separately from the implemented
behaviors below; [state and cache boundaries](../architecture/mobile-state-and-cache.md) describe the
corresponding planned ownership and lifecycle contracts.

## Navigation and reading

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
  up to three previews with a total count. Tapping the author avatar or name opens the native
  profile; tapping text opens the topic. Tapping a preview opens the shared lightbox at that image
  with every topic image available, including images beyond the feed preview limit. Author
  targets and previews support keyboard activation; previews announce their localized image
  position, and both author targets have at least 44-by-44 logical-pixel touch areas.
- `Current`: topic bodies, Markdown and Wiki reading surfaces open the shared image lightbox. It
  supports swipe navigation, pinch and double-tap zoom, actual-size viewing, long-press save and
  system sharing. Home feed previews use the same lightbox and image actions.
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
- `Current`: profiles use a 3:1 cover (112–200 logical pixels tall), an overlapping avatar,
  trailing edit/follow/message actions, distinct name and handle, readable bio and inline statistics.
  Following and follower counts open the corresponding lists. Extra account tools, including course
  reviews, remain in the profile menu. Profile and connection headers identify the viewed person.
  Connection rows show a 48-pixel avatar, name, handle and up to three bio lines, with a pill follow
  action. Narrow layouts and enlarged text move the action below the bio. Follow actions serialize
  per person, update immediately and roll back on failure; late reads cannot overwrite local actions.
  Session changes clear pending relationship state. Self rows and old-server rows without relationship
  state omit the action; guests are directed to sign-in before following.
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
  each bubble can run at once. The API has no message idempotency key, so ambiguous network failures
  cannot guarantee exactly-once delivery when manually retried.
- `Current`: unsent private-message text and caret/selection are kept per peer in app-private device
  secure storage, scoped by API origin and numeric account ID. Conversation rows show a localized draft
  preview, including new peers without a server conversation; list search also matches draft text.
  Unresolved new-peer rows remain visible but cannot open until the server conversation list succeeds;
  a resolved existing conversation still waits for its initial history before enabling send.
  Input remains editable during sending. A successful acknowledgement clears only the submitted
  revision, while newer input and failed sends remain available. Retrying the unchanged failed draft
  reuses its outbox bubble. Saving debounces for 500 ms and flushes on leaving or app inactivity;
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
  internal-routing/external-confirmation policy. Inline stickers remain supported; chat text is not
  interpreted as Markdown or HTML. The selection menu also offers whole-message copy, preserving
  sticker tokens that partial native text selection omits. The emoji accessory replaces the current
  selection and leaves the caret after insertion. Replacing the draft with text that has no valid
  selection resets insertion to the end. Opening it dismisses the software keyboard and keeps focus
  inside the composer for hardware shortcuts; the keyboard control restores
  focus. Its bounded scrollable grid has touch-sized controls, localized labels and system-back/Escape
  dismissal. Mobile return inserts a newline; hardware Ctrl/Cmd+Enter sends. Disabling the composer
  also disables emoji edits. Platform IME transitions still require physical-device verification.
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
  Interactive category chips have at least 44-pixel targets. Home and notification tabs
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

`Current`: opening a publishing field or moving its caret alone does not create unsaved work.
While the software keyboard is visible, the edit step hides its introductory guide and empty photo
placeholder, retaining the title, body, save status and writing toolbar. Title focus and controller
identity survive this layout change. The header keeps a small outer margin for its primary action.


- `Current`: publishing uses a type-coloured icon, contextual writing hint and a two-step
  edit/preview indicator above an unframed, multiline title and writing canvas. Classification sits
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
  100 drafts and only its title/description are searchable here. Counts describe displayed copies, so a
  device recovery copy and its cloud draft count separately. Empty matches offer a filter reset. Local
  loading is distinct from an empty list; failed local or cloud refreshes retain displayed content with
  an inline retry. Local snapshots use app-private device
  preferences scoped by API origin and numeric account ID, with no token or background cloud upload.
  Logging out hides them; logging back into the same account restores access. Explicit discard or
  successful server acknowledgement removes the matching recovery snapshot; local deletion is
  confirmed. The latest local deletion can be undone from a persistent action while the drafts page stays
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
- `Current`: while the app stays in the foreground, Campus remembers its selected section,
  independent academic/notice search text and scroll positions, and selected timetable week when
  switching sections, bottom destinations or returning from a pushed page. Scroll restoration waits
  for the selected section's data and clamps to the available content; a fresh section settles at
  the top immediately, and manual scrolling cancels pending restoration. These choices stay only in
  page memory; backgrounding, session/account/site changes, binding changes (including the first
  binding after an observed unbound state) and authorization loss clear them. Private views still unmount and cancel requests when hidden; grades and notice bodies
  are not retained by this navigation state or added to the device snapshot.
- `Current`: school authorization uses the current native forum session in a restricted WebView.
  The initial Bearer header goes only to the first-party session handoff; school navigation receives
  no native credential. The server callback returns to a native confirmation, including resuming
  a notice after a permission update. Leaving the campus tab drops its private view and cancels
  requests. Selected overview datasets have a five-minute foreground memory cache, reusable only
  after fresh binding-status verification; grades and notice bodies remain page-local.
  Backgrounding clears the foreground memory layer. A Drift device snapshot atomically retains only
  profile, calendar, timetable and server-adjusted today data, scoped by API origin, numeric forum
  account and binding revision. Grades, exams, campus messages/bodies and credentials are excluded.
  The private campus workspace shows snapshot time, stale/offline state and a manual refresh action.
  Repeated refreshes coalesce; restored snapshot tabs do not refetch the four persisted datasets when
  the foreground cache expires. Ordinary block failures keep usable same-day content visible; invalid
  teaching rules suppress old course results. Missing or expired-day data requests an explicit refresh.
  Settings can clear only campus memory, device snapshots and desktop data, preserve drafts/plans and
  school binding, report partial failure and retry. Pending refreshes cannot refill a cleared cache.
  Snapshot storage is bounded to 1 MiB per document and four scopes; reads discard data older than 30 days.
  Pull-to-refresh keeps the last successful same-identity snapshot when the network fails; logout,
  unbind/rebind, account/site changes and explicit identity invalidation clear both snapshot and Widget
  data. The visible minute clock does not poll the network. School-local date rollover invalidates the
  in-app teaching-day response; Widgets advance within their last verified eight-day local window and
  request a refresh when a future day is unknown. See [campus retention rules](campus.md).
- `Current`: Android and iOS expose native “Next class” and “Today schedule” home-screen Widgets from
  a versioned, minimal projection of that Drift snapshot. Android uses Jetpack Glance with 2x1 and
  resizable 4x2/4x4 surfaces, and adds a default 4x3 “Course timeline” Widget with independent
  today/tomorrow switching and a scrollbar-free vertical course list; iOS 14 and later use
  WidgetKit/SwiftUI for systemSmall, systemMedium and systemLarge. The iOS 13 app remains usable
  without desktop Widgets.
  Widgets never access the network, advance class state and Shanghai midnight from local alarms/
  timelines, and use a schema-2 rolling window whose first day remains the server-resolved authority.
  Large widgets show today and tomorrow side by side. They support light/dark, Android 12 dynamic color,
  iOS tinted rendering, large text and screen reader descriptions, and deep-link to Campus today.
  Android 12–14 picker previews use a 4×2 two-day layout and a 2×1 next-class layout. App Appearance
  settings adjust only the widget background transparency from 0% to 15% (default 9%), keeping course
  text fully opaque. Widget settings disclose displayed fields, can rebuild or clear desktop data,
  and provide optional OEM refresh diagnostics. The source is the official campus snapshot and is
  independent from the `/schedule` planner store.
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
- `Current`: Wiki body links open native Wiki pages and the Wiki overview only for the configured
  site origin (scheme, host and port). External links, including other sites' `/wiki/` paths, retain
  their destination and use the shared external-link confirmation. Same-site repository attachments
  under `/wiki/_assets/` open their actual URL in the system browser/app; launch failure keeps the
  reading page and shows a localized error. Encoded page/file paths, query strings and fragments are
  preserved, while page-local anchors continue scrolling inside the document.
- `Current`: sign-in offers account/password, Google, GitHub and Tongji when the published options
  allow it, grouped below the password form. Unconfigured providers are hidden. Native credential
  fields expose username/password/new-password autofill, email and one-time-code hints and explicit
  keyboard actions; password-manager saving is requested only after accepting the native session.
  Narrow layouts and larger text stack the captcha image above its input. Password captcha and TOTP remain
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

## Profile and privacy

- `Current`: activity, topics, liked posts and bookmarks use flat avatar-led content rows with fine
  separators. Activity actions sit above normal-weight excerpts; topics, likes and bookmarks show
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
  matching native connection lists. Unavailable counts show a placeholder with retry instead of zero; opening
  the drawer refreshes the card, and account changes discard previous identity data. Profile,
  bookmarks, drafts, my content, recycle bin and my course reviews are direct entries. Settings and
  permission-gated workspaces remain available. The profile overflow retains its infrequent entries.
  Account controls are outside the public profile.
- `Current`: activity entries distinguish signup, post, like, follow and comment with matching
  icons and localized captions in bordered cards with a content preview and compact timestamp.
  A first visit to a stream retains the collapsed profile header so loading, empty states and retry
  actions stay visible; revisiting restores that stream's loaded pages and scroll position. Empty
  badge lists use badge-specific feedback.
- `Current`: profile bios trim boundary whitespace; signatures use a separate feather mark and subtle
  underline. Avatar overlap participates in layout so it leaves no translated blank space. The role
  label stays beside the name; earned badges appear as bordered title/description cards with colored
  hexagons and their server-provided SVGs. The selected badge remains attached to the avatar.
  Settings allow selecting and ordering zero to five owned, enabled badges for the profile header.
  An explicit empty selection hides that row; existing accounts default to their first five badges.
  This selection does not change the avatar badge or the complete earned badge collection.
  Profile statistics prioritize the values and wrap into fewer columns on
  narrow screens or at large text sizes. Settings groups use rounded inset surfaces, multiline row
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
- `Current`: profile content tabs always show their localized text, with a stable selected underline
  and a pinned rail. Activity, topics, likes, own bookmarks and badges fetch their corresponding
  server streams. Each stream retains its pages, scroll position, loading and retry state while the
  page is open. Inactive reads cannot replace the selected stream; refresh and account changes
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

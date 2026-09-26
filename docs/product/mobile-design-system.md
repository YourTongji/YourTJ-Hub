# Mobile interaction and layout standard

> Doc type: product specification
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-26

This standard applies the native reading direction in [0037](../decisions/0037-adaptive-mobile-reading-window.md)
to the Flutter app. [Mobile experience](mobile-experience.md) records implemented behavior. Rules marked
`Planned` below are acceptance requirements, not a claim that every screen already conforms.

## Content and visual hierarchy

`Current`: the first viewport prioritizes readable posts. Home uses one compact identity/search header
and persistent sort tabs with an immediately visible horizontal category row. Category buttons have
compact rounded surfaces, full single-line labels and touch targets that grow with system text.
Selection uses a tinted surface and check mark. Announcements remain reachable through an expandable summary;
category selection filters Home in place, with a visible current filter and a clear-to-all action. Returning to a stream
restores its items, cursor and reading position. Latest, popular, trending and following are distinct
streams; following contains posts from followed authors in `created_at DESC, id DESC` order, with
pagination cursors based on that tuple. Latest retains its existing
server ordering unless the product explicitly changes that ordering.

The implemented reading scale is defined in [mobile experience](mobile-experience.md#navigation-and-reading).
`Planned`: all surfaces use that shared type hierarchy instead of shrinking text to fit controls.
Names and action labels remain legible; timestamps and secondary metadata are subordinate. A post
uses one author line, one body/media region and one action row, with subtle separators rather than
stacked decorative cards. Color communicates selection and status, not merely decoration.

`Planned`: likes consistently use an outlined/filled heart. Bookmark, reply and subscription retain
their own meanings and icons. Counts share alignment and formatting; counts are not duplicated in a
second action region. The visible icon may be 20–24 pixels, but its focusable hit region is at least
44 × 44 logical pixels, preferably 48 × 48 on touch surfaces. Adjacent hit regions must not overlap.
Pressed, selected, pending, failed and disabled states are distinguishable without color alone.

`Planned`: the avatar and author name open the author, images open the shared gallery at the selected
image, and the body opens the discussion. Child actions consume their own gestures. A guest/offline
identity uses the brand mark or a deliberate person glyph, never a broken-image placeholder.

## Navigation and windows

Navigation uses the same destinations, icons and unread meaning at every window width. A wide
rail stays reachable independently of reading scroll; compact controls reclaim vertical reading
space. Compose menus follow their source column and consume system insets once. Implemented
breakpoints and bounds live in [mobile experience](mobile-experience.md#navigation-and-reading).
`Planned`: contextual panes and bounded reading layouts on every pushed page.

Resizing, rotating or opening the keyboard must not recreate a draft, reset the selected tab, navigate
away or lose scroll position. No product operation depends solely on hover. Keyboard users can reach
all controls, see focus, dismiss a transient panel with Escape and use standard editing shortcuts.
When both list and detail are visible, back first closes the detail context; narrow windows use a
normal pushed route. System safe areas and keyboard insets are consumed exactly once.

`Current`: public profile tabs form a continuous row. The active item expands its icon and localized
label; other items retain accessible icon controls. The selected underline moves with the changing
cell widths, with reduced-motion support. Loading affects only the content stream; the identity and
tab row remain stable. Loaded pages and scroll positions are independent per stream, while obsolete
requests are cancelled or ignored. Following/follower statistics open the matching lists.
Device preferences are separate from account editing, binding and security. Appearance offers System,
Light and Dark, and an already open choice sheet updates with the selected theme.

## Transient surfaces and input

`Current`: repository-owned Flutter components define filled 16-pixel form fields, capsule search/chat,
pill buttons, 24-pixel dialogs and consistent outline symbols. Native text selection, autofill,
input purposes and focus remain intact. Search, short reply and long-form writing use distinct
surfaces with shared colors and state treatment; the component model is recorded in
[0039](../decisions/0039-native-gf-component-foundation.md).

Focusing an editor is not an edit. Keyboard layouts prioritize the active writing surface over
introductory guidance, retain focus, and keep the primary action reachable. Implemented behavior
lives in [mobile experience](mobile-experience.md#publishing).

`Current`: profile and password editing use pages and preserve input on failed saves. Shared short
choice sheets use a drag handle, 24-pixel top corners, one safe-area boundary and a 640-pixel width
limit. `Partial`: legacy forms still use bounded scrollable sheets; profile, password and course
review editing confirm discarding changed input.
Transient feedback uses the existing shared banner; field-specific errors stay beside the field.
Retry belongs beside the failed operation. A success banner must not precede server acknowledgement
or a confirmed local write.

Only one input accessory is active: software keyboard, emoji/sticker panel, or collapsed state. Panel
switches preserve focus intent, text selection and composition. Insertion replaces the current
selection at the caret; it does not append to the end. Selection is clamped after document changes.
Hardware keyboards retain editing shortcuts and do not force a large empty software-keyboard gap.

`Planned`: moment/question publishing retains type selection and the edit/preview/classification
sequence. Articles use the existing rich editor with a focused writing canvas, selection-aware
formatting, undo/redo, visible save state and an accessible command surface. Preview preserves editor
selection and scroll. Media uploads appear in a queue with thumbnail, progress, retry/cancel/remove and
ordering; one failed attachment must not erase successful attachments or typed content. Publishing
waits for required uploads and never occurs as an upload side effect.

`Planned`: every local creation has an independent draft identity. Resuming edits the same draft;
creating another draft never silently replaces it. Local drafts, server drafts and topic edits are
visibly distinct. Reply drafts retain their topic and reply target across collapse and navigation.
Unsaved text remains available after validation, network or storage failure. Discard is explicit;
successful acknowledgement clears only the submitted revision. Drafts are user work, not disposable
cache. A draft row presents type, title or text excerpt, media count, updated time and save state.

## Feedback, freshness and reading

`Planned`: refresh keeps existing content and position while showing progress. Failure retains that
content with retry and a freshness indication. Initial failure, refreshing with data, pagination
failure, empty result and end of results are separate states. New query/scope/account generations
invalidate old responses. Lists use lazy construction and automatic pagination with a manual retry
after failure or no progress; pagination does not repeatedly retry itself.

Topic interactions share account-scoped state across home, search, profile and detail. Optimistic
updates are serialized per action; failure rolls back only that action's own revision. Like, bookmark
and subscribe failures use the same feedback rules. Topic floor indicators follow the visible reply,
and a successfully posted reply becomes visible without discarding the surrounding discussion.

`Planned`: guest prompts use a shared visual treatment, explain the specific benefit and offer one
clear sign-in action. Login returns to a validated internal destination and restores the safe action
context. It does not silently publish, follow, send or perform a destructive action. Forms support
native autofill, password managers, input purposes and submit/next behavior. Tongji SSO is prominent
when available; legal and account-creation details remain reachable without a wall of small print.

`Planned`: foreground message/notification updates use a session-scoped event connection. When reading
older messages, incoming messages preserve the viewport and show a localized “new messages” control
near the bottom edge. Downloading, caching or receiving a message is not reading it. Only incoming
messages that actually enter the active conversation's visible area are acknowledged as read.
Message text supports selection, copying and the shared link-opening policy. Unsent input is stored
per conversation and indicated in the conversation list.

## Campus, courses and local storage

`Current`: the device snapshot for name, calendar and official timetable shows the last successful
update time and manual refresh. Revisiting Campus or advancing its local clock does not repeatedly
request school data. Identity changes invalidate the snapshot; offline display identifies snapshot
age. Accepted decision 0035 defines this retention model. The lifecycle and data boundaries belong
in the [state and cache model](../architecture/mobile-state-and-cache.md).

Course results support clear query/filter semantics, searchable long option lists, preserved filter
state and automatic pagination. Applying a filter is atomic. The scheduler and official timetable
share the web product's time axis, course colors and merged blocks. On phones, course cards prioritize
course name and location; readable day/period headers and optional horizontal navigation take
precedence over squeezing a desktop week into tiny text. Selection, week and plan survive tool
navigation. Wiki retains query/anchor/context and has an explicit empty, unavailable and retry state.

`Planned`: Settings exposes cache categories and manual clearing with visible progress, result and
failure. Cached reading data, media and campus snapshots may be cleared independently. Drafts,
unsent message text and unsynced plans must not be included in “clear cache”. Clearing is fenced against
in-flight responses so the just-cleared data cannot immediately reappear.

## Accessibility and localization

`Current`: shared icon buttons merge their localized label, button role, enabled state and action
into one accessibility node. Disabled icons use a subdued foreground. The login theme switch
announces the theme it will select. Campus connection explains which snapshots remain on-device
and provides the same confirmed cache-clearing control as Settings; the control preserves drafts,
schedule plans and the school binding.

`Planned`: every icon-only control has a localized semantic label and toggle state where applicable.
Traversal follows reading order; sheet opening/closing restores focus sensibly. Dynamic changes such
as send failure or refresh failure are announced without repeatedly announcing the whole list.
Meaning does not rely only on color, motion, hover or a gesture. Reduced motion suppresses decorative
transitions and automatic rotation. Text contrast targets WCAG AA; focus and essential non-text
controls remain visible in both themes.

Chinese, English, Japanese and German share layout rules: labels wrap or move into a second row rather
than shrink; tabs can scroll without hiding their names. Feed author metadata stays on one row:
long display names use ellipsis with their full accessible label, and categories scroll horizontally.
Other metadata can wrap before primary content.
Pluralization and numbers use locale-aware messages. User content retains its language. Relative
directional padding supports future right-to-left layouts even though no RTL locale is currently
shipped. Avoid fixed-height text containers, substring-based truncation and concatenated sentences.

Acceptance includes widths 320, 390, 600, 768 and 1024; portrait/landscape and window resizing; both
themes and system theme changes; text scale 1.0 and 2.0; four locales; screen-reader/keyboard traversal;
keyboard open, slow/offline network, stale responses and account changes. Widget tests validate state
and layout; platform screenshots, gestures, IME behavior and frame timing need runtime evidence.

## References

- [X timeline choices and navigation](https://help.x.com/en/using-x/x-timeline).
- [X following and follower navigation](https://help.x.com/en/using-x/following-faqs).
- [Apple layout and interaction tips](https://developer.apple.com/design/tips/).
- [Apple undo and redo](https://developer.apple.com/design/human-interface-guidelines/undo-and-redo).
- [Flutter adaptive layout practices](https://docs.flutter.dev/ui/adaptive-responsive/best-practices).
- [Flutter input and accessibility](https://docs.flutter.dev/ui/adaptive-responsive/input).

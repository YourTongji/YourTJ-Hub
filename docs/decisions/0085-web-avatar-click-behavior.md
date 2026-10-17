# Web avatar click behavior preference

## Status

Accepted
Class: feature

## Context and Problem Statement

Every avatar and author entry point on the Web site funnels through `showUserCard()`
(`apps/gooseforum/resource/src/runtime/user-card-events.ts`): the click opens the floating user card, and
reaching the full profile takes a second click on the card's profile link. Readers who already intend to
open the profile pay that extra click, while the card's value (quick preview, follow button) serves the
opposite intent. Issue #1134 asks for a preference. The dispatch point is also reused by username text
links and by avatar+nickname anchors that wrap both in a single `<a>`, so the scope of the preference is
a decision, not an implementation detail.

## Decision Drivers

- No backend, contract, or payload change: this is a device-level UI preference like the existing shell
  sidebar and home feed preferences.
- One dispatch point stays the single source of behavior; the 13 call sites keep their templates.
- Modifier-key clicks and anonymous public identities keep their current browser-default navigation.
- An unset preference (including signed-out visitors) behaves exactly as before.

## Considered Options

- Device-level `localStorage` preference applied inside `showUserCard()` to every entry point (avatars,
  username links, and combined avatar+nickname anchors).
- The same preference applied to avatars only, passing a source flag at each call site and splitting the
  combined anchors in templates (TopicPage author bars, PostStream mobile author bar, ModerationPage
  report rows).
- Account-level preference synced across devices through the settings payload.
- A per-tab preference read once at startup, leaving other open tabs on the previous behavior until a
  reload.

## Decision Outcome

The preference is `goose:avatar-click-behavior` with values `card` (default) and `profile`, stored in
device `localStorage` through `runtime/avatar-click-behavior.ts` and surfaced in the settings page
"General" tab next to the theme selector. "Restore defaults" is the last row of that same General card
and its description/confirm copy name general settings in all four locales, so clearing this preference
is inside the scope the button declares; the preference is not reset from any appearance-only entry.

Because the preference is device-level, the module also listens for `storage` events and re-reads the
key, so a change made in one tab reaches tabs that are already open (including `storage.clear()`, whose
event key is `null`).

`showUserCard()` reads the preference at click time and, in `profile` mode, returns without
`preventDefault()`/`stopPropagation()` so the anchor's own `/u/{id}` link runs; the document-level click
interceptor in `runtime/router.ts` turns it into SPA navigation — the same strategy the existing
`publicUid` branch uses. Both branches therefore require call sites to render a real `<a href>`, which all
13 current call sites do.

The preference applies to every `showUserCard()` entry point — avatar anchors, username text links, and
combined avatar+nickname anchors — rather than avatars alone; the username-link exclusion in the issue's
out-of-scope list is superseded by this record. Modifier-key clicks (middle/⌘/Ctrl/Shift/Alt) and
anonymous public identities (`/a/{uid}`) keep browser-default navigation in both modes.

## Pros and Cons of the Options

- All entry points: one dispatch point, no template split, and one consistent behavior per visual unit;
  gives up the "avatar only" granularity that the issue considered.
- Avatar-only: matches the literal issue wording, but combined anchors must be split so a nickname beside
  a clickable avatar no longer shares one link, and every template gains a source flag with no
  user-visible benefit.
- Account-level sync: consistent across devices, but needs settings payload and contract changes for a
  presentation preference, and the Web settings page has no server-side storage for such preferences.
- Per-tab preference: no cross-tab machinery, but the same browser would show two behaviors at once and a
  device-level preference could only be corrected by reloading every open tab.

## Links

- [Forum product spec](../product/forum.md)
- [Runtime dispatch point](../../apps/gooseforum/resource/src/runtime/user-card-events.ts)
- [Decision log](README.md)
- [Issue #1134](https://github.com/YourTongji/YourTJ-Hub/issues/1134)

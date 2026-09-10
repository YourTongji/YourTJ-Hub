# Unified mobile reading navigation

## Status
Accepted
Class: architecture

## Context and Problem Statement

The forum, campus tools and communication streams need a consistent mobile shell. Keeping profile
as a root destination crowds notifications and private messages into one surface, while large
permanent controls reduce reading space. Different card, tab and icon treatments obscure hierarchy.

## Decision Drivers

- Keep forum reading and campus tools directly reachable.
- Distinguish notifications from private conversations.
- Preserve scroll position, accessibility and existing permission boundaries.
- Use the Web color tokens and the approved editable mobile design.

## Considered Options

- Four roots with an avatar drawer and scroll-aware overlay controls.
- Permanent controls with profile as a root and combined communication streams.
- Five persistent destinations including profile.

## Decision Outcome

Use Home, Campus, Notifications and Messages as the four icon-only roots. The avatar drawer opens
profile, bookmarks, content management and permission-gated workspaces. Global detail/editor pages
sit above the shell. Root controls hide after downward travel and return on upward travel without
resizing content. Floating actions keep stable meanings; topic reply and floor controls use a dock.

Use flat surfaces, shared Web colors, Lucide navigation icons, short tab underlines and a consistent
mobile type scale. Campus presents course discovery instead of an unavailable official calendar.
Scheduler grids represent local plans and explicitly link to the full Web experience.

This supersedes [0011](0011-mobile-navigation-and-management.md) while retaining its native core and
shared first-party management boundary: the full console stays in the authenticated, origin-limited
WebView, with server permission checks and cookie cleanup. OAuth account binding remains an external
site-settings operation. No new backend identity, management form system or calendar sync is implied.

## Pros and Cons of the Options

### Four roots and an avatar drawer

- Good: separates communication streams and gives reading more space without moving content.
- Good: keeps campus tools and every authorized management workspace reachable.
- Bad: infrequent account functions require opening the drawer; hidden controls require upward travel.

### Permanent controls and combined communication

- Good: always-visible navigation with fewer motion states.
- Bad: consumes reading space and combines two distinct user tasks.

### Five persistent destinations

- Good: profile is one tap away.
- Bad: increases navigation density for an infrequent destination and reduces comfortable spacing.

## Links

- [Mobile experience](../product/mobile-experience.md)
- [Shared management boundary](0011-mobile-navigation-and-management.md)
- [Unified Figma design](https://www.figma.com/design/eLF6vFbmdwDQXec1IyuA4X/YourTJ_Mob_App_Design?node-id=284-302)

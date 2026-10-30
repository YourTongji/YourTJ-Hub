# Group official stickers into packs and let administrators upload market packs

## Status

Accepted
Class: feature

## Context and Problem Statement

Official stickers ([0030](0030-global-sticker-library.md)) were one flat list: each row carried a
`pack` key, but the presets' keys (`official`, `emojipackage`, `flowerhd`, `wxmeme`) reached
pickers as raw text, and administrators could neither name, order nor cover a group, nor move
stickers between groups. The market ([0115](0115-sticker-market-subscriptions.md)) only held packs
synced from FluxDO, so a site could not publish its own sets through the subscription flow.
Administration also lacked bulk operations, and the Web had no way to manage "My stickers"
([0038](0038-personal-sticker-library.md)).

## Decision Drivers

- No migration of existing stickers or tokens; old clients keep working.
- One upload interaction for official packs and uploaded market packs: images or ZIP archives,
  per-file results, and later additions to the same pack.
- Uploaded market packs reuse the market pipeline: stored assets, AVIF conversion, generated tokens,
  publication, subscriptions and renaming with aliases.

## Considered Options

1. Keep `Entity.Pack` as the group key and add an optional `sticker_packs` metadata table (name,
   order, cover); uploaded market packs become market packs with `source = upload`.
2. Add a pack id foreign key to every sticker and migrate the existing keys into rows.
3. Treat uploaded sets as official packs and skip the market.

## Decision Outcome

Choose option 1.

- **Official packs.** A pack is the set of official stickers sharing a `pack` key. The
  `sticker_packs` row (`pack_key`, name, sort order, cover sticker) is optional: preset keys have
  code defaults (默认, EmojiPackage, 花字, 程序员梗), other keys show the key until named. The
  `official` pack always exists and cannot be deleted; deleting another pack moves its stickers to a
  chosen pack and never deletes a sticker. Custom packs get generated keys (`p-` + 8 hex). A cover
  must be an official sticker of that pack; otherwise the first enabled sticker is used.
- **Public read.** `GET /api/forum/sticker-packs` lists packs that hold enabled stickers, in
  administrator order, with name, cover and count. Pickers group official stickers by it and fall
  back to raw keys when the endpoint is missing, so a failure never hides stickers.
- **Administration.** `sticker-packs`, `sticker-pack-save`, `sticker-pack-delete` and
  `sticker-batch` (enable, disable, move, delete; at most 500 ids) manage packs. `sticker-import`
  accepts one image or one ZIP per request plus a target `pack`, and appends after the pack's last
  sort order. The admin page shows a pack sidebar, a card grid with search and status filters, a
  bulk toolbar, and an upload queue that sends each file separately (two at a time) with its own
  result and retry. ZIP failures retain their complete entry paths; retry submits only the failed
  entries, preserving successful totals without creating the successful official stickers again.
- **Uploaded market packs.** `sticker-market-pack-create` takes a name (up to 128 characters) and
  an existing category or a new category label; the pack starts hidden with `source = upload`.
  `sticker-market-upload` adds one image or ZIP per request; items are keyed by content hash, so
  the same image is never added twice, and each item's display name is its file stem. Sync and
  upstream import never touch upload packs; only upload packs accept uploads and name or category
  edits. Publication, subscriptions, withdrawal and renaming work as for synced packs.
- **Web "My stickers".** The Web picker gains a "Mine" tab (pending, rejected and withdrawn
  stickers are listed but cannot be inserted) and a management dialog; the same manager is a
  Settings tab. It uploads images, renames private labels, reorders and removes members through the
  existing personal-library API.

## Pros and Cons of the Options

### pack key plus optional metadata (selected)

- Good: no data migration; old servers and clients interoperate; one upload flow everywhere;
  uploaded packs get the market's storage guarantees for free.
- Bad: a pack key is a string, so a typo in a direct database edit creates a new group; uploaded
  market images are stored even when the pack is never published.

### pack id foreign key (rejected)

- Good: referential integrity.
- Bad: a migration of every sticker and of every client that reads `pack`, for no behavior that the
  key cannot express.

### uploaded sets as official packs (rejected)

- Good: one less concept.
- Bad: official packs are always visible to everyone; members could not opt in through
  subscriptions, and administrators could not publish or withdraw a set as a whole.

## Links

- Issues #1428 and #1429
- [0030: global sticker library](0030-global-sticker-library.md)
- [0038: personal sticker library](0038-personal-sticker-library.md)
- [0114: sticker rename aliases](0114-sticker-rename-aliases.md)
- [0115: sticker market subscriptions](0115-sticker-market-subscriptions.md)
- [Forum stickers](../product/forum.md#links-markdown-and-stickers)

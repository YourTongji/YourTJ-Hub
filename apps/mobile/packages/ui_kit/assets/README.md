# Mobile symbols

`assets/icons/` primarily uses [ReIcon](https://github.com/dqev/reicon) Outline SVGs on a 24×24 grid.
Files were sourced from ReIcon's `data/icon-data.json` at commit
`18819b023cc91280e6b89dd5dc05ff8ecbb3f987`; its license is
[LICENSE-reicon.txt](LICENSE-reicon.txt). `GfSymbol` renders them at the requested
size and tints monochrome artwork with the current theme. Use about 16px for
compact inline marks, 20–24px for actions, and 24px for navigation; keep touch targets at
least 44×44 logical pixels. The bottom navigation and wide rail use matching
Outline and Filled ReIcon weights for inactive and active destinations.
ReIcon credits [Solar Icons](https://solar-icons.vercel.app/) elements under
CC BY 4.0 in its upstream README.

Existing local asset names remain stable for callers. The following names map
to differently named ReIcon glyphs; other ReIcon assets use the same name:

| Local name | ReIcon name | Local name | ReIcon name |
| --- | --- | --- | --- |
| `check-check` | `check-read` | `circle-alert` | `alert-circle` |
| `circle-check` | `check-circle` | `circle-help` | `help-circle` |
| `circle-user-round` | `user-circle` | `corner-down-left` | `reply` |
| `ellipsis` | `more-h` | `external-link` | `arrow-up-right-square` |
| `flag` | `flag4` | `pin` | `thumbtack` |
| `grip-vertical` | `reorder` | `heading` | `text-square` |
| `id-card` | `user-id` | `image-off` | `gallery-slash` |
| `info` | `info-circle` | `key-round` | `key` |
| `keyboard-hide` | `keyboard` | `languages` | `language` |
| `layout-grid` | `grid` | `list-ordered` | `ordered-list` |
| `log-out` | `logout` | `mail` | `envelope` |
| `quote` | `quote-down` | `redo-2` | `redo` |
| `refresh-cw` | `refresh` | `share-2` | `share` |
| `sliders-horizontal` | `slider-horizontal` | `smartphone` | `mobile` |
| `smile` | `face-smile` | `square-pen` | `pen-square` |
| `trash-2` | `trash` | `type` | `text` |
| `undo-2` | `undo` | `university` | `building` |
| `user-round-check` | `user-check` | `user-round-plus` | `user-add` |
| `user-round` | `user` | `users-round` | `users` |
| `message-circle` | `chat` |  |  |

The `-filled` suffix selects the Filled weight of the same mapped icon.
`gallery-duotone` layers the ReIcon Gallery Filled shape at low opacity behind
the matching Outline strokes for the short-post image picker.
The `circle`, `circle-dot`, and `strikethrough` assets remain control glyphs
because ReIcon has no matching radio or strikethrough symbol; see
[LICENSE-lucide.txt](LICENSE-lucide.txt). Provider and social brand marks
(`bilibili`, `github`, `google`, `linkedin`, `twitter`, `weibo`, `zhihu`)
retain their artwork. Google and GitHub retain the approved Figma marks; the
other brand assets keep their [Simple Icons license](LICENSE-simple-icons.txt).
Google keeps its original colors. Interactive wrappers provide accessible
names; SVGs are decorative.

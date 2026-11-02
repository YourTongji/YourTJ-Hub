# Channel wording

| Channel | File | Include |
| --- | --- | --- |
| Web | web.zh-CN.md | Shipped forum website behavior, simplified Chinese |
| Operators | operators.zh-CN.md | Required migration/configuration/server compatibility work |
| Android | android.zh-CN.md | Android-visible changes only, simplified Chinese |
| App Store | ios.zh-Hans.txt | iOS changes since the last live store version; plain text, at most 4000 characters |
| TestFlight | testflight.en-US.txt | iOS-specific testing instructions, plain English |

Shared Flutter functionality may appear separately in both Android and iOS if it actually works
on both. An iOS Widget fix belongs only to iOS; Android notification permission belongs only to
Android. Backend or status-site work is not a mobile feature merely because it shares the repository.

A store channel What's New file must not name the other mobile platform. This is a hard rule
with no comparative-QA exception: do not mention the other platform in order to say testers
should ignore it. `validate_note_text` / `validate_candidate` enforce it (`workflow.py validate-pr`
and `cli.py validate`). Validate previously checked shape, identity and disclosures only; it did
not prove this semantic boundary. The closed token sets are:

| Channel | Must not contain |
| --- | --- |
| `ios.zh-Hans.txt`, `testflight.en-US.txt` | Android, 安卓, APK |
| `android.zh-CN.md` | iOS, iPhone, iPad, iPadOS, App Store, TestFlight, 苹果 |

The same words on that channel's own platform are allowed, including an iOS system requirement.
`web.zh-CN.md` and `operators.zh-CN.md` are not store What's New files and are not checked.
ASCII tokens match case-insensitively, including when written next to Chinese, and do not match
inside a longer Latin word. A structured changelog entry fails when its title or summary contains
one of these tokens for a platform listed in `platforms`. An entry with no platform token may
still list both `android` and `ios`. Already merged note and changelog bytes that predate the
gate stay loadable only for their named historical candidate/channel/file and exact digest, so
catalog reload and recovery keep working. Editing them, or copying the whole text into a new
candidate, is checked again.
Do not rewrite those files to satisfy the gate. That changes the approved content digest.
Account changes, automatic analytics and other required disclosures cannot be removed for brevity.
Do not copy the static `apps/mobile/store/zh-Hans/metadata.json` What's New into another channel.
No unsupported “performance/stability improvements” filler. Reverted work is absent from net changes.
Oryn uncertainties and truncated evidence need reviewer attention, not confident paraphrases.

Schema-2 mobile entries are read in the app's update prompt. Keep titles to a short phrase of
about 12 Chinese characters or fewer, and put the detail in one summary sentence that does not
repeat the title. Check that each drafted kind matches the user-visible change; the PR type it
was derived from describes the code change, not always what users notice. Keep `pr-<number>`
IDs, and replace remaining `oryn-` IDs with semantic slugs.

Edit the structured changelog and render the selected platform files together. New changelogs use
`notesFormatVersion: 2`: App Store copy has plain-text section titles, blank lines and `•` bullets,
with required notices first and empty sections omitted. Android keeps Markdown; TestFlight keeps
English testing instructions. Keep the entire App Store file within 4000 characters, including
formatting. Do not hand-format the text file separately: validation rejects divergence from the
structured facts. Existing candidates without a format field retain their original bytes; follow
the runbook's explicit upgrade procedure before a fresh human review, never during approved recovery.

Keep the three iOS display surfaces distinct. The app's native update prompt and history render
structured formal-release facts; iOS history has no channel picker. App Store What's New is the
plain-text `ios.zh-Hans.txt` rendering. TestFlight receives `testflight.en-US.txt` as plain-text
testing instructions for Apple's UI, not as a second user-facing history tab or automatic in-app
beta-notes prompt. Missing formal history must not fall back to TestFlight copy.

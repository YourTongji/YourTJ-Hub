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
Account changes, automatic analytics and other required disclosures cannot be removed for brevity.
Do not copy the static `apps/mobile/store/zh-Hans/metadata.json` What's New into another channel.
No unsupported “performance/stability improvements” filler. Reverted work is absent from net changes.
Oryn uncertainties and truncated evidence need reviewer attention, not confident paraphrases.

Schema-2 mobile entries are read in the app's update prompt. Keep titles to a short phrase of
about 12 Chinese characters or fewer, and put the detail in one summary sentence that does not
repeat the title. Check that each drafted kind matches the user-visible change; the PR type it
was derived from describes the code change, not always what users notice. Keep `pr-<number>`
IDs, and replace remaining `oryn-` IDs with semantic slugs.

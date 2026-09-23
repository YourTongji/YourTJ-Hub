# Campus map

> Doc type: product contract
>
> Status: Active
>
> Owner: Platform maintainers
>
> Last verified: 2026-09-23

The campus map helps people find a building or sports facility and understand its
position among nearby paths and landmarks. It is available to anonymous visitors
at `/map`, with an entry in the community navigation.

## Supported behavior

| Capability | Status | Contract |
|---|---|---|
| Web map | Current | A standalone responsive Vue page with pan, zoom, north/reset controls, a collapsible search/detail panel and 2D/2.5D views. |
| Place discovery | Current | Search source names, aliases, raw activity tags and activity names in the selected language; filter academic, library, food, living and sports places. Unnamed outdoor sports polygons remain discoverable. Golf grounds and sports centres are included even when their source has no explicit activity tag; 嘉定高尔夫练习场 is searchable with 高尔夫 or 高尔夫球 and appears in the golf filter. |
| Map labels | Current | Named buildings participate in label placement at campus overview scale; screen-space collision avoidance controls density. Overview labels omit redundant campus prefixes and parenthesized department suffixes, while hover, selection and search retain full names. Selected places, campus landmarks and libraries take precedence. |
| Place selection | Current | A map label, building polygon or result opens a detail card and moves the map to the selected feature. Closing restores the place list. |
| Place links | Current | `?campus=<campus ID>#place=<source feature ID>` restores a selected place. Copying a link falls back to a selectable URL if clipboard access fails. |
| Map loading failure | Current | The place list remains usable when WebGL fails; data/network failures show an explicit retry state. |
| Campus switching | Current | Siping, Jiading, Huxi, Hubei, Zhangjiang and Lingang each have an independent dataset. Switching cancels obsolete loads and clears the previous search/selection. |
| Sports drawing | Current | Siping, Jiading and Huxi athletics tracks have red running surfaces, green infields, lanes and football markings. Hubei follows the official straight-track layout. Individual supported court footprints receive markings; aggregated court polygons do not imply a court count. |
| Current location | Current | Explicit button press requests one browser WGS84 fix. A blue point and geodesic accuracy circle show it; a nearby calibrated campus is selected automatically. Outside-campus, denial, unsupported, timeout and unavailable states are explicit. A fix received before the renderer is ready is focused after loading. The uncalibrated Zhangjiang plan reports an outside-campus fix without promising a position overlay. No navigation or continuous tracking. |
| Coverage and facility detail | Partial | Hubei and Lingang have approximately aligned official-plan building traces. Zhangjiang has the four named buildings and paths in an explicitly uncalibrated local plan: location may be obtained but is not drawn on that plan. Indoor rooms, entrances, court counts and live venue information are not fully verified. |
| Personal timetable on map | Partial | The `/campus` Today section header has one generic “view my courses on the map” link beside “View Week”; it opens the existing map with the timetable tab selected. The map menu switches between place browsing, the private personal timetable, and a course-module schedule search while keeping the same map, campus selector, and map controls. The private view verifies the forum session and campus binding, then reads the existing personal timetable APIs into an in-memory searchable list. The schedule search composes the existing PK `courses-by-time` and batched `course-details` APIs, supporting term, weekday, period group and teaching-week filters. Both views retain original campus/room text; a uniquely matched campus map building can be pinned. The intent URL contains no course identity or location; canonical links stay `/map`. |
| Timetable-to-map building matching | Partial | Existing explicit Siping and Jiading mappings are combined with unique exact matches against names and clear aliases in the selected campus GeoJSON. Ambiguous or unmatched locations keep their original text and do not place a pin. These matches are project-level name matches, not school-verified building IDs or `towerCode` mappings. |
| Building and classroom schedules | Partial | The existing map place detail opens a schedule query scoped to that building. It filters the PK course module's teaching arrangements by term, weekday, period group and teaching week, and displays the arrangement period, room, course, teacher, query date, source and latest successful PK-module sync date when available. An empty result means no matching arrangement was returned; request failure is shown separately. Coverage and freshness follow the module's synchronized data, which does not establish complete classroom schedules or live occupancy. No result may be called a free, open or reservable room. |
| Native mobile | Partial | The shared page-component identifier is mirrored in Dart; the Flutter app has no native campus-map screen. Mobile browsers use the responsive Web page. |

The UI uses a fixed cartographic palette, with green grounds, blue water, warm
building roofs and distinct sports surfaces. 2.5D building heights and sports markings are illustrative.
The decorative campus-name/English-name/coordinate caption is absent; campus
identity is controlled by the header selector.
Place details retain source-derived names; activity controls and descriptions follow the selected
interface language. Coverage notes appear in the
map information dialog. They do not advertise live venue status.

## Data and deployment boundary

The page uses MapLibre GL JS and versioned GeoJSON assets. Both the renderer and
data load only when visiting the map. A same-origin CSP worker avoids changing the
forum's script policy. Labels use the local DOM/font stack instead of remote glyph
services. No map API token, external tile service, additional database, PMTiles
service, or Cloudflare Worker is required.

The personal timetable menu tab is requested with `?mine=1`; this is only an intent
flag, not a shareable course URL or a separate map page. It keeps the original map
and switches the explorer menu between places and courses. Its response is `private, no-store`, props contain
no timetable data, and the canonical URL omits the flag. Course records are fetched
from the existing private campus APIs after session and binding checks, kept in page
memory, and cleared when the page is left or its binding changes. The course search
uses the existing PK scheduler's public read APIs; it is an additional mode inside
the same map menu and does not expose personal timetable records. Dataset
`updatedAt` is the server time when the normalized response is generated, not the
school's last-change timestamp; the timetable menu does not display it as a freshness
claim.

The Go handler returns `campus.map` through the existing HTML/page-payload renderer.
Vite includes map assets in `resource/static/dist`, which the forum embeds in its
single binary. There is no new JSON API or authentication system. Location coordinates stay in Vue memory, are never put in links, browser storage,
analytics events or application requests, and are discarded on document exit.
Browser/OS positioning services retain their own permission and provider behavior.
The page does not publish collaborative annotations.

Only the `GET /map` document permits same-origin geolocation through
`Permissions-Policy`; camera and microphone remain denied. Other documents and
maintenance responses retain the deny-all policy. Entering or leaving the atlas
uses a full document navigation, including programmatic router navigation, because
SPA payload changes cannot change a document permission policy. Native browser
permission is still required; HTTPS or localhost is necessary. A manual campus
switch invalidates a pending fix so its late callback cannot override the user.

The source and transformation contract, including ODbL attribution and reproducible
import instructions, is in the [data README](../../apps/gooseforum/resource/src/site/campus-map/data/README.md).
The importer marks feature centers inside the supplied campus boundary as
campus destinations. Surrounding buildings remain visual context. University
facilities outside that boundary are not guaranteed to appear in search.

## Verification

- `resource/test/campus-map.test.ts` covers unnamed sports discovery, alias search,
  stable place IDs, campus-boundary filtering, geometry bounds, all six datasets, sports geometry, location/error handling and data privacy.
- `resource/test/campus-map-page.test.ts` covers translated sports discovery, cached fixes before
  canvas readiness, the uncalibrated-plan location notice and data/renderer failure recovery.
- `app/http/controllers/forum/campus_map_test.go` covers anonymous HTML and page
  payload responses.
- `app/http/middleware/securityHeaders_test.go` verifies the map-only location policy.
- Web type checking, the shared client tests, the production asset build, and
  desktop/mobile browser checks validate the page integration.

See [decision 0022](../decisions/0022-campus-map-native-atlas.md) for the reuse boundary.

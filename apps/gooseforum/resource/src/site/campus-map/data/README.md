# Campus map data

OpenStreetMap-derived geometry is licensed under the
[Open Database License 1.0](https://opendatacommons.org/licenses/odbl/1-0/).
The UI keeps **© OpenStreetMap contributors** and the
[OSM copyright link](https://www.openstreetmap.org/copyright) visible.
The data license is separate from the application-code license. Official school
plans remain separately attributed; this repository includes reconstructed factual
outlines and names, not the source PDF/JPEG artwork.

## Sources and coordinate systems

| Dataset | Boundary / source | Coordinates |
|---|---|---|
| Siping | OSM relation 18788116, pinned YourTJ Pulse snapshot | WGS84 |
| Jiading | OSM way 384531657 | WGS84 |
| Huxi | OSM relation 18781565 | WGS84 |
| Hubei | OSM way 538413905; official plan building supplement | WGS84, approximate building alignment |
| Lingang | OSM way 1107715623; official plan building supplement | WGS84, approximate building alignment |
| Zhangjiang | Official plan: 闻智楼、广智楼、博智楼、见智楼 | Local schematic coordinates; **not WGS84** |

- Siping source: [YourTJ Pulse `data/full.geojson`, commit 4a8d2925](https://github.com/WALKERKILLER/YourTJ-Pulse/blob/4a8d2925dd2b7a5d654a1c2eda43a3adb5fde45a/data/full.geojson).
  SHA-256: `1a3611d03ef79c46b6933f8ab83a91f79e0f6ea690648337983f4bad3d60103b`.
- Other OSM snapshots were retrieved on 2026-09-13 through the
  [OSM map API](https://wiki.openstreetmap.org/wiki/API_v0.6#Retrieving_map_data_by_bounding_box:_GET_/api/0.6/map).
  `sources.json` records request bounds and SHA-256 digests. Raw snapshots remain in
  ignored `research/`; contributor IDs, changesets and edit metadata are stripped.
- Official plan references are the four May 2025 PDFs and the two sports-distribution
  guides supplied for this task. The [Tongji map collection](https://photo.tongji.edu.cn/zt/tjLOGOyxydt.htm)
  identifies the campus-plan source. Supplements are curated GeoJSON, not automatic
  OCR output. Their `source` property distinguishes approximate traces/classification
  from OSM geometry. Indoor locations and current building uses require separate checks.
- Hubei plan alignment uses NW, NE and SW boundary corners at
  `(121.4532596,31.2644635)`, `(121.4551873,31.2650629)`,
  `(121.4542049,31.262179)`. Lingang uses west, SW and SE corners at
  `(121.91885,30.8658845)`, `(121.9195879,30.8642053)`,
  `(121.9228112,30.8654961)`. Affine alignment of illustrated plans is approximate,
  not a surveyed entrance or indoor location.
- Zhangjiang's `coordinateMode: schematic` is intentional. Its small coordinates
  belong to the plan only and are excluded from geographic campus matching and
  position overlays. [The university confirms the address](https://srias.tongji.edu.cn/17839/list.htm),
  but an address alone is insufficient to calibrate these four footprints.

## Transformation and maintenance

`campuses.json` owns campus IDs, geographic/local bounds, boundary IDs and camera
orientation. `*-supplement.geojson` files own manually checked outlines and overrides;
matching source IDs replace existing features. Maintain them before regenerating.

The importer accepts a GeoJSON snapshot or OSM API JSON (`osmtogeojson`, development
only). It keeps map-related tags, strips contributor metadata, excludes linear
transport relations, derives bounding-box label centers and categories, and tests
centers against Polygon/MultiPolygon campus boundaries (including holes). Context
outside campus stays visible but is excluded from discovery. Boundaries may omit
university facilities outside the main grounds. Heights are capped illustrative
estimates, never measured elevations.

Sports classification also includes `leisure=golf_course` and `sports_centre`,
which may have no `sport` tag. The catalog infers golf from `golf_course`, just as
it infers running from tracks and swimming from pools. Jiading's named golf practice
ground (`way/263927462`, facility 13 in the supplied sports guide) and nearby unnamed
golf footprint (`way/1456436384`) retain their OSM geometry and IDs. Both appear in
the golf filter and match searches for 高尔夫 and 高尔夫球; only the named ground is
identified by the guide. Indoor activity lists are never inferred from a generic
sports-centre tag.

From `apps/gooseforum/resource`:

```bash
node scripts/import-campus-map.mjs siping /absolute/path/to/full.geojson
node scripts/import-campus-map.mjs jiading /absolute/path/to/jiading.osm.json
node scripts/import-campus-map.mjs huxi /absolute/path/to/huxi.osm.json
node scripts/import-campus-map.mjs hubei /absolute/path/to/hubei.osm.json
node scripts/import-campus-map.mjs lingang /absolute/path/to/lingang.osm.json
node scripts/import-campus-map.mjs zhangjiang /absolute/path/to/empty.geojson
```

For Zhangjiang, `empty.geojson` is an empty FeatureCollection; the maintained
supplement supplies the plan. No external tiles or remote API are requested by the
browser. Vite bundles the six derived assets in the forum binary; switching fetches
only the selected dataset. Source snapshots are research inputs, not build inputs.

`sports-details.ts` derives decorative red/green surfaces and linework from existing
sports footprints. It does not create measured lane/court counts or indoor floor
plans. The Hubei override explicitly preserves the straight west-side strip.

See the [product contract](../../../../../../../docs/product/campus-map.md)
for supported behavior and limits.

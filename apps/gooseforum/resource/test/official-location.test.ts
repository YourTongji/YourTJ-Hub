import { readFileSync } from 'node:fs'
import { expect, it } from 'vitest'
import type { CampusData } from '../src/site/campus-map/catalog'
import { officialCampusId, officialLocationApplies, officialLocationTarget, officialLocationTargets, parseOfficialLocation, parseOfficialLocations } from '../src/site/campus-map/official-location'

const jiadingData = JSON.parse(readFileSync(new URL('../src/site/campus-map/data/jiading.geojson', import.meta.url), 'utf8')) as CampusData

it.each([
  ['北楼115室（单周）', '单周', 1, 1, true],
  ['北楼115室（单周）', '单周', 2, 1, false],
  ['北楼115（双周）', '双周', 2, 1, true],
  ['北楼115（双周）', '双周', 1, 1, false],
  ['北楼115室（9月24日）', '9月24日', 2, 2, false],
  ['单周北楼115室（第3-4周）', '单周且第3-4周', 3, 1, true],
  ['单周北楼115室（第3-4周）', '单周且第3-4周', 1, 1, false],
  ['单周北楼115室（第3-4周）', '单周且第3-4周', 4, 1, false],
  ['单周北楼115室（双周）', '单周且双周', 1, 1, false],
  ['单周北楼115室（9月24日）', '单周且9月24日', 1, 1, false],
  ['北楼115室（周四）（第3周）', '周四且第3周', 3, 4, true],
  ['北楼115室（周四）（第3周）', '周四且第3周', 3, 5, false],
  ['北楼115室【15:25～17:05，第一周自2026年9月7日始】', '15:25～17:05，第一周自2026年9月7日始', 1, 1, false],
  ['北楼115室（时间另行通知）', '时间另行通知', 1, 1, false],
])('retains suffix conditions and requires every time restriction: %s (%s), week %s day %s', (raw, condition, week, day, applies) => {
  const location = officialLocationTargets('四平路校区', raw)[0]!
  expect(location.raw).toBe(raw)
  expect(location.condition).toBe(condition)
  expect(location.target?.featureId).toBe('way/183383474')
  expect(officialLocationApplies(location, { week, day })).toBe(applies)
})

it('retains ordinary room notes without inventing a time condition', () => {
  const location = officialLocationTargets('四平路校区', '北楼321智慧教室(中)')[0]!
  expect(location.room).toBe('321智慧教室(中)')
  expect(location.condition).toBe('')
  expect(officialLocationApplies(location, { week: 2, day: 1 })).toBe(true)
  const data = mapData(['court', '小足球场'])
  data.features[0]!.properties.category = 'sport'
  const numbered = officialLocationTargets('嘉定校区', '小足球场（2号）', data)[0]!
  expect(numbered.building).toBe('小足球场（2号）')
  expect(numbered.condition).toBe('')
  expect(numbered.target).toBeUndefined()
})

it('carries list prefixes without carrying one room suffix into the next room', () => {
  const parts = officialLocationTargets('嘉定校区', '单周A101、B201（第3周）', jiadingData)
  expect(parts.map(part => part.condition)).toEqual(['单周', '单周且第3周'])
  expect(parts.map(part => part.target?.featureId)).toEqual(['way/266167562', 'way/263922904'])
  const rooms = officialLocationTargets('四平路校区', '北楼115室（单周）、116室（双周）')
  expect(rooms.map(part => part.condition)).toEqual(['单周', '双周'])
  expect(rooms.map(part => officialLocationApplies(part, { week: 2, day: 1 }))).toEqual([false, true])
})

it('resolves bare building abbreviations independently using the actual Jiading aliases', () => {
  const parts = officialLocationTargets('嘉定校区', 'A101、201、B301', jiadingData)
  expect(parts.map(part => part.target?.featureId)).toEqual(['way/266167562', 'way/266167562', 'way/263922904'])
  expect(officialLocationTargets('嘉定校区', 'A101、B201', jiadingData).map(part => part.target?.featureId))
    .toEqual(['way/266167562', 'way/263922904'])
  expect(officialLocationTarget('嘉定校区', 'A101、B201', jiadingData)).toBeUndefined()
  expect(officialLocationTarget('嘉定校区', 'A101、A102', jiadingData)?.featureId).toBe('way/266167562')
  expect(officialLocationTargets('嘉定校区', '安楼A101、A102', jiadingData).map(part => part.target?.featureId))
    .toEqual(['way/266167562', 'way/266167562'])
  expect(officialLocationTargets('嘉定校区', '安楼A101、102、A103', jiadingData).map(part => part.target?.featureId))
    .toEqual(['way/266167562', 'way/266167562', 'way/266167562'])
})

it('does not recover an unknown or ambiguous bare building by inheriting the previous target', () => {
  expect(officialLocationTargets('嘉定校区', 'A101、Z201', jiadingData)[1]?.target).toBeUndefined()
  const ambiguous = mapData(['a', '安楼（A楼）'], ['b', '博楼（B楼）'], ['other', '其他楼（B楼）'])
  expect(officialLocationTargets('嘉定校区', 'A101、B201', ambiguous)[1]?.target).toBeUndefined()
})

it('keeps different lettered rooms under an explicit parent uncertain until a new building is named', () => {
  const parts = officialLocationTargets('嘉定校区', '济事楼A101、B201、202、博楼B203', jiadingData)
  expect(parts.map(part => part.target?.featureId)).toEqual(['way/135405205', undefined, undefined, 'way/263922904'])
  expect(officialLocationTargets('嘉定校区', '学院教室A101、B201', jiadingData).every(part => !part.target)).toBe(true)
})

it('splits only a clearly separated building phrase and room token for display', () => {
  expect(parseOfficialLocation('济事楼（软件学院） A101')).toEqual({
    building: '济事楼（软件学院）',
    room: 'A101',
  })
})

it.each([
  ['北115', { building: '北', room: '115' }],
  ['济事楼201', { building: '济事楼', room: '201' }],
  ['教学北楼 115 室', { building: '教学北楼', room: '115' }],
  ['嘉定校区 D404', { building: 'D', room: '404' }],
])('splits an explicit building prefix and room number for display: %s', (value, expected) => {
  expect(parseOfficialLocation(value, value.startsWith('嘉定') ? '嘉定校区' : '')).toEqual(expected)
})

it.each(['未知地点', '2024秋季1班'])('%s remains raw when it has no reliable building-room boundary', (value) => {
  expect(parseOfficialLocation(value)).toBeNull()
})

it.each([
  ['瑞安楼阶3', { building: '瑞安楼', room: '阶3' }],
  ['诚楼C408A', { building: '诚楼', room: 'C408A' }],
  ['物理馆2-4楼', { building: '物理馆', room: '2-4楼' }],
  ['济事楼419实验室', { building: '济事楼', room: '419实验室' }],
  ['开物馆机房A211', { building: '开物馆', room: '机房A211' }],
])('extracts real classroom descriptions without turning them into building aliases: %s', (value, expected) => {
  expect(parseOfficialLocation(value)).toEqual(expected)
})

it('maps a named sports venue without requiring a building footprint', () => {
  const data = mapData(['court', '网球场'])
  data.features[0]!.properties.category = 'sport'
  expect(officialLocationTarget('嘉定校区', '网球场', data)).toEqual({ campusId: 'jiading', featureId: 'court' })
})

it('does not split conjunction characters inside a building name', () => {
  const data = mapData(['zhonghe', '衷和楼'])
  expect(officialLocationTarget('四平路校区', '衷和楼201', data)).toEqual({ campusId: 'siping', featureId: 'zhonghe' })
  expect(parseOfficialLocations('衷和楼201')).toHaveLength(1)
})

it('preserves omitted buildings and week conditions across room lists', () => {
  const data = mapData(['ruian', '瑞安楼'], ['physics', '物理馆'])
  const parts = officialLocationTargets('四平路校区', '单周瑞安楼403、505、双周物理馆301、302', data)
  expect(parts.map(({ building, room, condition }) => ({ building, room, condition }))).toEqual([
    { building: '瑞安楼', room: '403', condition: '单周' },
    { building: '瑞安楼', room: '505', condition: '单周' },
    { building: '物理馆', room: '301', condition: '双周' },
    { building: '物理馆', room: '302', condition: '双周' },
  ])
  expect(parts.map(part => part.target?.featureId)).toEqual(['ruian', 'ruian', 'physics', 'physics'])
  expect(officialLocationTarget('四平路校区', '瑞安楼403，物理馆301', data)).toBeUndefined()
  expect(officialLocationTarget('四平路校区', '瑞安楼403、505', data)?.featureId).toBe('ruian')
})

it('keeps unresolved parts instead of treating a partial match as the only location', () => {
  const data = mapData(['a', '安楼（A楼）'])
  const parts = officialLocationTargets('嘉定校区', '安楼A101；实验室', data)
  expect(parts.map(part => part.raw)).toEqual(['安楼A101', '实验室'])
  expect(parts[0]?.target?.featureId).toBe('a')
  expect(parts[1]?.target).toBeUndefined()
  expect(officialLocationTarget('嘉定校区', '安楼A101；实验室', data)).toBeUndefined()
})

it.each([
  '学院教室 B403、B405、B407',
  '嘉定机房F214A/F214B/F217',
  '工程实践中心彰武路100号A楼、B楼',
  '彰武路100号，工程实践中心A楼、B楼',
  '未核验楼体F214A/F214B',
])('retains unknown qualifiers across abbreviated continuations: %s', (value) => {
  const data = mapData(['b', '红楼（B楼）'], ['f', '诚楼（F楼）'])
  expect(officialLocationTargets('四平路校区', value, data).every(part => !part.target)).toBe(true)
})

it('inherits a named building for lettered room continuations', () => {
  const data = mapData(['a', '安楼（A楼）'], ['b', '博楼（B楼）'])
  const parts = officialLocationTargets('嘉定校区', '安楼A101、A102、双周A103', data)
  expect(parts.map(part => part.building)).toEqual(['安楼', '安楼', '安楼'])
  expect(parts.map(part => part.room)).toEqual(['A101', 'A102', 'A103'])
  expect(parts.every(part => part.target?.featureId === 'a')).toBe(true)
})

it('inherits conditions when continuing a lettered building list', () => {
  const data = mapData(['a', '甲楼（A楼）'], ['b', '乙楼（B楼）'])
  const parts = officialLocationTargets('嘉定校区', '单周A楼、B楼', data)
  expect(parts.map(part => part.target?.featureId)).toEqual(['a', 'b'])
  expect(parts.map(part => part.condition)).toEqual(['单周', '单周'])
  expect(parts.every(part => !officialLocationApplies(part, { week: 2, day: 1 }))).toBe(true)
})

it('rejects explicit campus conflicts and keeps unknown conditions for manual selection', () => {
  const data = mapData(['a', '安楼（A楼）'])
  expect(officialLocationTargets('嘉定校区', '沪北安楼A101', data)[0]?.target).toBeUndefined()
  const location = officialLocationTargets('嘉定校区', '后8周安楼A101', data)[0]!
  expect(location.condition).toBe('后8周')
  expect(location.target?.featureId).toBe('a')
  expect(officialLocationApplies(location, { week: 12, day: 1 })).toBe(false)
})

it('retains an explicit campus across omitted rooms and later places', () => {
  const parts = officialLocationTargets('四平路校区', '嘉定校区北楼115、116、南203')
  expect(parts).toHaveLength(3)
  expect(parts.every(part => !part.target)).toBe(true)
  expect(officialLocationTargets('四平路校区', '北楼115室（嘉定、沪西）')[0]?.target).toBeUndefined()
})

it('keeps alternatives uncertain even when they also have a weekday condition', () => {
  const parts = officialLocationTargets('四平路校区', '周一北115或南203')
  expect(parts.map(part => part.target?.featureId)).toEqual(['way/183383474', 'way/183383472'])
  expect(parts.every(part => !officialLocationApplies(part, { week: 1, day: 1 }))).toBe(true)
})

it('does not remove inherited or pending condition text from a place name', () => {
  const data = mapData(['a', '安楼'], ['court', '网球场'], ['hall', '体育中心篮球馆'])
  data.features[1]!.properties.category = 'sport'
  const parts = officialLocationTargets('嘉定校区', '单周，安楼A101、102；双周，网球场', data)
  expect(parts.map(part => part.target?.featureId)).toEqual(['a', 'a', 'court'])
  expect(parts.map(part => part.condition)).toEqual(['单周', '单周', '双周'])
  const exact = officialLocationTargets('嘉定校区', '单周，体育中心篮球馆', data)[0]!
  expect(exact.building).toBe('体育中心篮球馆')
  expect(exact.room).toBe('')
  const rooms = officialLocationTargets('嘉定校区', '体育中心篮球馆、201、202', data)
  expect(rooms.map(part => part.target?.featureId)).toEqual(['hall', 'hall', 'hall'])
})

it.each([
  ['第3-4周', 3, 1, true], ['第3-4周', 5, 1, false],
  ['单周', 1, 1, true], ['单周', 2, 1, false], ['双周', 2, 1, true],
  ['周四', 1, 4, true], ['周四', 1, 5, false], ['【9月24日】', 1, 1, false],
])('filters only interpretable location conditions: %s', (condition, week, day, applies) => {
  expect(officialLocationApplies({ raw: '', building: '', room: '', condition }, { week, day })).toBe(applies)
})

it('does not split delimiters inside parentheses or map online announcements', () => {
  expect(parseOfficialLocations('同济大学图书馆（旧名，备注）201')).toHaveLength(1)
  const data = mapData(['a', '安楼'])
  expect(officialLocationTargets('嘉定校区', '在线体育课，关注Canvas安楼101通知', data).some(part => part.target)).toBe(false)
})

it('does not map ambiguous sports names, excluded categories or numeric fragments', () => {
  const data = mapData(['a', '网球场'], ['b', '网球场'], ['living', '12'])
  data.features[0]!.properties.category = 'sport'
  data.features[1]!.properties.category = 'sport'
  expect(officialLocationTarget('嘉定校区', '网球场', data)).toBeUndefined()
  expect(officialLocationTarget('嘉定校区', '12', data)).toBeUndefined()
  data.features[1]!.properties.campus = false
  expect(officialLocationTarget('嘉定校区', '网球场', data)?.featureId).toBe('a')
})

it.each([
  ['四平路校区', '北楼', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '教学北楼', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '南楼', { campusId: 'siping', featureId: 'way/183383472' }],
  ['四平路校区', '教学南楼', { campusId: 'siping', featureId: 'way/183383472' }],
  ['四平路校区', '北115', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '北楼 115', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '教学北楼115', { campusId: 'siping', featureId: 'way/183383474' }],
  ['四平路校区', '南楼 203', { campusId: 'siping', featureId: 'way/183383472' }],
  ['四平路校区', '教学南楼203', { campusId: 'siping', featureId: 'way/183383472' }],
  ['嘉定校区', '济事楼', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事南楼', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事北楼', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事楼（软件学院）', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事南楼 A101', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事北楼A101', { campusId: 'jiading', featureId: 'way/135405205' }],
  ['嘉定校区', '济事楼（软件学院） A101', { campusId: 'jiading', featureId: 'way/135405205' }],
])('maps only confirmed campus/building aliases: %s %s', (campus, room, target) => {
  expect(officialLocationTarget(campus, room)).toEqual(target)
})

function mapData(...names: [string, string][]): CampusData {
  return {
    type: 'FeatureCollection',
    features: names.map(([id, name]) => ({
      type: 'Feature',
      id,
      properties: { category: 'academic', center: [121, 31], campus: true, name },
      geometry: { type: 'Point', coordinates: [121, 31] },
    })),
  } as CampusData
}

it('maps obvious course building abbreviations from the selected campus dataset', () => {
  const jiading = mapData(['a', '安楼（A楼）'], ['b', '博楼（B楼）'])
  expect(officialCampusId('同济大学嘉定校区')).toBe('jiading')
  expect(officialLocationTarget('嘉定校区', 'D404', jiading)).toBeUndefined()
  expect(officialLocationTarget('嘉定校区', 'A101', jiading)).toEqual({ campusId: 'jiading', featureId: 'a' })
  expect(officialLocationTarget('嘉定校区', '博楼201', jiading)).toEqual({ campusId: 'jiading', featureId: 'b' })
})

it('does not guess when a course record names multiple campuses', () => {
  expect(officialCampusId('四平路校区、嘉定校区')).toBeUndefined()
})

it('does not pin an abbreviation when it maps to multiple buildings', () => {
  const data = mapData(['a', '甲楼（A楼）'], ['b', '乙楼（A楼）'])
  expect(officialLocationTarget('嘉定校区', 'A101', data)).toBeUndefined()
})

it.each([
  ['嘉定校区', '北115'],
  ['四平路校区', '济事北楼 A101'],
  ['沪西校区', '北楼115'],
  ['四平路校区', '北'],
  ['四平路校区', '南'],
])('does not guess a map building for unconfirmed or incomplete locations: %s %s', (campus, room) => {
  expect(officialLocationTarget(campus, room)).toBeUndefined()
})

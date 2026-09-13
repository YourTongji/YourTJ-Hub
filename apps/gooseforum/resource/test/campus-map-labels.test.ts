import { describe, expect, it } from 'vitest'
import type { CampusPlace } from '../src/site/campus-map/catalog'
import { layoutCampusLabels } from '../src/site/campus-map/labels'

function place(
  id: string,
  name: string,
  center: [number, number],
): CampusPlace {
  return {
    id,
    name,
    center,
    aliases: [],
    category: 'academic',
    sports: [],
    indoor: true,
    named: true,
    feature: {
      type: 'Feature',
      id,
      geometry: { type: 'Point', coordinates: center },
      properties: {
        name,
        center,
        campus: true,
        category: 'academic',
        building: 'yes',
      },
    },
  }
}
const options = {
  width: 600,
  height: 400,
  zoom: 15,
  featured: [],
  project: ([x, y]: [number, number]) => ({ x, y }),
}
describe('campus labels at overview scale', () => {
  it('shows ordinary building names at overview zoom, without a featured whitelist', () => {
    const places = [
      place('a', '济事楼（软件学院）', [150, 140]),
      place('b', '通达馆', [320, 220]),
    ]
    expect(layoutCampusLabels(places, options).map((l) => l.place.id)).toEqual([
      'a',
      'b',
    ])
  })
  it('fits adjacent short names while preserving readable spacing', () => {
    const places = [
      place('a', '济人楼', [120, 150]),
      place('b', '济事楼', [180, 150]),
    ]
    expect(layoutCampusLabels(places, { ...options, zoom: 17 })).toHaveLength(2)
  })
  it('shortens department suffixes at overview scale and restores the full name on selection', () => {
    const p = place('a', '济事楼（软件学院）', [150, 140])
    expect(layoutCampusLabels([p], options)[0]?.text).toBe('济事楼')
    expect(
      layoutCampusLabels([p], { ...options, selectedId: 'a' })[0]?.text,
    ).toBe(p.name)
  })
  it('keeps the library identifiable when a nearby building competes for space', () => {
    const building = place('a', '安楼', [150, 150])
    const library = {
      ...place('b', '同济大学嘉定校区图书馆', [150, 150]),
      category: 'library' as const,
    }
    expect(
      layoutCampusLabels([building, library], options).map((l) => l.text),
    ).toEqual(['图书馆'])
  })
  it('keeps a selected place over a colliding label and excludes offscreen labels', () => {
    const places = [
      place('a', '济人楼', [150, 150]),
      place('b', '济事楼', [150, 150]),
      place('c', '校外楼', [-80, 140]),
    ]
    expect(
      layoutCampusLabels(places, { ...options, selectedId: 'b' }).map(
        (l) => l.place.id,
      ),
    ).toEqual(['b'])
  })
})

import type { CampusPlace } from './catalog'

export interface CampusLabel {
  place: CampusPlace
  x: number
  y: number
  text: string
}
interface LabelOptions {
  width: number
  height: number
  zoom: number
  selectedId?: string
  featured: string[]
  project: (center: [number, number]) => { x: number; y: number }
}
export function layoutCampusLabels(
  places: CampusPlace[],
  options: LabelOptions,
): CampusLabel[] {
  const width = options.width
  const height = options.height
  const zoom = options.zoom
  const occupied: Array<{ x: number; y: number; w: number; h: number }> = []
  const result: CampusLabel[] = []
  const ordered = [...places].sort((a, b) => {
    const priority = (p: CampusPlace) =>
      p.id === options.selectedId
        ? 100
        : options.featured.includes(p.name)
          ? 20
          : p.category === 'library'
            ? 18
            : p.named
              ? p.feature.properties.building || p.category === 'sport'
                ? 10
                : 5
              : 0
    return priority(b) - priority(a)
  })
  for (const place of ordered) {
    // Overview labels are limited by available screen space, never by a
    // zoom/whitelist gate. Named buildings take priority over unnamed courts.
    const point = options.project(place.center)
    const fullName = place.name.replace(/^同济大学/, '')
    const selected = place.id === options.selectedId
    const name =
      zoom < 16.6 && !selected
        ? fullName
            .replace(/^(?:嘉定|四平路?|沪西|沪北)校区/, '')
            .replace(/[（(][^）)]*[）)]/g, '')
            .trim() || fullName
        : fullName
    // Match the 12px labels, including the selected label's larger text box.
    const w = Math.max(
      28,
      [...name].reduce(
        (width, char) => width + (/[^\x00-\xff]/.test(char) ? 12 : 7),
        0,
      ) + (selected ? 24 : 6),
    )
    const h = selected ? 44 : 32
    if (
      point.x < w / 2 ||
      point.y < 20 ||
      point.x > width - w / 2 ||
      point.y > height - 25
    )
      continue
    if (
      occupied.some(
        (r) =>
          Math.abs(r.x - point.x) < (r.w + w) / 2 + 4 &&
          Math.abs(r.y - point.y) < (r.h + h) / 2 + 4,
      )
    )
      continue
    result.push({ place, x: point.x, y: point.y, text: name })
    occupied.push({ x: point.x, y: point.y, w, h })
    if (result.length >= 65) break
  }
  return result
}

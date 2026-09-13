import configs from './data/campuses.json'
import siping from './data/siping.geojson?url'
import jiading from './data/jiading.geojson?url'
import huxi from './data/huxi.geojson?url'
import hubei from './data/hubei.geojson?url'
import zhangjiang from './data/zhangjiang.geojson?url'
import lingang from './data/lingang.geojson?url'
export interface CampusConfig {
  coordinateMode?: string
  id: string
  boundaryId: string
  bounds: number[]
  bearing: number
  suggestions: string[]
  url: string
}
const urls: Record<string, string> = {
  siping,
  jiading,
  huxi,
  hubei,
  zhangjiang,
  lingang,
}
export const campuses: CampusConfig[] = configs.map((c) => ({
  ...c,
  url: urls[c.id]!,
}))
export function getCampus(id: string | null | undefined): CampusConfig {
  return campuses.find((c) => c.id === id) ?? campuses[0]!
}
export function containingCampus(
  longitude: number,
  latitude: number,
): CampusConfig | undefined {
  return campuses.find(
    ({ bounds: b, coordinateMode }) =>
      coordinateMode !== 'schematic' &&
      longitude >= b[0]! &&
      longitude <= b[2]! &&
      latitude >= b[1]! &&
      latitude <= b[3]!,
  )
}

/** Split a recognizable building prefix and room token for display.
 * Pin placement must go through the explicit campus-scoped mappings below.
 */
export function parseOfficialLocation(value: string): { building: string; room: string } | null {
  const match = value.trim().match(/^(.+?(?:楼|馆|中心|大厦)(?:（[^（）]+）|\([^()]+\))?|[北南东西]|[A-Za-z])\s*([A-Za-z]?\d{1,4}(?:[-/]\w{1,4})?)(?:\s*室)?$/u)
  if (!match) return null
  return { building: match[1], room: match[2] }
}

export function officialLocationTarget(campus: string, value: string): { campusId: 'siping' | 'jiading'; featureId: string } | undefined {
  const parsed = parseOfficialLocation(value)
  const building = parsed?.building ?? value.trim()
  if (campus === '四平路校区') {
    if (building && (building === '北' ? parsed !== null : ['北楼', '教学北楼'].includes(building)))
      return { campusId: 'siping', featureId: 'way/183383474' }
    if (building && (building === '南' ? parsed !== null : ['南楼', '教学南楼'].includes(building)))
      return { campusId: 'siping', featureId: 'way/183383472' }
  }
  if (
    campus === '嘉定校区' &&
    building &&
    ['济事楼', '济事南楼', '济事北楼', '济事楼（软件学院）'].includes(building)
  )
    return { campusId: 'jiading', featureId: 'way/135405205' }
}

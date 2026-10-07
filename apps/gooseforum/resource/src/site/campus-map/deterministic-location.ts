import places from './data/locations/places.json'
import type { ExtractedLocation, LocationExtraction, StablePlace } from './location-types'

export const placeCatalog: Record<string, StablePlace[]> = places.campuses
const key = (value: string) => value.normalize('NFKC').toLowerCase()
const room = /^(?:[A-Za-z]?\d+[A-Za-z]?(?:室|教室|实验室)?|[一二三四五六七八九十\d]+(?:楼|层))$/u
const numericContinuation = /^\d+[A-Za-z]?(?:室|教室|实验室)?$/u
const timePrefix = /^(?:上课)?(单周|双周|周[一二三四五六日天]|第?\d+(?:[-~～至]\d+)?周|[前后]\d+周)\s*/u
const timeNote = /^(?:单周|双周|周[一二三四五六日天]|第?\d+(?:[-~～至]\d+)?周|[前后]\d+周|\d+月\d+日)$/u

export function missingExtraction(raw: string): LocationExtraction {
  return { relation: 'single', locations: [{ source_text: raw, kind: 'unknown', place: null, detail: null,
    campus_text: null, address: null, time: null, conditions: [] }], unassigned_conditions: [], needs_review: false, review_reasons: [] }
}

/** Anchored longest verified-name match, not substring inference from model-produced place names. */
export function parseStableLocation(raw: string, campusId: string, catalog = placeCatalog): LocationExtraction {
  const entries = catalog[campusId] ?? []
  const names = entries.flatMap(place => [place.name, ...place.aliases].map(alias => ({ alias, place })))
    .sort((a, b) => b.alias.length - a.alias.length)
  const pieces = raw.split(/[、,，;；]/u)
  const locations: ExtractedLocation[] = [], unresolved: string[] = []
  let inherited: StablePlace | undefined
  let continued = false
  let hasSuffixNote = false
  for (const source_text of pieces) {
    let text = source_text.trim()
    const time: string[] = [], conditions: string[] = []
    // Only recognized prefixes are detached. Unknown leading context cannot expose an inner name.
    for (let match = text.match(timePrefix); match; match = text.match(timePrefix)) {
      time.push(match[1]!); text = text.slice(match[0].length)
    }
    const note = text.match(/[（(]([^（）()]*)[）)]$/u)
    if (note && !names.some(({ alias }) => key(alias) === key(text))) {
      hasSuffixNote = true
      // Preserve even unsupported notes. They never silently become an unconditional room.
      if (timeNote.test(note[1]!)) time.push(note[1]!)
      else conditions.push(note[1]! || note[0])
      text = text.slice(0, -note[0].length).trim()
    }
    const matches = names.filter(({ alias }) => key(text).startsWith(key(alias)))
      // User-confirmed 南/北 shorthand requires a numeric room; the direction alone is not a building.
      .filter(({ alias }) => !['南', '北'].includes(alias) || numericContinuation.test(text.slice(alias.length).trim()))
      .filter(({ alias }) => !text.slice(alias.length).trim() || room.test(text.slice(alias.length).trim()))
    const longest = matches[0]?.alias.length
    const candidates = matches.filter(match => match.alias.length === longest)
    const ids = new Set(candidates.map(match => match.place.featureId))
    let place = ids.size === 1 ? candidates[0]?.place : undefined
    let detail = place ? text.slice(candidates[0]!.alias.length).trim() : ''
    // Letters are never inherited. A/B/etc must resolve their own verified alias first.
    if (!place && !matches.length && inherited && numericContinuation.test(text)) {
      place = inherited; detail = text; continued = true
    }
    if (!place) {
      locations.push(missingExtraction(source_text).locations[0]!)
      unresolved.push(source_text)
      inherited = undefined
      continue
    }
    // A letter abbreviation is both a building identity and part of its room notation.
    if (/^[A-Za-z]$/u.test(candidates[0]?.alias ?? '')) detail = text
    locations.push({ source_text, kind: 'named', place: place.name, detail: detail || null,
      campus_text: null, address: null, time: time.length ? time : null, conditions })
    inherited = place
  }
  // Unknown members may be unrecognized conditions; continuation makes suffix scope ambiguous.
  const hasConditions = locations.some(member => member.time?.length || member.conditions.length)
  if (locations.length > 1 && (unresolved.length || (hasConditions &&
    (continued || hasSuffixNote || locations.some(member => !member.time?.length))))) {
    unresolved.push(...locations.flatMap(member => [...(member.time ?? []), ...member.conditions]))
  }
  return { relation: locations.length === 1 ? 'single' : locations.some(member => member.time?.length) ? 'conditional' : 'list',
    locations, unassigned_conditions: [...new Set(unresolved)], needs_review: false, review_reasons: [] }
}

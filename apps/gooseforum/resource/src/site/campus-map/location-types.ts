/** The offline prompt/result contract. Kept in sync with result.schema.json by tests. */
export type LocationKind = 'named' | 'generic' | 'online' | 'pending' | 'no_room' | 'unknown'
export type LocationRelation = 'single' | 'list' | 'conditional' | 'alternative' | 'mixed' | 'unclear'
export interface ExtractedLocation {
  source_text: string
  kind: LocationKind
  place: string | null
  detail: string | null
  campus_text: string | null
  address: string | null
  conditions: string[]
  time: string[] | null
}
export interface LocationExtraction {
  relation: LocationRelation
  locations: ExtractedLocation[]
  unassigned_conditions: string[]
  needs_review: boolean
  review_reasons: string[]
}
export interface LocationLookupContext {
  calendarId?: number
  /** School-calendar display name; only term-scoped overrides require it or a calendar ID. */
  term?: string
  dataCampusId?: string
}
export type LocationHint = LocationKind | 'missing' | 'review' | 'campus' | 'unmapped'
export interface LocationDictionarySource {
  schema_version: 'campus-location-extraction/v2'
  calendar_id: number
  term: string
  term_names: string[]
  audience: string
  synced_at: string
  review_status: 'review_2_pending' | 'reviewed'
  input_count: number
  source_sha256: string
  candidate_sha256: string
  model_requested: string
  run_fingerprint: string
  source_prompt_sha256: string
  correction_rule: string
}
export interface StablePlace {
  name: string
  aliases: string[]
  featureId: string
  evidence: string
}
export interface LocationOverride {
  campusId: string
  raw: string
  scope?: { calendarId: number; termNames: string[] }
  action: 'replace' | 'block'
  result: LocationExtraction
  reviewPending: boolean
  reason: string
  source: string
}
export interface LocationOverrideConfig {
  schema_version: 'campus-location-overrides/v1'
  rules: LocationOverride[]
}
export interface LocationDictionary {
  _meta: LocationDictionarySource & { sources?: LocationDictionarySource[] }
  dictionary: Record<string, Record<string, LocationExtraction>>
}

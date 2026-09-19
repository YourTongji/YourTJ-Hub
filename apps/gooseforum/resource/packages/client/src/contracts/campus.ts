// School callbacks use a clean 303 redirect even on session/account rejection.
import type { components } from '../gen/openapi.js'
export type CampusStatus = components['schemas']['CampusStatus']
export type CampusDataset = components['schemas']['CampusDataset']
export type CampusDatasetKey = CampusDataset['key']
export type CampusEvent = components['schemas']['CampusEvent']
export type CampusMessageSummary = components['schemas']['CampusMessageSummary']
export type CampusMessageDetail = components['schemas']['CampusMessageDetail']
export type CampusCalendarExport = components['schemas']['CampusCalendarExport']

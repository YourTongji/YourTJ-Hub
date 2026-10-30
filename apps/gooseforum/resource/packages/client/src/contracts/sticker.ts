export interface StickerItem {
  id?: number
  displayName?: string
  pack?: string
  isOfficial?: boolean
  isEnabled?: boolean
  /** Personal uploads only; older servers omit it and mean approved. */
  reviewStatus?: 'pending' | 'approved' | 'rejected'
  name: string
  url: string
}

export interface AdminStickerItem {
  id: number
  name: string
  fileName: string
  url: string
  /** Key of the official pack holding the sticker. */
  pack: string
  sortOrder: number
  isEnabled: boolean
  createdBy: number
  aliases: string[]
}

export interface AdminStickerSaveRequest {
  id?: number
  name: string
  fileName?: string
  /** Moves the sticker to another official pack; omitted keeps it. */
  pack?: string
  sortOrder?: number
  isEnabled?: boolean
}

export interface AdminStickerImportIssue {
  /** Complete ZIP entry path, single image filename, or `...` for the entry cap. */
  name: string
  reason: string
}

export interface AdminStickerImportResult {
  imported: number
  skipped: number
  failed: AdminStickerImportIssue[]
}

/** An official pack members pick from (GET /api/forum/sticker-packs). */
export interface StickerOfficialPack {
  key: string
  name: string
  coverUrl: string
  stickerCount: number
}

export interface AdminStickerPack {
  key: string
  name: string
  sortOrder: number
  coverStickerId: number
  coverUrl: string
  stickerCount: number
  enabledCount: number
}

export interface AdminStickerPackSaveRequest {
  /** Empty creates a pack with a generated key. */
  key?: string
  name: string
  sortOrder: number
  coverStickerId?: number
}

export type AdminStickerBatchAction = 'enable' | 'disable' | 'move' | 'delete'

export interface AdminStickerBatchRequest {
  ids: number[]
  action: AdminStickerBatchAction
  pack?: string
}

export interface MyStickerSaveRequest {
  stickerName?: string
  fileName?: string
  displayName?: string
}

export interface StickerNamesRequest {
  names: string[]
}

export interface StickerMarketTopic {
  id: string
  label: string
}

export interface StickerMarketPack {
  id: number
  name: string
  topic: string
  stickerCount: number
  coverUrl: string
}

export interface StickerMarketPage {
  topics: StickerMarketTopic[]
  packs: StickerMarketPack[]
  total: number
  page: number
  pageSize: number
}

export interface StickerMarketPackDetail {
  pack: StickerMarketPack
  stickers: StickerItem[]
}

export interface StickerPackRequest {
  packId: number
}

export type AdminStickerMarketStatus = 'new' | 'queued' | 'importing' | 'ready' | 'partial' | 'failed'

export type AdminStickerMarketSource = 'fluxdo' | 'upload'

export interface AdminStickerMarketPack {
  id: number
  /** `upload` packs are assembled from administrator files. */
  source: AdminStickerMarketSource
  upstreamId: string
  name: string
  topic: string
  upstreamCount: number
  archived: boolean
  missing: boolean
  status: AdminStickerMarketStatus
  published: boolean
  sortOrder: number
  itemCount: number
  failedCount: number
  importTotal: number
  lastError: string
  coverUrl: string
  iconUrl: string
  iconText: string
  syncedAt: string
  importedAt: string | null
}

export interface AdminStickerMarketPage {
  enabled: boolean
  source: string
  lastSyncedAt: string | null
  topics: StickerMarketTopic[]
  packs: AdminStickerMarketPack[]
  total: number
  page: number
  pageSize: number
}

export interface AdminStickerMarketSyncResult {
  topics: number
  packs: number
  missing: number
}

export interface AdminStickerMarketItem {
  upstreamId: string
  stickerId: number
  name: string
  /** Previous names that sent content still resolves, oldest first (issue #1428). */
  aliases: string[]
  url: string
  isEnabled: boolean
  error: string
}

export interface AdminStickerMarketPackDetail {
  pack: AdminStickerMarketPack
  items: AdminStickerMarketItem[]
}

export interface AdminStickerMarketPackSaveRequest {
  id: number
  published?: boolean
  sortOrder?: number
  /** Uploaded packs only. */
  name?: string
  topic?: string
  topicLabel?: string
}

export interface AdminStickerMarketPackCreateRequest {
  name: string
  topic?: string
  topicLabel?: string
}

export interface AdminStickerMarketUploadResult {
  result: AdminStickerImportResult
  pack: AdminStickerMarketPack
}

export interface AdminStickerMarketItemSaveRequest {
  stickerId: number
  /** Renames the sticker when set; the previous name keeps resolving. */
  name?: string
  isEnabled: boolean
}

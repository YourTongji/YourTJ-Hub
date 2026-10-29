export interface StickerItem {
  id?: number
  displayName?: string
  pack?: string
  isOfficial?: boolean
  isEnabled?: boolean
  name: string
  url: string
}

export interface AdminStickerItem {
  id: number
  name: string
  fileName: string
  url: string
  sortOrder: number
  isEnabled: boolean
  createdBy: number
  aliases: string[]
}

export interface AdminStickerSaveRequest {
  id?: number
  name: string
  fileName?: string
  sortOrder?: number
  isEnabled?: boolean
}

export interface AdminStickerImportIssue {
  name: string
  reason: string
}

export interface AdminStickerImportResult {
  imported: number
  skipped: number
  failed: AdminStickerImportIssue[]
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

export interface AdminStickerMarketPack {
  id: number
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
}

export interface AdminStickerMarketItemSaveRequest {
  stickerId: number
  isEnabled: boolean
}

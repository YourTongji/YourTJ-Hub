export interface StickerItem {
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

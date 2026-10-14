export interface DecorationStyle {
  code: string
  name: string
  kind: 'avatarFrame' | 'nicknameStyle'
  lightColors: string[]
  darkColors: string[]
  isEnabled: boolean
  sortOrder: number
  assetUrl?: string
  assetScale?: number
}
export interface UserDecoration {
  avatarFrame?: DecorationStyle | null
  nicknameStyle?: DecorationStyle | null
}
export interface DecorationState {
  audits?: Array<{ id: number; requestKey: string; userId: number; code: string; action: string; source: string; actorId: number; reason: string; createdAt: string }>
  catalog: DecorationStyle[]
  ownedCodes: string[]
  equipped: { avatarFrame: string; nicknameStyle: string }
}

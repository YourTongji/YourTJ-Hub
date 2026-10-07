import { readApiResponse } from './api'
import { i18n } from './i18n'

export interface Persona {
  kind: 'persona'
  publicUid: string
  name: string
  avatarUrl: string
  profileUrl: string
}

export interface NameBatch {
  id: string
  day: string
  words: string[]
  expiresAt: string
  createdAt: string
}

export interface IdentityState {
  persona: Persona | null
  nameSelectedAt: string | null
  nameChangeAvailableAt: string | null
  disabled: boolean
  governanceDisabled: boolean
  day: string
  remaining: number
  resetsAt: string
  batches: NameBatch[]
  lexiconVersion: string
}

async function request<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(`/api/forum/anonymous/${path}`, {
    method: body === undefined ? 'GET' : 'POST',
    credentials: 'same-origin',
    cache: 'no-store',
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  })
  return readApiResponse<T>(response, String(i18n.global.t('anonymous.retry')))
}

export const getIdentityState = () => request<IdentityState>('state')
export const generateNames = (day: string, requestKey: string) =>
  request<NameBatch>('batches', { day, requestKey })
export const confirmName = (batchId: string, index: number) =>
  request<Persona>('confirm', { batchId, index })
export const disableIdentity = (disabled: boolean) => request<boolean>('disable', { disabled })

export interface PublicAuthor {
  id: number
  publicUid?: string
  profileUrl?: string
}

export function authorURL(author: PublicAuthor): string {
  return author.publicUid ? `/a/${author.publicUid}` : author.id > 0 ? `/u/${author.id}` : ''
}

export function authorKey(author: PublicAuthor): string {
  return author.publicUid ? `persona:${author.publicUid}` : `member:${author.id}`
}

export const governIdentity = (postId: number, disabled: boolean, reason: string) =>
  request<boolean>('govern', { postId, disabled, reason })
export const revealIdentity = (publicUid: string, reason: string) =>
  request<{ publicUid: string; userId: number; username: string }>('reveal', { publicUid, reason })

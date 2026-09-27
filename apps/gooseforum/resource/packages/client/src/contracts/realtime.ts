// Hand-maintained mirror of the /api/forum/events SSE data contract. Frames
// invalidate REST snapshots; they contain no message body or replay cursor.
export interface ForumRealtimeHello {
  version: number
  heartbeatSeconds: number
  resync: boolean
  capabilities: { visibleRead: boolean }
}

export interface ForumRealtimeChatChanged {
  convId: number
  change: string
}

export interface ForumRealtimeNotificationsChanged {
  change: string
}

export type ForumRealtimeEvent =
  | { event: 'hello'; data: ForumRealtimeHello }
  | { event: 'chat.changed'; data: ForumRealtimeChatChanged }
  | { event: 'notifications.changed'; data: ForumRealtimeNotificationsChanged }
  | { event: 'unread.changed'; data: Record<string, never> }
  | { event: 'session.invalidated'; data: Record<string, never> }

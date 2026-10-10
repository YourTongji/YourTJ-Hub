// For You recommendation reasons; unknown values render nothing.
const reasonKeys: Record<string, string> = {
  following: 'feed.following',
  category: 'feed.category',
  newreply: 'feed.newreply',
  recent: 'feed.recent',
}

export function feedReasonKey(reason?: string | null): string {
  return (reason && reasonKeys[reason]) || ''
}

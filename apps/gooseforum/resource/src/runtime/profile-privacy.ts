import type { ProfileAudience, ProfilePrivacySection, UserPrivacySettings } from './api'

// Mirrors users.PrivacyAudiences on the server (issue #1124). The server
// rejects anything else, so this list only decides which choices to offer.
export const PROFILE_PRIVACY_SECTIONS: ReadonlyArray<{ key: ProfilePrivacySection, audiences: ProfileAudience[] }> = [
  { key: 'profileTopics', audiences: ['everyone', 'members', 'followers', 'mutual', 'self'] },
  { key: 'profileActivity', audiences: ['everyone', 'members', 'followers', 'mutual', 'self'] },
  { key: 'profileLikes', audiences: ['everyone', 'followers', 'self'] },
  { key: 'followingList', audiences: ['everyone', 'followers', 'mutual', 'self'] },
  { key: 'followersList', audiences: ['everyone', 'followers', 'mutual', 'self'] },
]

const AUDIENCE_RANK: Record<ProfileAudience, number> = { everyone: 0, members: 1, followers: 2, mutual: 3, self: 4 }

/** True when the new audience lets more people see the section. */
export function widensAudience(from: ProfileAudience, to: ProfileAudience): boolean {
  return AUDIENCE_RANK[to] < AUDIENCE_RANK[from]
}

// The pre-#1124 Web settings stored these switches only in localStorage; the
// server never read them. They are offered for an explicit one-time migration.
export const LEGACY_PRIVACY_KEY = 'goose-privacy-settings'

export function readLegacyPrivacy(storage: Pick<Storage, 'getItem'> = window.localStorage): Partial<UserPrivacySettings> | null {
  let raw: unknown
  try {
    raw = JSON.parse(storage.getItem(LEGACY_PRIVACY_KEY) || 'null')
  } catch {
    return null
  }
  if (!raw || typeof raw !== 'object') return null
  const legacy = raw as { showTopics?: unknown, showFollowing?: unknown }
  const update: Partial<UserPrivacySettings> = {}
  if (legacy.showTopics === false) {
    update.profileTopics = 'self'
    update.profileActivity = 'self'
    update.profileLikes = 'self'
  }
  if (legacy.showFollowing === false) {
    update.followingList = 'self'
    update.followersList = 'self'
  }
  return Object.keys(update).length ? update : null
}

export function clearLegacyPrivacy(storage: Pick<Storage, 'removeItem'> = window.localStorage) {
  try {
    storage.removeItem(LEGACY_PRIVACY_KEY)
  } catch {
    // Storage can be unavailable in private modes; nothing was applied from it.
  }
}

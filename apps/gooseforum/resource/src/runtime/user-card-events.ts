import type { UserDecoration, UserBadgePayload } from '@gooseforum/client'
import { useAvatarClickBehavior } from './avatar-click-behavior'

export interface UserCardTarget {
 publicUid?: string
  id: number
  username: string
  decoration?: UserDecoration
  avatarUrl: string
  wornBadge?: UserBadgePayload | null
  isFollowing?: boolean
}

export interface UserCardShowDetail {
  user: UserCardTarget
  target: HTMLElement
}

const { avatarClickBehavior } = useAvatarClickBehavior()

/**
 * 全站头像与作者入口的唯一分发点：默认弹出用户卡片（`UserCard.vue`），
 * 「直接打开用户主页」偏好（issue #1134）下放行调用点锚点自身的链接。
 * 调用点必须是带 `href` 的 `<a>`；修饰键点击与匿名公开身份始终交还浏览器默认行为。
 */
export function showUserCard(user: UserCardTarget, event: MouseEvent) {
  if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return

  if (user.publicUid) { return }
  // 直达模式不 preventDefault：锚点 href 与全站点击拦截器完成 SPA 导航（与 publicUid 分支同一策略）。
  if (avatarClickBehavior.value === 'profile') return
  event.preventDefault()
  event.stopPropagation()

  const target = event.currentTarget
  if (!(target instanceof HTMLElement)) return

  window.dispatchEvent(
    new CustomEvent<UserCardShowDetail>('goose:user-card-show', {
      detail: { user, target },
    }),
  )
}

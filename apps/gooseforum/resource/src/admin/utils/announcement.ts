import { toBool } from './toBool'
import type { AnnouncementConfig, AnnouncementItemConfig } from '../types'

export function normalizeAnnouncement(settings: Partial<AnnouncementConfig> = {}): AnnouncementConfig {
  const items = settings.items ?? []
  // 旧版 content-only 单则配置迁移为一条常规条目（issue #465）：直接换常规 id
  // 并始终走 items 模式，保证标题（items[].title）从首次保存起即可持久化。
  // 不再依赖「legacy 单则回退 content-only」路径——content-only 结构无法表达标题，
  // 曾导致公告标题保存后被静默丢弃。
  if (items.length === 0 && settings.content) {
    return {
      enabled: toBool(settings.enabled, false),
      content: settings.content,
      items: [{ id: `ann-${Date.now().toString(36)}`, title: '', content: settings.content, enabled: true }],
    }
  }
  return {
    enabled: toBool(settings.enabled, false),
    content: settings.content ?? '',
    items,
  }
}

export function serializeAnnouncement(
  form: Pick<AnnouncementConfig, 'enabled' | 'content'> & { items?: AnnouncementItemConfig[] },
): AnnouncementConfig {
  const items = (form.items ?? []).filter((item) => item.content.trim())
  // 单则公告（含迁移自旧版的条目）时同步 content：服务端 items 非空时优先展示，
  // content 仅作单则回退（GetHtmlContent），同步可保持单则语义一致（issue #465）
  return {
    enabled: form.enabled,
    content: items.length === 1 ? items[0].content : form.content,
    items,
  }
}

// Single source for admin navigation: the sidebar and the command search both read it.
import { adminText } from '@/admin/runtime/i18n-text'
import {
  Calendar,
  Activity,
  BookOpen,
  Award,
  Bot,
  Clock,
  Database,
  FileText,
  Files,
  HardDrive,
  Heart,
  Link,
  ListChecks,
  Mail,
  Megaphone,
  MessageSquare,
  Monitor,
  PanelsTopLeft,
  PanelLeft,
  RefreshCw,
  ScanEye,
  ScrollText,
  Search,
  Sparkles,
  Sticker,
  Cpu,
  Shield,
  ShieldCheck,
  Tags,
  UserCog,
  Webhook,
} from '@lucide/vue'
import type { LucideIcon } from '@lucide/vue'
import { AdminPermission, canVisitAdminPath } from '@/admin/runtime/access'
import { i18n } from '@/runtime/i18n'

export interface NavItem {
  title: string
  url: string
  icon: LucideIcon
  permission?: AdminPermission
  active?: boolean
  external?: boolean
  items?: NavItem[]
}

export interface NavGroup {
  title: string
  items: NavItem[]
}

/** Groups visible to the current admin; call inside a computed so locale changes re-run it. */
export function adminNavGroups(): NavGroup[] {
  const t = (key: string) => i18n.global.t(key)
  return [
  {
    title: 'YourTJHub',
    items: [
      { title: adminText('k004c'), url: '/admin', icon: Monitor, permission: AdminPermission.Admin },
      { title: t('feedAdmin.title'), url: '/admin/feed-statistics', icon: Activity, permission: AdminPermission.Admin },
      { title: adminText('k006i'), url: '/admin/users', icon: UserCog, permission: AdminPermission.UserManager },
      { title: t('anonymousAdmin.title'), url: '/admin/anonymous-identities', icon: Shield, permission: AdminPermission.RevealAnonymousIdentity },
      { title: adminText('k00k7'), url: '/admin/agents', icon: Bot, permission: AdminPermission.Admin },
      { title: adminText('k00wh3'), url: '/admin/agent-comment-policy', icon: MessageSquare, permission: AdminPermission.Admin },
      { title: adminText('k007f'), url: '/admin/roles', icon: ShieldCheck, permission: AdminPermission.RoleManager },
      { title: adminText('k005l'), url: '/admin/categories', icon: Tags, permission: AdminPermission.TopicsManager },
      { title: adminText('k005u'), url: '/admin/posts', icon: FileText, permission: AdminPermission.TopicsManager },
      { title: adminText('k002j'), url: '/admin/links', icon: Link, permission: AdminPermission.PageManager },
      { title: adminText('k00n4'), url: '/admin/wiki', icon: BookOpen, permission: AdminPermission.PageManager },
      { title: adminText('k004o'), url: '/admin/sponsors', icon: Heart, permission: AdminPermission.PageManager },
      { title: adminText('k0058'), url: '/admin/badges', icon: Award, permission: AdminPermission.SiteManager },
      { title: adminText('k00vg8'), url: '/admin/stickers', icon: Sticker, permission: AdminPermission.SiteManager },
      { title: adminText('k00f6'), url: '/admin/files/resources', icon: Files, permission: AdminPermission.SiteManager },
      { title: adminText('k007c'), url: '/admin/opt-records', icon: ListChecks, permission: AdminPermission.Admin },
      { title: adminText('k00ge'), url: '/admin/review-queue', icon: ListChecks, permission: AdminPermission.SiteManager },
      { title: t('searchAdmin.title'), url: '/admin/search', icon: Search, permission: AdminPermission.SiteManager },
      { title: adminText('k00h0'), url: '/admin/data', icon: Database, permission: AdminPermission.SiteManager },
    ],
  },
  {
    title: adminText('k007t'),
    items: [
      { title: adminText('k007u'), url: '/admin/settings/site-info', icon: PanelsTopLeft, permission: AdminPermission.SiteManager },
      { title: adminText('k00d9'), url: '/admin/settings/site-chrome', icon: PanelLeft, permission: AdminPermission.SiteManager },
      { title: adminText('k007v'), url: '/admin/settings/mail', icon: Mail, permission: AdminPermission.SiteManager },
      { title: adminText('k0005'), url: '/admin/settings/security', icon: ShieldCheck, permission: AdminPermission.SiteManager },
      { title: t('aiModerationAdmin.title'), url: '/admin/settings/ai-moderation', icon: ScanEye, permission: AdminPermission.SiteManager },
      { title: adminText('k007w'), url: '/admin/settings/posting', icon: FileText, permission: AdminPermission.SiteManager },
      { title: adminText('k00ig'), url: '/admin/settings/rate-limit', icon: Shield, permission: AdminPermission.SiteManager },
      { title: adminText('k00mj'), url: '/admin/settings/mcp', icon: Cpu, permission: AdminPermission.SiteManager },
      { title: adminText('k00p0'), url: '/admin/settings/ai-summary', icon: Sparkles, permission: AdminPermission.SiteManager },
      { title: adminText('k0009'), url: '/admin/settings/announcement', icon: Megaphone, permission: AdminPermission.PageManager },
      { title: adminText('k00cj'), url: '/admin/settings/http-notify', icon: Webhook, permission: AdminPermission.SiteManager },
      { title: adminText('k00fn'), url: '/admin/settings/storage', icon: HardDrive, permission: AdminPermission.SiteManager },
      { title: adminText('k00gp'), url: '/admin/settings/terms', icon: ScrollText, permission: AdminPermission.SiteManager },
      { title: adminText('k00gu'), url: '/admin/settings/privacy', icon: ShieldCheck, permission: AdminPermission.SiteManager },
      { title: adminText('k00t4'), url: '/admin/settings/onesystem', icon: RefreshCw, permission: AdminPermission.SiteManager },
      { title: t('campus.adminTitle'), url: '/admin/settings/campus-calendar', icon: Calendar, permission: AdminPermission.SiteManager },
      { title: adminText('k00u1'), url: '/admin/settings/schedule', icon: Clock, permission: AdminPermission.SiteManager },
    ],
  },
  ].map(group => ({
    ...group,
    items: group.items.filter(item => item.permission === undefined || canVisitAdminPath(item.url)),
  })).filter(group => group.items.length > 0)
}

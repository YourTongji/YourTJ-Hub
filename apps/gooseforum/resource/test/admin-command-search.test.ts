import { expect, test } from 'vitest'
import { commandEntries, searchCommands } from '../src/admin/runtime/command-search'
import type { NavGroup } from '../src/admin/runtime/navigation'

const icon = {} as NavGroup['items'][number]['icon']
const groups: NavGroup[] = [
  { title: '概览', items: [{ title: '站点统计', url: '/admin', icon }, { title: '推荐统计', url: '/admin/feed-statistics', icon }] },
  { title: '社区管理', items: [{ title: '用户管理', url: '/admin/users', icon }, { title: '帖子管理', url: '/admin/posts', icon }] },
  { title: '系统设置', items: [{ title: '邮件设置', url: '/admin/settings/mail', icon }, { title: '外部文档', url: 'https://example.com', icon, external: true }] },
]
const titles = (query: string) => searchCommands(commandEntries(groups), query).map((entry) => entry.item.title)

test('empty query lists every internal page in sidebar order', () => {
  expect(titles('  ')).toEqual(['站点统计', '推荐统计', '用户管理', '帖子管理', '邮件设置'])
})

test('title prefix ranks before substring and in-order characters', () => {
  expect(titles('统计')).toEqual(['站点统计', '推荐统计'])
  expect(titles('推荐')).toEqual(['推荐统计'])
  expect(titles('用管')).toEqual(['用户管理'])
  expect(titles('管理')).toEqual(['用户管理', '帖子管理'])
})

test('route words and group names match in any UI language', () => {
  expect(titles('Users')).toEqual(['用户管理'])
  expect(titles('mail')).toEqual(['邮件设置'])
  expect(titles('dashboard')).toEqual(['站点统计'])
  expect(titles('社区')).toEqual(['用户管理', '帖子管理'])
  expect(titles('zzz')).toEqual([])
})

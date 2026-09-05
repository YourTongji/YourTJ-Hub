import { describe, expect, test } from 'vitest'
import { normalizeAnnouncement, serializeAnnouncement } from '../src/admin/utils/announcement'

// issue #465 回归：后台公告页保存时标题被静默丢弃。
// 根因：normalizeAnnouncement 把旧版 content-only 配置迁移成 id='legacy' 的条目展示，
// serializeAnnouncement 在唯一 legacy 项时无条件序列化回 content-only 结构
// （items: []），而标题只存在于 items[].title，于是被静默丢弃。
// 修复：加载时 legacy 条目换常规 id 并始终走 items 模式，序列化不再回退 content-only。
// PR review 收紧：content 仅在「单则且启用」时同步条目正文，其余清空——
// 已删除或已停用的公告不得经服务端 GetHtmlContent 单则回退继续展示。
describe('announcement serialize round-trip (issue #465)', () => {
  test('legacy 单则公告编辑标题后保存，title 必须进 payload', () => {
    const form = normalizeAnnouncement({ enabled: true, content: '欢迎来到 YourTJHub' })
    expect(form.items).toHaveLength(1)
    expect(form.items![0].id).toMatch(/^ann-/)

    form.items![0].title = '维护通知'

    const payload = serializeAnnouncement(form)
    expect(payload.items).toHaveLength(1)
    expect(payload.items![0]).toMatchObject({ title: '维护通知', content: '欢迎来到 YourTJHub', enabled: true })
  })

  test('存量 legacy 项（未编辑）保存后同样退出 content-only 形态', () => {
    const form = normalizeAnnouncement({ enabled: true, content: '公告正文' })
    const payload = serializeAnnouncement(form)
    expect(payload.items).toHaveLength(1)
    expect(payload.items![0]).toMatchObject({ id: expect.not.stringContaining('legacy'), content: '公告正文' })
    expect(payload.items![0].title).toBe('')
  })

  test('单则且启用的条目：content 与条目正文同步（保持单则回退语义）', () => {
    const payload = serializeAnnouncement({
      enabled: true,
      items: [{ id: 'ann-x', title: '新标题', content: '编辑后的正文', enabled: true }],
    })
    expect(payload.content).toBe('编辑后的正文')
  })

  test('唯一条目停用时 content 清空（停用公告不得经 content 回退展示）', () => {
    const payload = serializeAnnouncement({
      enabled: true,
      items: [{ id: 'ann-x', title: '停用', content: '停用的正文', enabled: false }],
    })
    expect(payload.content).toBe('')
    expect(payload.items).toHaveLength(1)
  })

  test('清空全部条目后 content 同步清空（已删除公告不得经 content 回退展示）', () => {
    const payload = serializeAnnouncement({ enabled: true, items: [] })
    expect(payload).toEqual({ enabled: true, content: '', items: [] })
  })

  test('多则公告保存保持 items 原样，content 不透传表单值（置空）', () => {
    const payload = serializeAnnouncement({
      enabled: true,
      items: [
        { id: 'ann-a', title: '第一条', content: '内容一', enabled: true },
        { id: 'ann-b', title: '第二条', content: '内容二', enabled: false },
      ],
    })
    expect(payload).toEqual({
      enabled: true,
      content: '',
      items: [
        { id: 'ann-a', title: '第一条', content: '内容一', enabled: true },
        { id: 'ann-b', title: '第二条', content: '内容二', enabled: false },
      ],
    })
  })

  test('空正文项被过滤，与既有行为一致', () => {
    const payload = serializeAnnouncement({
      enabled: true,
      items: [
        { id: 'ann-a', title: '有内容', content: '正文', enabled: true },
        { id: 'ann-b', title: '空条目', content: '   ', enabled: true },
      ],
    })
    expect(payload.items).toHaveLength(1)
    expect(payload.items![0].id).toBe('ann-a')
  })

  test('全部清空后保存为空配置', () => {
    const payload = serializeAnnouncement({ enabled: false, items: [] })
    expect(payload).toEqual({ enabled: false, content: '', items: [] })
  })

  test('normalizeAnnouncement 无公告数据时给空表单', () => {
    expect(normalizeAnnouncement()).toEqual({ enabled: false, content: '', items: [] })
  })

  test('normalizeAnnouncement 迁移旧版 content-only 配置为常规可编辑条目', () => {
    const form = normalizeAnnouncement({ enabled: true, content: '旧版单则公告' })
    expect(form.items).toHaveLength(1)
    expect(form.items![0]).toMatchObject({ title: '', content: '旧版单则公告', enabled: true })
    expect(form.items![0].id).toMatch(/^ann-/)
  })
})

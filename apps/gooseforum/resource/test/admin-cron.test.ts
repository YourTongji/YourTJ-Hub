import { describe, expect, it } from 'vitest'
import { isValidCron5Field } from '@/admin/cron'

describe('isValidCron5Field', () => {
  it('accepts the standard daily 02:30 spec', () => {
    expect(isValidCron5Field('30 2 * * *')).toBe(true)
  })

  it('accepts steps, ranges, lists and */step', () => {
    expect(isValidCron5Field('*/15 0 * * 1-5')).toBe(true)
    expect(isValidCron5Field('0 2-4/2 * * *')).toBe(true)
    expect(isValidCron5Field('0,15,30,45 * * * *')).toBe(true)
    expect(isValidCron5Field('0 3 * JAN,MAR *')).toBe(true)
    expect(isValidCron5Field('0 3 * * MON-FRI')).toBe(true)
    expect(isValidCron5Field('0 0 1 */2 *')).toBe(true)
  })

  it('accepts robfig @ descriptors', () => {
    expect(isValidCron5Field('@daily')).toBe(true)
    expect(isValidCron5Field('@hourly')).toBe(true)
    expect(isValidCron5Field('@every 30m')).toBe(true)
    expect(isValidCron5Field('@every 1h30m')).toBe(true)
    expect(isValidCron5Field('@every 500ms')).toBe(true)
    expect(isValidCron5Field('@every 1.5h')).toBe(true)
  })

  it.each(['0 0 0 * *', '0 0 * 0 *', '@every 1d', '*/1e1 * * * *', '*/2.5 * * * *'])('rejects server-invalid expression %s', (spec) => {
    expect(isValidCron5Field(spec)).toBe(false)
  })

  it('rejects malformed and out-of-range specs', () => {
    expect(isValidCron5Field('')).toBe(false)
    expect(isValidCron5Field('not-a-cron')).toBe(false)
    expect(isValidCron5Field('30 2 * *')).toBe(false) // 只有 4 段
    expect(isValidCron5Field('30 2 * * * *')).toBe(false) // 6 段（秒）
    expect(isValidCron5Field('61 * * * *')).toBe(false) // 分钟越界
    expect(isValidCron5Field('0 24 * * *')).toBe(false) // 小时越界
    expect(isValidCron5Field('0 0 32 * *')).toBe(false) // 日越界
    expect(isValidCron5Field('0 0 * 13 *')).toBe(false) // 月越界
    expect(isValidCron5Field('0 0 * * 7')).toBe(false) // 周 7 后端不收
    expect(isValidCron5Field('*/0 * * * *')).toBe(false) // step=0
    expect(isValidCron5Field('30-10 * * * *')).toBe(false) // start > end
  })
})

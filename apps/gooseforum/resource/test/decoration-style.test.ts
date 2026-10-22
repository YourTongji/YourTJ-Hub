import { describe, expect, it } from 'vitest'
import { NICKNAME_CONTRAST_MIN, decorationVariables, hasNicknameContrast, isSafeDecorationAssetUrl, nicknameContrastFloor } from '../src/site/utils/decoration-style'
import type { DecorationStyle } from '@gooseforum/client'
const style: DecorationStyle = { code: 'tea', name: 'Tea', kind: 'nicknameStyle', lightColors: ['#166534'], darkColors: ['#86efac'], isEnabled: true, sortOrder: 1 }
describe('decoration rendering boundary', () => {
  it('rejects disabled and arbitrary CSS colors while allowing one-color gradients', () => {
    expect(decorationVariables(style)['--decoration-light']).toBe('linear-gradient(120deg, #166534, #166534)')
    expect(decorationVariables({ ...style, isEnabled: false })).toEqual({})
    expect(decorationVariables({ ...style, lightColors: ['url(https://invalid)'] })).toEqual({})
    expect(decorationVariables({ ...style, darkColors: [] })).toEqual({})
  })
  it('keeps names readable against custom backgrounds at the 3:1 endpoint floor', () => {
    expect(NICKNAME_CONTRAST_MIN).toBe(3)
    expect(hasNicknameContrast(style.lightColors, [255, 255, 255])).toBe(true)
    expect(hasNicknameContrast(style.darkColors, [17, 24, 39])).toBe(true)
    expect(hasNicknameContrast(style.lightColors, [22, 101, 52])).toBe(false)
    expect(hasNicknameContrast(['#000000', '#ffffff'], [255, 255, 255])).toBe(false)
    expect(hasNicknameContrast(['red'], [255, 255, 255])).toBe(false)
  })
  it('accepts green-to-blue stops between 3:1 and the old 4.5:1 floor', () => {
    expect(hasNicknameContrast(['#3B82F6', '#059669'], [255, 255, 255])).toBe(true)
    expect(hasNicknameContrast(['#3B82F6', '#059669'], [243, 244, 246])).toBe(true)
  })
  it('rejects palettes whose stops fall below 3:1', () => {
    expect(hasNicknameContrast(['#9CA3AF', '#A3A3A3'], [255, 255, 255])).toBe(false)
    expect(hasNicknameContrast(['#3B82F6', '#9CA3AF'], [255, 255, 255])).toBe(false)
    expect(hasNicknameContrast(['#4B5563', '#374151'], [31, 41, 55])).toBe(false)
  })
  it('measures the lowest stop/background contrast so the admin can show the real ratio', () => {
    const light = [[255, 255, 255], [243, 244, 246]]
    expect(nicknameContrastFloor(['#3B82F6', '#059669'], light)).toBeCloseTo(3.34, 2)
    expect(nicknameContrastFloor(['#9CA3AF', '#A3A3A3'], light)).toBeCloseTo(2.29, 2)
    expect(nicknameContrastFloor(['#9CA3AF'], light)).toBeCloseTo(2.31, 2)
    expect(hasNicknameContrast(['#9CA3AF', '#A3A3A3'], [255, 255, 255])).toBe(false)
    expect(nicknameContrastFloor([], light)).toBe(0)
    expect(nicknameContrastFloor(['red'], light)).toBe(0)
    expect(nicknameContrastFloor(['#FFFFFF'], [])).toBe(0)
  })
})

it('accepts all artwork formats without widening trusted decoration paths', () => {
  for (const ext of ['png', 'webp', 'gif', 'svg']) expect(isSafeDecorationAssetUrl(`/static/decorations/frame.${ext}`)).toBe(true)
  for (const url of ['https://example.com/frame.gif', '/static/decorations/../frame.svg', '/static/decorations/frame.gif?x=1', '/static/decorations/frame.html']) expect(isSafeDecorationAssetUrl(url)).toBe(false)
})

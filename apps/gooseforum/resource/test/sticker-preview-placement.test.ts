import { describe, expect, it } from 'vitest'
import { placeStickerPreview, type PlacementBox } from '../src/site/utils/sticker-preview-placement'

const size = { width: 140, height: 164 }
const box = (left: number, top: number, width: number, height: number): PlacementBox => ({ left, top, right: left + width, bottom: top + height })
const overlaps = (a: PlacementBox, b: PlacementBox) => a.left < b.right && b.left < a.right && a.top < b.bottom && b.top < a.bottom
const rectOf = (place: { left: number, top: number }) => box(place.left, place.top, size.width, size.height)

describe('placeStickerPreview', () => {
  const desktop = { width: 1280, height: 800 }
  const panel = box(300, 360, 272, 288)
  const cell = box(360, 420, 44, 44)

  it('sits beside the panel on the trailing side, level with the cell', () => {
    const place = placeStickerPreview(cell, panel, size, desktop)
    expect(place.side).toBe('end')
    expect(place.left).toBe(panel.right + 8)
    expect(place.top + size.height / 2).toBe(442)
    expect(overlaps(rectOf(place), panel)).toBe(false)
    expect(place.origin).toBe('0 82px')
  })

  it('mirrors to the left in right-to-left layouts', () => {
    const place = placeStickerPreview(cell, panel, size, desktop, true)
    expect(place.side).toBe('end')
    expect(place.left + size.width).toBe(panel.left - 8)
  })

  it('takes the leading side when the trailing one runs out of room', () => {
    const place = placeStickerPreview(box(1000, 420, 44, 44), box(990, 360, 272, 288), size, desktop)
    expect(place.side).toBe('start')
    expect(place.left + size.width).toBe(990 - 8)
  })

  it('stacks above the panel on a narrow screen and never covers the grid', () => {
    const phone = { width: 390, height: 844 }
    const narrowPanel = box(33, 337, 272, 288)
    const place = placeStickerPreview(box(90, 400, 44, 44), narrowPanel, size, phone)
    expect(place.side).toBe('above')
    expect(place.top + size.height).toBe(337 - 8)
    expect(overlaps(rectOf(place), narrowPanel)).toBe(false)
    expect(place.left).toBeGreaterThanOrEqual(8)
  })

  it('drops below the panel when the panel hugs the top edge', () => {
    const phone = { width: 390, height: 844 }
    const place = placeStickerPreview(box(90, 120, 44, 44), box(33, 60, 272, 288), size, phone)
    expect(place.side).toBe('below')
    expect(place.top).toBe(60 + 288 + 8)
  })

  it('falls back above the cell, inside the viewport, when nothing outside fits', () => {
    const tiny = { width: 320, height: 420 }
    const place = placeStickerPreview(box(30, 150, 44, 44), box(16, 60, 288, 300), size, tiny)
    expect(place.side).toBe('cell')
    const rect = rectOf(place)
    expect(rect.left).toBeGreaterThanOrEqual(8)
    expect(rect.right).toBeLessThanOrEqual(tiny.width - 8)
    expect(rect.top).toBeGreaterThanOrEqual(8)
  })

  it('keeps a side placement inside the viewport near the bottom edge', () => {
    const place = placeStickerPreview(box(360, 760, 44, 30), box(300, 500, 272, 288), size, desktop)
    expect(place.top + size.height).toBeLessThanOrEqual(desktop.height - 8)
  })
})

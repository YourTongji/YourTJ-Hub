import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, test } from 'vitest'
import { IMAGE_MIN_RENDERED_WIDTH, IMAGE_WIDTH_PRESETS, imageWidthOf, snapImageWidth, withImageWidth, type ImageWidth } from '../src/runtime/image-width'

const fixture = JSON.parse(
  readFileSync(resolve(process.cwd(), '../../../packages/api-contract/fixtures/markdown-image-width.json'), 'utf8'),
) as {
  presets: number[]
  minRenderedWidth: number
  parse: { src: string; width: number | null }[]
  set: { src: string; width: number | null; result: string }[]
  snap: { ratio: number; width: number }[]
}

describe('relative image width (MADR 0131)', () => {
  test('presets and minimum match the shared fixture', () => {
    expect([...IMAGE_WIDTH_PRESETS]).toEqual(fixture.presets)
    expect(IMAGE_MIN_RENDERED_WIDTH).toBe(fixture.minRenderedWidth)
  })
  for (const item of fixture.parse) test(`parse ${item.src}`, () => expect(imageWidthOf(item.src)).toBe(item.width))
  for (const item of fixture.set) {
    test(`set ${item.src} → ${item.width}`, () => expect(withImageWidth(item.src, item.width as ImageWidth | null)).toBe(item.result))
  }
  for (const item of fixture.snap) test(`snap ${item.ratio}`, () => expect(snapImageWidth(item.ratio)).toBe(item.width))
})

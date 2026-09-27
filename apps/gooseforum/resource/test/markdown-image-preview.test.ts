// @vitest-environment happy-dom
import { afterEach, describe, expect, it } from 'vitest'
import { getMarkdownImagePreview } from '@/runtime/markdown-image-preview'

afterEach(() => { document.body.innerHTML = '' })

function images() {
  document.body.innerHTML = `
    <div class="gf-prose-post">
      <img id="first" src="/photo-one.png" alt="Photo one">
      <img id="official" src="/official.png" alt="sticker:smile" data-gf-sticker="smile">
      <img id="personal" src="/personal.png" alt="sticker:u_abc" data-gf-sticker="u_abc">
      <a href="/photo-two.png"><img id="second" src="/photo-two.png" alt="Photo two"></a>
    </div>`
  return (id: string) => document.getElementById(id)!
}

describe('Markdown image preview boundaries', () => {
  it('keeps ordinary sticker-prefixed alt images in the gallery even at a sticker URL', () => {
    document.body.innerHTML = `<div class="gf-prose-post">
      <img src="/same.png" alt="sticker:smile" data-gf-sticker="smile">
      <img id="photo" src="/same.png" alt="sticker:smile">
      <img src="/other.png" alt="sticker:unknown">
    </div>`
    const preview = getMarkdownImagePreview(document.getElementById('photo'))!
    expect(preview).not.toBeNull()
    expect(preview.images.map(({ alt }) => alt)).toEqual(['sticker:smile', 'sticker:unknown'])
    expect(preview.index).toBe(0)
  })
  for (const kind of ['official', 'personal']) {
    it(`keeps ${kind} stickers out of the lightbox`, () => {
      expect(getMarkdownImagePreview(images()(kind))).toBeNull()
    })
  }

  it('collects ordinary photos only and indexes the clicked photo after filtering', () => {
    const preview = getMarkdownImagePreview(images()('second'))!
    expect(preview.images.map(({ alt }) => alt)).toEqual(['Photo one', 'Photo two'])
    expect(preview.index).toBe(1)
    expect(preview.images[1].src).toContain('/photo-two.png')
  })

  it('preserves a linked photo destination instead of opening a gallery', () => {
    document.body.innerHTML = '<div class="gf-prose-post"><a href="https://example.test/article"><img src="/photo.png"></a></div>'
    expect(getMarkdownImagePreview(document.querySelector('img'))).toBeNull()
  })

  it('ignores non-image targets, outside images and missing sources', () => {
    document.body.innerHTML = '<img id="outside" src="/avatar.png"><div class="gf-prose-post"><img id="empty"></div>'
    for (const target of [null, document, document.body, document.getElementById('outside'), document.getElementById('empty')]) {
      expect(getMarkdownImagePreview(target)).toBeNull()
    }
  })

  it('uses currentSrc for a responsive photo without changing its readable label', () => {
    const first = images()('first')
    Object.defineProperty(first, 'currentSrc', { value: 'https://cdn.example.test/photo-large.png' })
    const preview = getMarkdownImagePreview(first)!
    expect(preview.images[0]).toEqual({ src: 'https://cdn.example.test/photo-large.png', alt: 'Photo one' })
    expect(preview.index).toBe(0)
  })
})

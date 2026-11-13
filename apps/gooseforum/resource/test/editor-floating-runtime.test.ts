// @vitest-environment happy-dom
import { describe, expect, test } from 'vitest'
import { computeFloatingPosition, rectVisibleIn } from '@/runtime/floating-position'
import { editableImage, editorImageSource, insideRenderedBlock, removeEditorImage, setEditorImageAlt, setEditorImageSource } from '@/runtime/editor-image-dom'
import { clampZoomView, IDENTITY_ZOOM, zoomViewAt } from '@/runtime/image-gestures'

const boundary = { top: 100, left: 0, right: 400, bottom: 700 }
const size = { width: 200, height: 40 }

describe('computeFloatingPosition', () => {
  test('centres above the anchor', () => {
    const anchor = { top: 300, bottom: 320, left: 150, right: 250 }
    expect(computeFloatingPosition(size, anchor, boundary)).toEqual({ left: 100, top: 252, side: 'top' })
  })

  test('flips below when the top edge would clip', () => {
    const anchor = { top: 120, bottom: 140, left: 150, right: 250 }
    expect(computeFloatingPosition(size, anchor, boundary)).toMatchObject({ top: 148, side: 'bottom' })
  })

  test('clamps horizontally inside the boundary margin', () => {
    expect(computeFloatingPosition(size, { top: 300, bottom: 320, left: 0, right: 10 }, boundary).left).toBe(8)
    expect(computeFloatingPosition(size, { top: 300, bottom: 320, left: 390, right: 400 }, boundary).left).toBe(192)
  })

  test('an anchor taller than the boundary keeps the bar inside it', () => {
    const anchor = { top: 50, bottom: 900, left: 0, right: 400 }
    const { top } = computeFloatingPosition(size, anchor, boundary)
    expect(top).toBeGreaterThanOrEqual(108)
    expect(top + size.height).toBeLessThanOrEqual(692)
  })

  test('an anchor scrolled out of the boundary is not visible', () => {
    expect(rectVisibleIn({ top: 40, bottom: 90, left: 0, right: 10 }, boundary)).toBe(false)
    expect(rectVisibleIn({ top: 90, bottom: 110, left: 0, right: 10 }, boundary)).toBe(true)
  })
})

describe('editor image DOM', () => {
  function wysiwyg(html: string) {
    const root = document.createElement('pre')
    root.className = 'vditor-reset'
    root.innerHTML = html
    document.body.append(root)
    return root
  }

  test('only body images are editable, even inside the <pre> editing root', () => {
    const root = wysiwyg('<p><img src="/a.png"><img class="twemoji" src="/e.svg"><img data-gf-sticker src="/s.png"></p><div class="vditor-wysiwyg__block"><pre><img src="/chart.png"></pre></div>')
    const [body, emoji, sticker, chart] = root.querySelectorAll('img')
    expect(editableImage(body, root)).toBe(body)
    expect(editableImage(emoji, root)).toBeNull()
    expect(editableImage(sticker, root)).toBeNull()
    expect(editableImage(chart, root)).toBeNull()
    root.remove()
  })

  test('selections inside code blocks are rendered blocks, inline code is not', () => {
    const root = wysiwyg('<p>a <code>b</code></p><div class="vditor-wysiwyg__block"><pre><code>c</code></pre></div>')
    const [inline, block] = root.querySelectorAll('code')
    expect(insideRenderedBlock(inline.firstChild, root)).toBe(false)
    expect(insideRenderedBlock(block.firstChild, root)).toBe(true)
    root.remove()
  })

  test('removing the only image of a paragraph removes the empty paragraph', () => {
    const root = wysiwyg('<p>keep</p><p><img src="/a.png"></p>')
    removeEditorImage(root.querySelector('img')!, 'wysiwyg')
    expect(root.innerHTML).toBe('<p>keep</p>')
    root.remove()
  })

  test('ir markers carry the source and the alt text', () => {
    const root = document.createElement('div')
    root.innerHTML = '<p><span class="vditor-ir__node" data-type="img"><span class="vditor-ir__marker">!</span><span class="vditor-ir__marker vditor-ir__marker--bracket">[</span><span class="vditor-ir__marker vditor-ir__marker--bracket">]</span><span class="vditor-ir__marker vditor-ir__marker--paren">(</span><span class="vditor-ir__marker vditor-ir__marker--link">/a.png</span><span class="vditor-ir__marker vditor-ir__marker--paren">)</span><img src="http://host/a.png"></span></p>'
    const img = root.querySelector('img')!
    expect(editorImageSource(img, 'ir')).toBe('/a.png')
    setEditorImageSource(img, '/a.png#w=50', 'ir')
    setEditorImageAlt(img, '说明', 'ir')
    expect(root.querySelector('.vditor-ir__node')!.textContent).toBe('![说明](/a.png#w=50)')
    setEditorImageAlt(img, '新说明', 'ir')
    expect(root.querySelector('.vditor-ir__node')!.textContent).toBe('![新说明](/a.png#w=50)')
    // 标记是 Markdown 原文：`\`、`[`、`]` 转义，img.alt 保留原始文字
    setEditorImageAlt(img, 'a]b[c\\d', 'ir')
    expect(root.querySelector('.vditor-ir__node')!.textContent).toBe('![a\\]b\\[c\\\\d](/a.png#w=50)')
    expect(img.alt).toBe('a]b[c\\d')
  })
})

describe('lightbox zoom', () => {
  const content = { width: 400, height: 300 }
  const stage = { width: 400, height: 600 }

  test('zooming keeps the point under the pointer still', () => {
    const view = zoomViewAt(IDENTITY_ZOOM, 2, { x: 100, y: 50 }, content, stage)
    expect(view.scale).toBe(2)
    // The image pixel at (100, 50) from the centre stays at (100, 50): x + 2 * 50 = 100.
    expect(view.x).toBe(-100)
    // Vertically the doubled image (600px) still fits the stage, so it stays centred.
    expect(view.y).toBe(0)
  })

  test('scale and pan are bounded, back to 1 recentres', () => {
    expect(zoomViewAt(IDENTITY_ZOOM, 9, { x: 0, y: 0 }, content, stage).scale).toBe(4)
    expect(clampZoomView({ scale: 2, x: 999, y: -999 }, content, stage)).toEqual({ scale: 2, x: 200, y: 0 })
    expect(clampZoomView({ scale: 0.5, x: 30, y: 30 }, content, stage)).toEqual(IDENTITY_ZOOM)
  })
})

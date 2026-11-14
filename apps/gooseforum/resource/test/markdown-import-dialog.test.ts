// @vitest-environment happy-dom
import { afterEach, describe, expect, test, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { i18n } from '../src/runtime/i18n'
import MarkdownImportDialog from '../src/site/components/MarkdownImportDialog.vue'

const upload = vi.hoisted(() => vi.fn(async (file: File) => `/file/img/${file.name}`))
vi.mock('@/runtime/api', async importOriginal => ({ ...await importOriginal<typeof import('../src/runtime/api')>(), uploadImage: upload }))
vi.mock('@/runtime/image', () => ({ MAX_IMAGE_UPLOAD_BYTES: 10 * 1024 * 1024, validateImageFile: () => '', processImageFile: async (file: File) => ({ file }) }))

let wrapper: VueWrapper | undefined
afterEach(() => { wrapper?.unmount(); wrapper = undefined; upload.mockClear(); upload.mockImplementation(async (file: File) => `/file/img/${file.name}`); document.body.innerHTML = '' })

function fileAt(path: string, content = 'x', type = 'image/png') {
  const file = new File([content], path.split('/').pop()!, { type })
  Object.defineProperty(file, 'webkitRelativePath', { value: path })
  return file
}

async function choose(input: HTMLInputElement, files: File[]) {
  Object.defineProperty(input, 'files', { value: files, configurable: true })
  input.dispatchEvent(new Event('change'))
  for (let i = 0; i < 10; i++) await flushPromises()
}

const inputs = () => Array.from(document.body.querySelectorAll<HTMLInputElement>('input[type="file"]'))
const rows = () => Array.from(document.body.querySelectorAll<HTMLElement>('[data-test="import-image"]'))
const button = (root: ParentNode, label: string) =>
  Array.from(root.querySelectorAll<HTMLButtonElement>('button')).find(el => el.getAttribute('aria-label') === label || el.textContent?.trim() === label)!
const tr = (key: string) => i18n.global.t(`editor.import.${key}`)

describe('Markdown 导入对话框的本地图片与回滚', () => {
  test('按文档目录自动匹配文件夹里的图片，可补选、移除、恢复、替换与还原', async () => {
    wrapper = mount(MarkdownImportDialog, { props: { open: true, bodyEmpty: true }, attachTo: document.body, global: { plugins: [i18n] } })
    await flushPromises()
    const [, folderInput, imagesInput, replaceInput] = inputs()
    expect(folderInput.webkitdirectory).toBe(true)

    const markdown = fileAt('blog/posts/post.md', '正文\n\n![甲](img/a.png)\n\n![乙](../shared/b.png)\n\n![丙](missing/c.png)', 'text/markdown')
    await choose(folderInput, [markdown, fileAt('blog/posts/img/a.png'), fileAt('blog/shared/b.png'), fileAt('blog/notes.txt', 'n', 'text/plain')])

    expect(upload.mock.calls.map(([file]) => file.name)).toEqual(['a.png', 'b.png'])
    expect(rows()).toHaveLength(3)
    expect(rows()[0].textContent).toContain('blog/posts/img/a.png')
    const missing = document.body.querySelector('[data-test="import-missing-local"]')!
    expect(missing.textContent).toContain(i18n.global.t('editor.import.missingLocalTitle', { count: 1 }))

    // 已载入文档后再选文件夹只补齐图片，不重新载入文档、不重复上传
    await choose(folderInput, [markdown, fileAt('blog/posts/img/a.png')])
    expect(upload).toHaveBeenCalledTimes(2)
    // 补选图片文件后自动重新匹配，提示消失
    await choose(imagesInput, [new File(['c'], 'c.png', { type: 'image/png' })])
    expect(upload).toHaveBeenLastCalledWith(expect.objectContaining({ name: 'c.png' }))
    expect(document.body.querySelector('[data-test="import-missing-local"]')).toBeNull()

    // 移除后导入只保留说明文字，恢复后不重新上传
    button(rows()[0], tr('removeImage')).click()
    await flushPromises()
    expect(rows()[0].textContent).toContain(tr('statusExcludedText'))
    button(rows()[0], tr('restoreImage')).click()
    await flushPromises()

    // 替换第二张，再还原回替换前已上传的图片
    button(rows()[1], tr('replaceImage')).click()
    await choose(replaceInput, [new File(['n'], 'new.png', { type: 'image/png' })])
    expect(rows()[1].textContent).toContain(tr('statusReplaced'))
    button(rows()[1], tr('revertReplacement')).click()
    await flushPromises()
    expect(upload).toHaveBeenCalledTimes(4)

    button(rows()[2], tr('removeImage')).click()
    await flushPromises()
    button(document.body, tr('confirm')).click()
    await flushPromises()
    expect(wrapper.emitted('apply')?.[0]?.[0]).toEqual({
      title: '',
      body: '正文\n\n![甲](/file/img/a.png)\n\n![乙](/file/img/b.png)\n\n丙',
      mode: 'replace',
    })
  })

  test('文件夹里有多份文档时先让用户选择', async () => {
    wrapper = mount(MarkdownImportDialog, { props: { open: true, bodyEmpty: true }, attachTo: document.body, global: { plugins: [i18n] } })
    await flushPromises()
    const [, folderInput] = inputs()
    await choose(folderInput, [fileAt('notes/b.md', '# 乙\n\n![](a.png)', 'text/markdown'), fileAt('notes/a.md', '# 甲', 'text/markdown'), fileAt('notes/a.png')])
    const choices = document.body.querySelector('[data-test="import-markdown-choices"]')!
    expect(Array.from(choices.querySelectorAll('span[title]')).map(span => span.getAttribute('title'))).toEqual(['notes/a.md', 'notes/b.md'])
    choices.querySelectorAll('button')[1].click()
    for (let i = 0; i < 10; i++) await flushPromises()
    expect(upload).toHaveBeenCalledWith(expect.objectContaining({ name: 'a.png' }))
    expect(rows()).toHaveLength(1)
  })

  test('较早选择但读取较慢的文档不会覆盖后选择的文档', async () => {
    wrapper = mount(MarkdownImportDialog, { props: { open: true, bodyEmpty: true }, attachTo: document.body, global: { plugins: [i18n] } })
    await flushPromises()
    const [fileInput] = inputs()
    let release!: (text: string) => void
    const slow = new File(['slow'], 'slow.md', { type: 'text/markdown' })
    Object.defineProperty(slow, 'text', { value: () => new Promise<string>(resolve => { release = resolve }) })
    await choose(fileInput, [slow])
    await choose(fileInput, [new File(['新文档'], 'fast.md', { type: 'text/markdown' })])
    release('旧文档')
    for (let i = 0; i < 10; i++) await flushPromises()
    button(document.body, tr('confirm')).click()
    await flushPromises()
    expect(wrapper.emitted('apply')?.[0]?.[0]).toMatchObject({ body: '新文档' })
  })

  test('连续替换后还原回到最近一次替换前的图片，替换失败时也是', async () => {
    wrapper = mount(MarkdownImportDialog, { props: { open: true, bodyEmpty: true }, attachTo: document.body, global: { plugins: [i18n] } })
    await flushPromises()
    const [, folderInput, , replaceInput] = inputs()
    await choose(folderInput, [fileAt('doc/post.md', '![图](a.png)', 'text/markdown'), fileAt('doc/a.png')])
    const replace = async (name: string) => {
      button(rows()[0], tr('replaceImage')).click()
      await choose(replaceInput, [new File(['n'], name, { type: 'image/png' })])
    }
    const body = async () => {
      button(document.body, tr('confirm')).click()
      await flushPromises()
      return (wrapper!.emitted('apply')?.at(-1)?.[0] as { body: string }).body
    }
    await replace('b.png')
    await replace('c.png')
    button(rows()[0], tr('revertReplacement')).click()
    await flushPromises()
    expect(rows()[0].querySelector('img')?.getAttribute('src')).toBe('/file/img/b.png')

    // 第二次替换失败：还原仍回到 B
    await replace('c2.png')
    upload.mockImplementationOnce(async () => { throw new Error('upload failed') })
    await replace('d.png')
    expect(rows()[0].textContent).toContain('upload failed')
    button(rows()[0], tr('revertReplacement')).click()
    await flushPromises()
    expect(await body()).toBe('![图](/file/img/c2.png)')
  })

  test('一级标题里的图片随正文上传；用作标题时只移除标题行', async () => {
    const source = '# ![封面](a.png) 期末复习\n\n正文'
    for (const titleEmpty of [true, false]) {
      wrapper = mount(MarkdownImportDialog, { props: { open: true, bodyEmpty: true, allowTitle: true, titleEmpty }, attachTo: document.body, global: { plugins: [i18n] } })
      await flushPromises()
      const [, folderInput] = inputs()
      await choose(folderInput, [fileAt('doc/post.md', source, 'text/markdown'), fileAt('doc/a.png')])
      expect(upload).toHaveBeenCalledWith(expect.objectContaining({ name: 'a.png' }))
      expect(document.body.querySelector('[data-test="import-images-status"]')?.textContent).toBe(
        i18n.global.t('editor.import.imagesStatus', { done: 1, total: 1 }),
      )
      button(document.body, tr('confirm')).click()
      await flushPromises()
      expect(wrapper.emitted('apply')?.[0]?.[0]).toMatchObject(
        titleEmpty
          ? { title: '期末复习', body: '正文' }
          : { title: '', body: '# ![封面](/file/img/a.png) 期末复习\n\n正文' },
      )
      wrapper.unmount()
      wrapper = undefined
      document.body.innerHTML = ''
      upload.mockClear()
    }
  })
})

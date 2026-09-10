import assert from 'node:assert/strict'
import { after, before, test } from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright'
import { createServer } from 'vite'
let server, browser, origin
before(async () => {
  server = await createServer({ root: fileURLToPath(new URL('../', import.meta.url)), server: { host: '127.0.0.1', port: 0, open: false } })
  await server.listen()
  origin = `http://127.0.0.1:${server.httpServer.address().port}`
  browser = await chromium.launch()
})
after(async () => { await browser?.close(); await server?.close() })
for (const input of [String.raw`\(x+y\)`, String.raw`\[x+y\]`, String.raw`\begin{align}a&=b\end{align}`, String.raw`\begin{pmatrix}a&b\\c&d\end{pmatrix}`]) {
  test(`Vditor preserves and the reading view renders ${input}`, async () => {
    const page = await browser.newPage()
    try {
      await page.goto(`${origin}/assets/test/fixtures/browser/math-audit.html?source=${encodeURIComponent(input)}`)
      await page.waitForFunction(() => window.mathAudit?.editor.value?.editorReady)
      const result = await page.evaluate(async () => {
        const api = window.mathAudit
        const stored = api.editor.value.getValue()
        const root = document.querySelector('#rendered')
        root.innerHTML = api.renderMarkdownPreview(stored)
        await api.enhanceMathText(root)
        return { stored, math: root.querySelectorAll('.katex').length, errors: root.querySelectorAll('.katex-error').length }
      })
      assert.equal(result.stored.trim(), input)
      assert.equal(result.math, 1)
      assert.equal(result.errors, 0)
    } finally { await page.close() }
  })
}

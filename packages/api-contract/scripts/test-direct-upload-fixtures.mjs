import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

test('direct upload completion fixture identifies the published object consistently', () => {
  const { result } = JSON.parse(readFileSync(new URL('../fixtures/direct-img-upload-complete-success.json', import.meta.url), 'utf8'))
  assert.equal(result.url, `/file/img/${result.filename}`)
  assert.equal(result.imageMetadata.url, result.url)
})

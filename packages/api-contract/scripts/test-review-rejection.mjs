import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import yaml from 'js-yaml'
import Ajv2020 from '@redocly/ajv/dist/2020.js'
const schemas = yaml.load(readFileSync(new URL('../components/schemas.yaml', import.meta.url), 'utf8'))
const validate = new Ajv2020({ strict: false, validateFormats: false }).compile(schemas.AdminReviewActionRequest)
const base = { kind: 'topic', id: 7, revisionId: 19 }
test('approval and personal-sticker review retain their request compatibility', () => {
  assert.equal(validate({ ...base, approve: true }), true)
  assert.equal(validate({ ...base, kind: 'sticker', approve: false }), true)
})
test('manual topic and post rejection require an explanation, including omitted approve', () => {
  for (const kind of ['topic', 'post']) for (const approve of [false, undefined]) for (const reason of [undefined, '', ' \n\u2003 ']) {
    assert.equal(validate({ ...base, kind, ...(approve === undefined ? {} : { approve }), ...(reason === undefined ? {} : { reason }) }), false)
  }
})
test('the request boundary counts Unicode characters consistently', () => {
  assert.equal(validate({ ...base, approve: false, reason: '😀'.repeat(512) }), true)
  assert.equal(validate({ ...base, approve: false, reason: '😀'.repeat(513) }), false)
})

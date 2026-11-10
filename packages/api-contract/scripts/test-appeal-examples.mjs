import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import yaml from 'js-yaml'

const source = new URL('../paths/moderation-appeals.yaml', import.meta.url)
const paths = yaml.load(readFileSync(source, 'utf8'))
const fixture = example => JSON.parse(readFileSync(new URL(example.externalValue, source), 'utf8'))

for (const method of ['get', 'post']) {
  test(`author appeal ${method} documents the privacy-preserving not-found envelope`, () => {
    const examples = paths.contentAppeals[method].responses['200'].content['application/json'].examples
    assert.ok(examples.notFound)
    assert.equal(examples.permissionDenied, undefined)
    assert.deepEqual(fixture(examples.notFound), { result: null, code: 1, messageCode: 'content.appeal.notFound' })
  })
}

test('moderator permission failures and author write CSRF failures keep their distinct contracts', () => {
  for (const key of ['moderationAppeals', 'moderationAppealAction']) {
    const examples = paths[key].post.responses['200'].content['application/json'].examples
    assert.equal(fixture(examples.permissionDenied).messageCode, 'permission.denied')
  }
  const forbidden = paths.contentAppeals.post.responses['403'].content['application/json']
  assert.equal(fixture(forbidden.examples.failure).messageCode, 'auth.csrf.rejected')
})

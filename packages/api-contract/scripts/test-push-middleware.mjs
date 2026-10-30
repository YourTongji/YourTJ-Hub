import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { test } from 'node:test'
import yaml from 'js-yaml'

const paths = yaml.load(await readFile(new URL('../paths/forum-notifications.yaml', import.meta.url), 'utf8'))
for (const operation of ['subscribePush', 'unsubscribePush', 'registerPushDevice', 'unregisterPushDevice']) {
  test(`${operation} declares its actual writable-account and interact middleware failures`, async () => {
    const responses = paths[operation].post.responses
    const limited = responses['429']
    assert.ok(limited, 'RateLimitInteract requires a 429 response')
    assert.equal(limited.headers['Retry-After'].required, true)
    assert.deepEqual(limited.headers['Retry-After'].schema, { type: 'integer', minimum: 1 })
    const wire = limited.content['application/json']
    assert.equal(wire.schema.$ref, '../components/schemas.yaml#/RateLimitedFailure')
    const sample = JSON.parse(await readFile(new URL(wire.examples.rateLimited.externalValue, new URL('../paths/forum-notifications.yaml', import.meta.url)), 'utf8'))
    assert.equal(sample.messageCode, 'common.rateLimited')
    assert.equal(sample.params.action, 'interact')
    assert.ok(sample.params.retryAfterSeconds >= 1)
    const forbidden = responses['403']
    assert.match(forbidden.description, /pending.activation/)
    assert.equal(forbidden.content['application/json'].examples.emailRequired.externalValue, '../fixtures/permission-email-required.json')
    assert.equal(responses['401'].content['application/json'].examples.unauthenticated.externalValue, '../fixtures/auth-required.json')
  })
}

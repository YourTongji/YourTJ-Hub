import assert from 'node:assert/strict'
import { test } from 'node:test'
import { publishedTreeMatches } from '../scripts/production-build.mjs'

const tree = 'a'.repeat(40)
test('only skips a matching tree published in production, never a preview or previous version', async () => {
  for (const [manifest, expected] of [
    [{ schema: 1, context: 'production', tree }, true],
    [{ schema: 1, context: 'deploy-preview', tree }, false],
    [{ schema: 1, context: 'production', tree: 'b'.repeat(40) }, false],
    [{ schema: 2, context: 'production', tree }, false],
    [{}, false],
  ]) {
    assert.equal(await publishedTreeMatches('https://status.example.com', tree, async (url, init) => {
      assert.equal(String(url), 'https://status.example.com/status-build.json')
      assert.equal(init.redirect, 'error')
      assert.equal(init.cache, 'no-store')
      return Response.json(manifest)
    }), expected)
  }
})

test('builds when the published manifest cannot be trusted or reached', async () => {
  for (const fetcher of [
    async () => { throw new Error('offline') },
    async () => new Response('', { status: 404 }),
    async () => new Response('<html>unavailable</html>'),
    async () => new Response(' '.repeat(4096)),
  ]) assert.equal(await publishedTreeMatches('https://status.example.com', tree, fetcher), false)
  for (const url of ['', 'http://status.example.com', 'https://user:secret@status.example.com', 'https://status.example.com/preview']) {
    assert.equal(await publishedTreeMatches(url, tree, async () => { assert.fail('must not fetch') }), false)
  }
})

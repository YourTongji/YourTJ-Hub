import assert from 'node:assert/strict'
import {readFileSync} from 'node:fs'
import {test} from 'node:test'

const source=readFileSync(new URL('../.github/workflows/ci.yml',import.meta.url),'utf8')
const template=source.match(/^  group: (.+)$/m)?.[1]
assert.ok(template,'CI declares its workflow concurrency group')
// Evaluate the workflow's actual scalar expressions with representative event
// contexts. This covers the small ==/&&/|| subset used by this group template.
function group(event,ref='refs/heads/dev',number) {
 const context={event_name:event,ref,event:{pull_request:{number}}}
 return template.replace(/\$\{\{([\s\S]*?)\}\}/g,(_,expression)=>{
  const scalar=expression.replace(/\bgithub(?:\.[A-Za-z_]\w*)+/g,path=>JSON.stringify(path.split('.').slice(1).reduce((value,key)=>value?.[key],context)??null))
  return String(Function(`"use strict"; return (${scalar});`)())
 })
}
test('manual full validation cannot cancel the same-source dev push',()=>assert.notEqual(group('workflow_dispatch'),group('push')))
test('successive pushes on one branch still replace older validation',()=>assert.equal(group('push'),group('push')))
test('PR cancellation follows PR identity rather than its current source ref',()=>assert.equal(group('pull_request','refs/pull/10/merge',10),group('pull_request','refs/heads/updated-head',10)))
test('separate PRs and branches retain separate groups',()=>{assert.notEqual(group('pull_request','refs/pull/10/merge',10),group('pull_request','refs/pull/11/merge',11));assert.notEqual(group('workflow_dispatch'),group('workflow_dispatch','refs/heads/main'))})

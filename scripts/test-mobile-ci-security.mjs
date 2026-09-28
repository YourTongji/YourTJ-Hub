import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { test } from 'node:test';

for (const name of ['ci-mobile.yml', 'ci-mobile-native.yml']) {
  const workflow = readFileSync(new URL(`../.github/workflows/${name}`, import.meta.url), 'utf8');
  const steps = workflow.split(/^      - /m).slice(1);
  test(`${name}: external actions use immutable commits`, () => {
    const actions = [...workflow.matchAll(/\buses: (\S+)/g)].map(match => match[1]);
    assert.ok(actions.length > 0);
    for (const action of actions) assert.match(action, /@[0-9a-f]{40}$/, action);
  });
  test(`${name}: PR build steps cannot inherit checkout credentials`, () => {
    const checkouts = steps.filter(step => step.startsWith('uses: actions/checkout@'));
    assert.ok(checkouts.length > 0);
    for (const checkout of checkouts) {
      assert.match(checkout, /^          persist-credentials: false$/m);
    }
  });
}

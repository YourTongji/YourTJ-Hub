import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { matchesGlob } from 'node:path/posix';
import { test } from 'node:test';

const workflow = readFileSync(new URL('../.github/workflows/ci-mobile.yml', import.meta.url), 'utf8');
// Read the actual path list so a change to the workflow cannot leave this test
// checking a separate copy of its filters. These paths use ordinary POSIX globs.
const block = workflow.match(/^            flutter:\n((?:              - .+\n)+)/m)?.[1];
assert.ok(block, 'Flutter input filter must be present');
const patterns = [...block.matchAll(/- '([^']+)'/g)].map((match) => match[1]);
const selected = (path) => patterns.some((pattern) => matchesGlob(path, pattern));

for (const path of [
  'apps/mobile/packages/forum_app/integration_test/campus_native_test.dart',
  'apps/mobile/packages/forum_app/test_driver/integration_driver.dart',
  'apps/mobile/packages/forum_app/tool/generate_launcher_artwork.dart',
  'apps/mobile/packages/core/lib/src/client.dart',
  'apps/mobile/packages/ui_kit/test/theme_test.dart',
]) {
  test(`analyze Dart input: ${path}`, () => assert.equal(selected(path), true));
}

test('documentation and store metadata do not select Flutter jobs', () => {
  for (const path of ['apps/mobile/README.md', 'apps/mobile/store/zh-CN/description.txt']) {
    assert.equal(selected(path), false, path);
  }
});

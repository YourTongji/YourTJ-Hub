import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { test } from 'node:test';
// Exercise the authoritative classifier, including Dart files outside lib/ and test/.
for (const path of [
  'apps/mobile/packages/forum_app/integration_test/campus_native_test.dart',
  'apps/mobile/packages/forum_app/test_driver/integration_driver.dart',
  'apps/mobile/packages/forum_app/tool/generate_launcher_artwork.dart',
  'apps/mobile/packages/core/lib/src/client.dart',
  'apps/mobile/packages/ui_kit/test/theme_test.dart',
]) {
  test(`analyze Dart input: ${path}`, () => {
    const run = spawnSync('python3', ['-c', 'import sys,json; sys.path.insert(0,"scripts/ci"); from select_inputs import select; print(json.dumps(select([sys.argv[1]])))', path], {encoding: 'utf8'});
    assert.equal(run.status, 0, run.stderr);
    assert.equal(JSON.parse(run.stdout).mobile.run, true);
  });
}

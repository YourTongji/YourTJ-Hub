"""Workflow jobs that re-run release-freeze must hold the token permission to do it."""
from pathlib import Path
import re
import unittest

WORKFLOWS = Path(__file__).resolve().parents[2] / '.github' / 'workflows'
RERUNS_FREEZE = re.compile(r'cycle_control\.py checks\b')


def permissions(lines, indent, default=None):
    """Read a `permissions:` mapping (block or flow style) at the given indentation."""
    key = ' ' * indent + 'permissions:'
    for i, line in enumerate(lines):
        if not line.startswith(key):
            continue
        rest = line[len(key):].strip()
        if rest.startswith('{'):
            pairs = (item.split(':', 1) for item in rest.strip('{} ').split(',') if item.strip())
            return {k.strip(): v.strip() for k, v in pairs}
        if rest:
            return {'*': rest}
        result = {}
        for item in lines[i + 1:]:
            match = re.match(' ' * (indent + 2) + r'([a-z-]+):\s*(\S+)', item)
            if not match:
                break
            result[match.group(1)] = match.group(2)
        return result
    return default or {}


def jobs(text):
    """Split the `jobs:` section into {job id: lines} by its two-space job keys."""
    if '\njobs:\n' not in text:
        return {}
    result, name = {}, None
    for line in text.split('\njobs:\n', 1)[1].splitlines():
        match = re.match(r'  ([A-Za-z0-9_-]+):\s*$', line)
        if match:
            name = match.group(1)
            result[name] = []
        elif line and not line.startswith(' '):
            break
        elif name and line.startswith('    '):
            result[name].append(line)
    return result


class FreezeRerunPermissions(unittest.TestCase):
    def test_every_job_that_reruns_release_freeze_can_write_actions(self):
        checked = []
        for path in sorted(WORKFLOWS.glob('*.y*ml')):
            text = path.read_text(encoding='utf-8')
            if not RERUNS_FREEZE.search(text):
                continue
            default = permissions(text.splitlines(), 0)
            for name, lines in jobs(text).items():
                if not any(RERUNS_FREEZE.search(line) for line in lines):
                    continue
                checked.append(f'{path.name}:{name}')
                granted = permissions(lines, 4, default)
                with self.subTest(job=checked[-1]):
                    self.assertIn(granted.get('actions', granted.get('*')), ('write', 'write-all'),
                                  'cycle_control.py checks re-runs release-freeze and needs actions: write')
        self.assertIn('release-prerelease.yml:begin', checked)
        self.assertIn('release-control.yml:project', checked)


class CatalogFailureIsolation(unittest.TestCase):
    def test_huawei_failure_does_not_block_catalog_but_stays_visible(self):
        text = (WORKFLOWS / 'mobile-release-catalog.yml').read_text()
        steps = re.split(r'(?m)^      - ', text)[1:]
        huawei = next(step for step in steps if 'run: python3 scripts/release/harmony_reconcile.py' in step)
        self.assertRegex(huawei, r'(?m)^        continue-on-error: true$')
        self.assertRegex(huawei, r'(?m)^        id: harmony$')
        catalog = next(step for step in steps if 'run: python3 scripts/release/workflow.py publish-catalog' in step)
        dev = next(step for step in steps if 'run: python3 scripts/release/dev_delivery.py status' in step)
        self.assertNotIn('steps.harmony', catalog)
        self.assertNotIn('steps.harmony', dev)
        report = next(step for step in steps if "steps.harmony.outcome == 'failure'" in step)
        self.assertIn('!cancelled()', report)
        self.assertRegex(report, r'(?m)^          exit 1$')
        self.assertLess(steps.index(huawei), steps.index(dev))
        self.assertLess(steps.index(dev), steps.index(catalog))
        self.assertLess(steps.index(catalog), steps.index(report))


if __name__ == '__main__':
    unittest.main()

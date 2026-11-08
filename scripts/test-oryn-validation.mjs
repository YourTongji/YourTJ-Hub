import assert from 'node:assert/strict';
import { test } from 'node:test';
import { selectedScopes } from '../.github/oryn/validate.mjs';

test('docs need governance, without application suites', () => {
  assert.deepEqual(selectedScopes(['docs/operations/example.md']), ['governance']);
});
test('database changes require backend and PostgreSQL', () => {
  assert.deepEqual(selectedScopes(['apps/gooseforum/app/models/forum/topic/topic.go']), ['governance', 'backend', 'postgres']);
});
test('HTTP changes also verify the controlled API contract', () => {
  assert.deepEqual(selectedScopes(['apps/gooseforum/app/http/controllers/api/post.go']), ['governance', 'backend', 'contract']);
});
test('web and shared contract changes include their consumers', () => {
  assert.deepEqual(selectedScopes(['packages/api-contract/openapi.yaml']), ['governance', 'backend', 'frontend', 'contract', 'mobile']);
});
test('unknown executable surfaces fail closed by checking every application domain', () => {
  assert.deepEqual(selectedScopes(['scripts/new-build.sh']), ['governance', 'backend', 'postgres', 'frontend', 'contract', 'mobile']);
});

test('scope includes committed PR changes, staged repair and untracked files', async () => {
  const { mkdtempSync, writeFileSync, rmSync } = await import('node:fs');
  const { tmpdir } = await import('node:os');
  const { join } = await import('node:path');
  const { execFileSync } = await import('node:child_process');
  const { changedPaths } = await import('../.github/oryn/validate.mjs');
  const directory = mkdtempSync(join(tmpdir(), 'oryn-scope-'));
  const previous = process.cwd();
  const git = (...args) => execFileSync('git', args, { cwd: directory, encoding: 'utf8' }).trim();
  try {
    git('init', '-q');
    git('config', 'user.name', 'test');
    git('config', 'user.email', 'test@example.invalid');
    writeFileSync(join(directory, 'base.txt'), 'base');
    git('add', '.');
    git('commit', '-qm', 'base');
    const base = git('rev-parse', 'HEAD');
    writeFileSync(join(directory, 'committed.txt'), 'PR');
    git('add', '.');
    git('commit', '-qm', 'PR');
    writeFileSync(join(directory, 'staged.txt'), 'repair');
    git('add', '.');
    writeFileSync(join(directory, 'untracked.txt'), 'repair');
    process.chdir(directory);
    assert.deepEqual(changedPaths(base).sort(), ['committed.txt', 'staged.txt', 'untracked.txt']);
    assert.throws(() => changedPaths('0'.repeat(40)), /Cannot establish/);
  } finally {
    process.chdir(previous);
    rmSync(directory, { recursive: true, force: true });
  }
});

// The checked-in concurrency expression uses the JS-compatible boolean/format subset of
// GitHub expressions. Evaluate that exact text with typed workflow-dispatch fixtures.
async function orynConcurrency(inputs, event = 'workflow_dispatch', runId = 101) {
  const { readFileSync } = await import('node:fs');
  const workflow = readFileSync(new URL('../.github/workflows/oryn.yml', import.meta.url), 'utf8');
  const expression = workflow.match(/^  group: \$\{\{ (.+) \}\}$/m)?.[1];
  assert.ok(expression, 'Oryn must declare its workflow concurrency group');
  const evaluate = new Function('github', 'inputs', 'vars', 'format', `return (${expression});`);
  const format = (template, ...values) => template.replace(/\{(\d+)\}/g, (_, index) => String(values[index]));
  return evaluate({ event_name: event, run_id: runId, event: { comment: { user: { type: 'User' } } } },
    { release_notes_smoke: false, preflight: false, publish: false, ...inputs },
    { ORYN_ENABLED: 'true', ORYN_EVENT_ENABLED: 'true' }, format);
}

test('non-publishing manual probes can run without waiting for the publication queue', async () => {
  assert.equal(await orynConcurrency({}), 'oryn-probe-101');
  assert.equal(await orynConcurrency({}, 'workflow_dispatch', 102), 'oryn-probe-102');
  assert.equal(await orynConcurrency({ preflight: true, publish: true }), 'oryn-probe-101');
  assert.equal(await orynConcurrency({ release_notes_smoke: true }), 'oryn-release-smoke-101');
});

const verifiedOrynSha = 'ad2b9847c01cba36f8a0abd5564df7144d47878e';

test('the setup action and operator guide pin the same verified Oryn commit', async () => {
  const { readFileSync } = await import('node:fs');
  const action = readFileSync(new URL('../.github/actions/setup-oryn/action.yml', import.meta.url), 'utf8');
  const guide = readFileSync(new URL('../docs/operations/oryn.md', import.meta.url), 'utf8');
  const refs = action.match(/^ {8}ref: ([0-9a-f]{40})$/m);
  assert.deepEqual(refs?.slice(1), [verifiedOrynSha]);
  assert.equal(guide.includes(verifiedOrynSha), true);
  assert.equal(guide.includes('32af3fbc705ea34f8e4229d38035513ce165d511'), false);
});

const hubSensitivePaths = [
  '.github/actions/',
  '.github/oryn/',
  '.github/workflows/',
  'apps/gooseforum/app/bundles/jwtopt/',
  'apps/gooseforum/app/bundles/logincrypto/',
  'apps/gooseforum/app/bundles/oidcprovider/',
  'apps/gooseforum/app/bundles/sessionstore/',
  'apps/gooseforum/app/http/controllers/api/mobileWebSession',
  'apps/gooseforum/app/http/controllers/api/oauth',
  'apps/gooseforum/app/http/controllers/api/oidc',
  'apps/gooseforum/app/http/controllers/api/passkey',
  'apps/gooseforum/app/http/controllers/api/session',
  'apps/gooseforum/app/http/controllers/api/tongjiLogin',
  'apps/gooseforum/app/http/controllers/api/tongji_login',
  'apps/gooseforum/app/http/controllers/api/totp',
  'apps/gooseforum/app/http/middleware/agentAuth',
  'apps/gooseforum/app/http/middleware/checkPermission',
  'apps/gooseforum/app/http/middleware/csrfProtection',
  'apps/gooseforum/app/http/middleware/jwtAuth',
  'apps/gooseforum/app/http/middleware/securityHeaders',
  'apps/gooseforum/app/http/middleware/totpChallenge',
  'apps/gooseforum/app/http/routes/contract_passkeys',
  'apps/gooseforum/app/http/routes/contract_totp',
  'apps/gooseforum/app/http/routes/csrf_',
  'apps/gooseforum/app/http/routes/oidc_routes',
  'apps/gooseforum/app/http/routes/totp_',
  'apps/gooseforum/app/models/forum/oidcAuthRequests/',
  'apps/gooseforum/app/models/forum/rolePermissionRs/',
  'apps/gooseforum/app/models/forum/userOAuth/',
  'apps/gooseforum/app/models/forum/userPasskeys/',
  'apps/gooseforum/app/models/forum/userSessions/',
  'apps/gooseforum/app/models/forum/userTotp/',
  'apps/gooseforum/app/models/forum/userTotpChallenges/',
  'apps/gooseforum/app/models/forum/userTotpRecoveryCodes/',
  'apps/gooseforum/app/service/appleauthservice/',
  'apps/gooseforum/app/service/campusservice/login',
  'apps/gooseforum/app/service/oauthservice/',
  'apps/gooseforum/app/service/oidcservice/',
  'apps/gooseforum/app/service/passkeyservice/',
  'apps/gooseforum/app/service/sessionservice/',
  'apps/gooseforum/app/service/totpservice/',
  'scripts/ci/',
];

function prefixCovers(file, prefixes) {
  return prefixes.some((prefix) => prefix.length > 0 && prefix.length <= 200 && !prefix.startsWith('/') && !prefix.includes('..') && !prefix.includes('\\') && file.startsWith(prefix));
}

test('Hub Oryn policy enables low-risk approval and keeps check, lesson, and merge gates', async () => {
  const { readFileSync } = await import('node:fs');
  const policy = JSON.parse(readFileSync(new URL('../.github/oryn/repositories.json', import.meta.url), 'utf8'));
  const repository = policy.repositories.find((item) => item.name === 'YourTongji/YourTJ-Hub');
  assert.ok(repository);
  assert.equal(repository.allowLowRiskApprove, true);
  assert.deepEqual(repository.requiredChecks, ['ci-required']);
  assert.deepEqual(repository.trustedCheckApps, [{ name: 'ci-required', app: 'github-actions' }]);
  assert.deepEqual(repository.lessonPaths, ['docs/decisions', 'docs/postmortems']);
  assert.equal(repository.allowMerge, true);
  assert.deepEqual(repository.protectedMergePaths, ['releases/requests/']);
  assert.deepEqual(repository.sensitivePaths, hubSensitivePaths);
  assert.equal(repository.sensitivePaths.includes('releases/requests/'), false);
  assert.equal(prefixCovers('.github/oryn/repositories.json', repository.sensitivePaths), true);
  assert.equal(prefixCovers('scripts/ci/select_inputs.py', repository.sensitivePaths), true);
  assert.equal(prefixCovers('apps/gooseforum/app/bundles/oidcprovider/oidcprovider.go', repository.sensitivePaths), true);
  assert.equal(prefixCovers('apps/gooseforum/app/http/middleware/jwtAuth.go', repository.sensitivePaths), true);
  assert.equal(prefixCovers('releases/requests/example/request.json', repository.sensitivePaths), false);
  assert.equal(prefixCovers('releases/requests/example/request.json', repository.protectedMergePaths), true);
  for (const ordinary of ['docs/operations/oryn.md', 'apps/gooseforum/app/http/middleware/logger.go', 'apps/gooseforum/app/http/controllers/api/wikiController.go', 'apps/gooseforum/app/service/datamigration/topic_excerpt.go']) {
    assert.equal(prefixCovers(ordinary, repository.sensitivePaths), false, ordinary);
  }
});

test('the pinned Oryn modules parse this policy and classify the same paths', async (t) => {
  const source = process.env.ORYN_PIN_SRC;
  if (!source) {
    t.skip('ORYN_PIN_SRC is unset, so this run does not import the private pin');
    return;
  }
  const { spawnSync } = await import('node:child_process');
  const { existsSync } = await import('node:fs');
  const bun = existsSync('/home/ubuntu/.bun/bin/bun') ? '/home/ubuntu/.bun/bin/bun' : 'bun';
  const probe = spawnSync(bun, ['-e', `
    import { RepositoryPolicy } from ${JSON.stringify(source + '/src/contracts.ts')};
    import { childFindingsClear, childRiskLine, isSensitivePath } from ${JSON.stringify(source + '/src/review-dispatch.ts')};
    import { readFileSync } from 'node:fs';
    const raw = JSON.parse(readFileSync(${JSON.stringify(new URL('../.github/oryn/repositories.json', import.meta.url).pathname)}, 'utf8'));
    const parsed = RepositoryPolicy.parse(raw.repositories[0]);
    const prefixes = [...parsed.sensitivePaths, ...parsed.protectedMergePaths];
    const files = {
      policy: '.github/oryn/repositories.json',
      ci: 'scripts/ci/select_inputs.py',
      oidc: 'apps/gooseforum/app/bundles/oidcprovider/oidcprovider.go',
      jwt: 'apps/gooseforum/app/http/middleware/jwtAuth.go',
      migration: 'apps/gooseforum/app/migration/migration.go',
      deploy: 'deploy/render_config.py',
      release: 'releases/requests/example/request.json',
      ordinary: 'docs/operations/oryn.md',
      logger: 'apps/gooseforum/app/http/middleware/logger.go',
      wiki: 'apps/gooseforum/app/http/controllers/api/wikiController.go',
      excerpt: 'apps/gooseforum/app/service/datamigration/topic_excerpt.go',
      author: 'src/author.ts',
    };
    const classified = Object.fromEntries(Object.entries(files).map(([key, file]) => [key, isSensitivePath(file, prefixes)]));
    console.log(JSON.stringify({
      allowLowRiskApprove: parsed.allowLowRiskApprove,
      requiredChecks: parsed.requiredChecks,
      trustedCheckApps: parsed.trustedCheckApps,
      lessonPaths: parsed.lessonPaths,
      sensitiveCount: parsed.sensitivePaths.length,
      unsafePrefixIgnored: isSensitivePath('src/util.ts', ['../src/']),
      classified,
      findings: {
        clear: childFindingsClear('Findings: none\\nUnknowns: none'),
        period: childFindingsClear('Findings: none.\\nUnknowns: none'),
        present: childFindingsClear('Findings: authentication bypass\\nUnknowns: none'),
        missingUnknowns: childFindingsClear('Findings: none'),
      },
      risk: {
        low: childRiskLine('Risk: low'),
        medium: childRiskLine('Risk: medium'),
        high: childRiskLine('Risk: HIGH'),
        lateHigh: childRiskLine(['Risk: low', ...Array.from({length: 30}, () => 'note'), 'Risk: high'].join('\\n')),
        unknown: childRiskLine('Risk: severe'),
      },
    }));
  `], { encoding: 'utf8' });
  assert.equal(probe.status, 0, probe.stderr);
  const report = JSON.parse(probe.stdout);
  assert.equal(report.allowLowRiskApprove, true);
  assert.deepEqual(report.requiredChecks, ['ci-required']);
  assert.deepEqual(report.trustedCheckApps, [{ name: 'ci-required', app: 'github-actions' }]);
  assert.deepEqual(report.lessonPaths, ['docs/decisions', 'docs/postmortems']);
  assert.equal(report.sensitiveCount, hubSensitivePaths.length);
  assert.equal(report.unsafePrefixIgnored, false);
  assert.equal(report.classified.policy, true);
  assert.equal(report.classified.ci, true);
  assert.equal(report.classified.oidc, true);
  assert.equal(report.classified.jwt, true);
  assert.equal(report.classified.migration, true);
  assert.equal(report.classified.deploy, true);
  assert.equal(report.classified.release, true);
  assert.equal(report.classified.ordinary, false);
  assert.equal(report.classified.logger, false);
  assert.equal(report.classified.wiki, false);
  assert.equal(report.classified.excerpt, false);
  assert.equal(report.classified.author, false);
  assert.deepEqual(report.findings, { clear: true, period: false, present: false, missingUnknowns: false });
  assert.deepEqual(report.risk, { low: 'low', medium: 'medium', high: 'high', lateHigh: 'high', unknown: 'unknown' });
});

test('publishing manual work and automatic events keep their shared receipt-writer queue', async () => {
  assert.equal(await orynConcurrency({ publish: true }), 'oryn-sweep');
  for (const event of ['schedule', 'pull_request_target', 'issues', 'issue_comment']) {
    assert.equal(await orynConcurrency({}, event), 'oryn-sweep');
  }
});

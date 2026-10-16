import assert from 'node:assert/strict';
import { chmodSync, existsSync, mkdtempSync, mkdirSync, readFileSync, realpathSync, rmSync, symlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { delimiter, join } from 'node:path';
import { spawnSync } from 'node:child_process';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';

const hook = fileURLToPath(new URL('./hooks/check-golangci.sh', import.meta.url));
const freshSha = 'f'.repeat(40);
const staleSha = '1'.repeat(40);

function fixture({ remotes = [], prBase = '', ghError = '', ghApiError = '', fetchFails = '', includeGh = true, localBaseSha = '' } = {}) {
  const directory = mkdtempSync(join(tmpdir(), 'yourtj-pre-push-'));
  const root = join(directory, 'repo');
  const bin = join(directory, 'bin');
  const noGhBin = join(directory, 'no-gh-bin');
  mkdirSync(join(root, 'apps', 'gooseforum'), { recursive: true });
  mkdirSync(bin);
  mkdirSync(noGhBin);
  if (!includeGh) {
    // Keep the test independent of host tools such as gh being installed in /usr/bin.
    // The test runner and hook also need these utilities on the restricted PATH.
    for (const name of ['bash', 'cat', 'tr']) {
      const executablePath = process.env.PATH.split(delimiter).map((path) => join(path, name)).find(existsSync);
      if (!executablePath) throw new Error(`${name} is required to test the GitHub CLI fallback`);
      symlinkSync(executablePath, join(noGhBin, name));
    }
  }

  const remoteNames = join(directory, 'remote-names');
  const remoteUrls = join(directory, 'remote-urls');
  const gitLog = join(directory, 'git.log');
  const lintArgs = join(directory, 'lint.args');
  const lintCwd = join(directory, 'lint.cwd');
  const ghOutput = join(directory, 'gh-output');
  const ghLog = join(directory, 'gh.log');
  const baseShaFile = join(directory, 'base-sha');
  writeFileSync(remoteNames, remotes.map(([name]) => name).join('\n') + (remotes.length ? '\n' : ''));
  writeFileSync(remoteUrls, remotes.map(([name, url]) => name + '\t' + url).join('\n') + (remotes.length ? '\n' : ''));
  writeFileSync(ghOutput, prBase);
  if (localBaseSha) writeFileSync(baseShaFile, localBaseSha + '\n');

  const gitStub = [
    '#!/bin/sh',
    'printf "%s\\n" "$*" >> "$GIT_LOG"',
    'if [ "$1" = "rev-parse" ] && [ "$2" = "--show-toplevel" ]; then printf "%s\\n" "$REPO_ROOT"; exit 0; fi',
    'if [ "$1" = "remote" ] && [ "$#" = 1 ]; then cat "$REMOTE_NAMES"; exit 0; fi',
    'if [ "$1" = "config" ] && [ "$2" = "--get" ]; then',
    '  key=${3#remote.}; wanted=${key%.url}',
    '  while IFS="$(printf "\\t")" read -r name url; do [ "$name" = "$wanted" ] && { printf "%s\\n" "$url"; exit 0; }; done < "$REMOTE_URLS"',
    '  exit 1',
    'fi',
    'if [ "$1" = "check-ref-format" ]; then exit 0; fi',
    'if [ "$1" = "fetch" ]; then',
    '  if [ "$FETCH_FAILS" = "all" ] || { [ "$FETCH_FAILS" = "remote" ] && [ "$4" != "https://github.com/YourTongji/YourTJ-Hub.git" ]; }; then echo "mock fetch failure" >&2; exit 1; fi',
    '  printf "%s\\n" "$FRESH_SHA" > "$BASE_SHA_FILE"; exit 0',
    'fi',
    'if [ "$1" = "rev-parse" ] && [ "$2" = "--verify" ]; then',
    '  case "$3" in',
    '    origin/dev) printf "%s\\n" "$STALE_SHA"; exit 0 ;;',
    '    FETCH_HEAD\^\{commit\}) [ -f "$BASE_SHA_FILE" ] && { cat "$BASE_SHA_FILE"; exit 0; } ;;',
    '  esac',
    '  exit 1',
    'fi',
    'echo "unexpected git invocation: $*" >&2',
    'exit 2',
  ].join('\n');
  writeFileSync(join(bin, 'git'), gitStub);
  chmodSync(join(bin, 'git'), 0o755);

  const lintStub = [
    '#!/bin/sh',
    'printf "%s\\n" "$@" > "$LINT_ARGS"',
    'pwd > "$LINT_CWD"',
    'exit 0',
  ].join('\n');
  writeFileSync(join(bin, 'golangci-lint'), lintStub);
  chmodSync(join(bin, 'golangci-lint'), 0o755);

  if (includeGh) {
    const ghStub = [
      '#!/bin/sh',
      'if [ -n "$GH_ERROR" ]; then echo "$GH_ERROR" >&2; exit 1; fi',
      'if [ "$1" = "pr" ]; then',
      '  if [ -s "$GH_OUTPUT" ]; then base_repo=$(cut -f1 "$GH_OUTPUT"); printf "123\\thttps://github.com/%s/pull/123\\n" "$base_repo"; exit 0; fi',
      '  echo "no pull requests found for branch test" >&2; exit 1',
      'fi',
      'if [ "$1" = "api" ] && [ -n "$GH_API_ERROR" ]; then echo "$GH_API_ERROR" >&2; exit 1; fi',
      'if [ "$1" = "api" ] && [ -s "$GH_OUTPUT" ]; then printf "%s\\n" "$*" >> "$GH_LOG"; cat "$GH_OUTPUT"; exit 0; fi',
      'echo "unexpected gh invocation: $*" >&2; exit 2',
    ].join('\n');
    writeFileSync(join(bin, 'gh'), ghStub);
    chmodSync(join(bin, 'gh'), 0o755);
  }

  return {
    directory,
    root,
    bin,
    gitLog,
    lintArgs,
    env: {
      ...process.env,
      PATH: bin + ':' + (includeGh ? process.env.PATH : noGhBin),
      REPO_ROOT: root,
      REMOTE_NAMES: remoteNames,
      REMOTE_URLS: remoteUrls,
      GIT_LOG: gitLog,
      LINT_ARGS: lintArgs,
      LINT_CWD: lintCwd,
      GH_OUTPUT: ghOutput,
      GH_LOG: ghLog,
      BASE_SHA_FILE: baseShaFile,
      GH_ERROR: ghError,
      GH_API_ERROR: ghApiError,
      FETCH_FAILS: fetchFails,
      FRESH_SHA: freshSha,
      STALE_SHA: staleSha,
    },
  };
}

function runHook(f, extraEnv = {}) {
  return spawnSync('bash', [hook], {
    cwd: f.root,
    env: { ...f.env, ...extraEnv },
    encoding: 'utf8',
  });
}

function readLines(path) {
  try {
    return readFileSync(path, 'utf8').trimEnd().split('\n');
  } catch {
    return [];
  }
}

function command(commandName, args, options = {}) {
  const result = spawnSync(commandName, args, { encoding: 'utf8', ...options });
  assert.equal(result.status, 0, commandName + ' ' + args.join(' ') + ': ' + result.stderr);
  return result.stdout.trim();
}

test('fork PR uses the fetched base on a differently named remote even when origin/dev is stale', (t) => {
  const f = fixture({
    remotes: [
      ['origin', 'git@github.com:AlbertYang514/YourTJ-Hub.git'],
      ['canonical', 'https://github.com/YourTongji/YourTJ-Hub.git'],
    ],
    prBase: 'YourTongji/YourTJ-Hub\tdev\n',
    localBaseSha: staleSha,
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.equal(result.status, 0, result.stderr);
  assert.ok(readLines(f.gitLog).some((line) => line.includes('fetch --no-tags --quiet canonical +refs/heads/dev')));
  assert.ok(readLines(join(f.directory, 'gh.log')).some((line) => line.includes('repos/YourTongji/YourTJ-Hub/pulls/123')));
  assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
  assert.deepEqual(readLines(join(f.directory, 'lint.cwd')), [join(f.root, 'apps', 'gooseforum')]);
  assert.ok(!readLines(f.lintArgs).includes('--new-from-rev=origin/dev'));
});

test('a single fork remote uses canonical dev when GitHub confirms no PR exists', (t) => {
  const f = fixture({
    remotes: [['solo', 'ssh://git@github.com/albertyang514/YourTJ-Hub.git']],
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.equal(result.status, 0, result.stderr);
  assert.ok(readLines(f.gitLog).some((line) => line.includes('fetch --no-tags --quiet https://github.com/YourTongji/YourTJ-Hub.git +refs/heads/dev')));
  assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
});

test('a single canonical remote works with a nonstandard remote name', (t) => {
  const f = fixture({
    remotes: [['primary', 'https://github.com/YourTongji/YourTJ-Hub.git']],
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.equal(result.status, 0, result.stderr);
  assert.ok(readLines(f.gitLog).some((line) => line.includes('fetch --no-tags --quiet primary +refs/heads/dev')));
  assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
});

for (const [description, url] of [
  ['HTTPS', 'https://github.com/YourTongji/YourTJ-Hub.git'],
  ['SSH URL', 'ssh://git@github.com/YourTongji/YourTJ-Hub.git'],
  ['SCP-style SSH with a username', 'git@github.com:YourTongji/YourTJ-Hub.git'],
  ['SCP-style SSH without a username', 'github.com:YourTongji/YourTJ-Hub.git'],
]) {
  test(`reuses a secure ${description} remote for the exact base repository`, (t) => {
    const f = fixture({
      remotes: [['canonical', url]],
      prBase: 'YourTongji/YourTJ-Hub\tdev\n',
    });
    t.after(() => rmSync(f.directory, { recursive: true, force: true }));

    const result = runHook(f);
    assert.equal(result.status, 0, result.stderr);
    const gitCalls = readLines(f.gitLog);
    assert.ok(gitCalls.includes('fetch --no-tags --quiet canonical +refs/heads/dev'));
    assert.ok(!gitCalls.includes('fetch --no-tags --quiet https://github.com/YourTongji/YourTJ-Hub.git +refs/heads/dev'));
    assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
  });
}

for (const [description, url] of [
  ['HTTP transport', 'http://github.com/YourTongji/YourTJ-Hub.git'],
  ['git transport', 'git://github.com/YourTongji/YourTJ-Hub.git'],
  ['wrong repository path', 'https://github.com/YourTongji/Other.git'],
]) {
  test(`uses canonical HTTPS instead of a ${description} remote`, (t) => {
    const f = fixture({
      remotes: [['untrusted', url]],
      prBase: 'YourTongji/YourTJ-Hub\tdev\n',
    });
    t.after(() => rmSync(f.directory, { recursive: true, force: true }));

    const result = runHook(f);
    assert.equal(result.status, 0, result.stderr);
    const gitCalls = readLines(f.gitLog);
    assert.ok(gitCalls.includes('fetch --no-tags --quiet https://github.com/YourTongji/YourTJ-Hub.git +refs/heads/dev'));
    assert.ok(!gitCalls.includes('fetch --no-tags --quiet untrusted +refs/heads/dev'));
    assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
  });
}

for (const [description, url] of [
  ['a mirror URL containing github.com in its path', 'https://mirror.example/github.com/YourTongji/YourTJ-Hub.git'],
  ['a hostname with a github.com suffix', 'https://github.com.evil.example/YourTongji/YourTJ-Hub.git'],
  ['a github.com userinfo value with a different actual host', 'https://github.com@mirror.example/YourTongji/YourTJ-Hub.git'],
  ['a GitHub URL with an extra repository path segment', 'https://github.com/YourTongji/YourTJ-Hub.git/extra'],
]) {
  test(`does not reuse a remote for ${description}`, (t) => {
    const f = fixture({
      remotes: [['mirror', url]],
      prBase: 'YourTongji/YourTJ-Hub\tdev\n',
    });
    t.after(() => rmSync(f.directory, { recursive: true, force: true }));

    const result = runHook(f);
    assert.equal(result.status, 0, result.stderr);
    const gitCalls = readLines(f.gitLog);
    assert.ok(gitCalls.some((line) => line === 'fetch --no-tags --quiet https://github.com/YourTongji/YourTJ-Hub.git +refs/heads/dev'));
    assert.ok(!gitCalls.some((line) => line === 'fetch --no-tags --quiet mirror +refs/heads/dev'));
    assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
  });
}

test('an existing PR targeting a non-dev branch selects that exact repository and branch', (t) => {
  const f = fixture({
    remotes: [['origin', 'git@github.com:AlbertYang514/YourTJ-Hub.git']],
    prBase: 'YourTongji/YourTJ-Hub\trelease\n',
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.equal(result.status, 0, result.stderr);
  assert.ok(readLines(f.gitLog).some((line) => line.includes('fetch --no-tags --quiet https://github.com/YourTongji/YourTJ-Hub.git +refs/heads/release')));
  assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
});

test('an explicit base works before PR creation when the planned target is not dev', (t) => {
  const f = fixture({
    remotes: [['origin', 'git@github.com:AlbertYang514/YourTJ-Hub.git']],
    includeGh: false,
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f, { YOURTJ_LINT_BASE: 'YourTongji/YourTJ-Hub:release' });
  assert.equal(result.status, 0, result.stderr);
  assert.ok(readLines(f.gitLog).some((line) => line.includes('fetch --no-tags --quiet https://github.com/YourTongji/YourTJ-Hub.git +refs/heads/release')));
  assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
});

test('an open PR base takes precedence over an outdated environment override', (t) => {
  const f = fixture({
    remotes: [['canonical', 'https://github.com/YourTongji/YourTJ-Hub.git']],
    prBase: 'YourTongji/YourTJ-Hub\trelease\n',
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f, { YOURTJ_LINT_BASE: 'YourTongji/YourTJ-Hub:dev' });
  assert.equal(result.status, 0, result.stderr);
  assert.ok(readLines(f.gitLog).some((line) => line.includes('canonical +refs/heads/release')));
  assert.ok(!readLines(f.gitLog).some((line) => line.includes('canonical +refs/heads/dev')));
});

test('a failed base fetch fails the hook and never starts lint', (t) => {
  const f = fixture({
    remotes: [['canonical', 'https://github.com/YourTongji/YourTJ-Hub.git']],
    prBase: 'YourTongji/YourTJ-Hub\tdev\n',
    fetchFails: 'all',
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Unable to fetch.*YourTongji\/YourTJ-Hub:dev/);
  assert.deepEqual(readLines(f.lintArgs), []);
});

test('an inaccessible configured SSH remote retries the same repository branch over HTTPS', (t) => {
  const f = fixture({
    remotes: [['upstream', 'git@github.com:YourTongji/YourTJ-Hub.git']],
    prBase: 'YourTongji/YourTJ-Hub\tdev\n',
    fetchFails: 'remote',
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.equal(result.status, 0, result.stderr);
  assert.ok(readLines(f.gitLog).some((line) => line.includes('fetch --no-tags --quiet upstream +refs/heads/dev')));
  assert.ok(readLines(f.gitLog).some((line) => line.includes('fetch --no-tags --quiet https://github.com/YourTongji/YourTJ-Hub.git +refs/heads/dev')));
  assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
});

test('GitHub lookup errors fail closed instead of silently selecting a different base', (t) => {
  const f = fixture({
    remotes: [['canonical', 'https://github.com/YourTongji/YourTJ-Hub.git']],
    ghError: 'authentication failed',
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /authentication failed/);
  assert.deepEqual(readLines(f.lintArgs), []);
});

test('an explicit base is used when GitHub lookup fails unexpectedly', (t) => {
  const f = fixture({
    remotes: [['canonical', 'https://github.com/YourTongji/YourTJ-Hub.git']],
    ghError: 'temporary GitHub API failure',
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f, { YOURTJ_LINT_BASE: 'YourTongji/YourTJ-Hub:dev' });
  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stderr, /using the explicitly configured lint base/);
  assert.deepEqual(readLines(f.lintArgs), ['run', '--new-from-rev=' + freshSha]);
});

test('a discovered PR with an unreadable base does not fall back to an override', (t) => {
  const f = fixture({
    remotes: [['canonical', 'https://github.com/YourTongji/YourTJ-Hub.git']],
    prBase: 'YourTongji/YourTJ-Hub\tdev\n',
    ghApiError: 'REST API unavailable',
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f, { YOURTJ_LINT_BASE: 'YourTongji/YourTJ-Hub:release' });
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /actual PR base could not be verified/);
  assert.deepEqual(readLines(f.lintArgs), []);
});

test('without GitHub CLI or an explicit base, the hook fails rather than guessing', (t) => {
  const f = fixture({
    remotes: [['origin', 'git@github.com:AlbertYang514/YourTJ-Hub.git']],
    includeGh: false,
  });
  t.after(() => rmSync(f.directory, { recursive: true, force: true }));

  const result = runHook(f);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Cannot resolve the actual PR base.*GitHub CLI is unavailable/);
  assert.deepEqual(readLines(f.lintArgs), []);
});

test('real git fetch refreshes a stale base and excludes upstream-only Go files from the PR lint range', (t) => {
  const directory = mkdtempSync(join(tmpdir(), 'yourtj-pre-push-git-'));
  t.after(() => rmSync(directory, { recursive: true, force: true }));
  const canonicalBare = join(directory, 'canonical.git');
  const forkBare = join(directory, 'fork.git');
  const seed = join(directory, 'seed');
  const root = join(directory, 'repo');
  const bin = join(directory, 'bin');
  const lintArgs = join(directory, 'lint.args');
  const lintCwd = join(directory, 'lint.cwd');
  const ghOutput = join(directory, 'gh-output');
  mkdirSync(bin);
  command('git', ['init', '--bare', canonicalBare]);
  command('git', ['init', '--bare', forkBare]);
  command('git', ['init', '--initial-branch=dev', seed]);
  command('git', ['config', 'user.name', 'Test'], { cwd: seed });
  command('git', ['config', 'user.email', 'test@example.com'], { cwd: seed });
  writeFileSync(join(seed, 'README.md'), 'base commit\n');
  command('git', ['add', 'README.md'], { cwd: seed });
  command('git', ['commit', '-m', 'base commit'], { cwd: seed });
  const staleSha = command('git', ['rev-parse', 'HEAD'], { cwd: seed });
  command('git', ['remote', 'add', 'origin', canonicalBare], { cwd: seed });
  command('git', ['push', 'origin', 'dev'], { cwd: seed });
  command('git', ['push', forkBare, 'dev:refs/heads/dev'], { cwd: seed });

  mkdirSync(join(seed, 'apps', 'gooseforum'), { recursive: true });
  writeFileSync(join(seed, 'apps', 'gooseforum', 'legacy.go'), 'package gooseforum\n\nfunc oldLintFinding() { var unused int }\n');
  command('git', ['add', 'apps/gooseforum/legacy.go'], { cwd: seed });
  command('git', ['commit', '-m', 'advance canonical Go baseline'], { cwd: seed });
  const freshSha = command('git', ['rev-parse', 'HEAD'], { cwd: seed });
  command('git', ['push', 'origin', 'dev'], { cwd: seed });

  command('git', ['clone', forkBare, root]);
  command('git', ['config', 'user.name', 'Test'], { cwd: root });
  command('git', ['config', 'user.email', 'test@example.com'], { cwd: root });
  command('git', ['remote', 'add', 'central', 'https://github.com/YourTongji/YourTJ-Hub.git'], { cwd: root });
  command('git', ['config', 'url.' + canonicalBare + '.insteadOf', 'https://github.com/YourTongji/YourTJ-Hub.git'], { cwd: root });
  command('git', ['fetch', 'central', 'dev'], { cwd: root });
  command('git', ['switch', '-c', 'fix/no-go-diff', 'refs/remotes/central/dev'], { cwd: root });
  command('git', ['update-ref', 'refs/remotes/central/dev', staleSha], { cwd: root });
  writeFileSync(join(root, 'README.md'), 'feature documentation only\n');
  command('git', ['add', 'README.md'], { cwd: root });
  command('git', ['commit', '-m', 'documentation-only change'], { cwd: root });

  const staleDiff = command('git', ['diff', '--name-only', 'origin/dev...HEAD'], { cwd: root }).split('\n');
  const actualBaseDiff = command('git', ['diff', '--name-only', freshSha + '...HEAD'], { cwd: root }).split('\n');
  assert.ok(staleDiff.includes('apps/gooseforum/legacy.go'));
  assert.deepEqual(actualBaseDiff, ['README.md']);

  writeFileSync(ghOutput, 'YourTongji/YourTJ-Hub\tdev\n');
  writeFileSync(join(bin, 'gh'), [
    '#!/bin/sh',
    'if [ "$1" = "pr" ]; then printf "123\\thttps://github.com/YourTongji/YourTJ-Hub/pull/123\\n"; exit 0; fi',
    'if [ "$1" = "api" ]; then cat "$GH_OUTPUT"; exit 0; fi',
    'exit 2',
  ].join('\n'));
  chmodSync(join(bin, 'gh'), 0o755);
  writeFileSync(join(bin, 'golangci-lint'), [
    '#!/bin/sh',
    'printf "%s\\n" "$@" > "$LINT_ARGS"',
    'pwd > "$LINT_CWD"',
    'exit 0',
  ].join('\n'));
  chmodSync(join(bin, 'golangci-lint'), 0o755);
  const result = spawnSync('bash', [hook], {
    cwd: root,
    env: { ...process.env, PATH: bin + ':' + process.env.PATH, GH_OUTPUT: ghOutput, LINT_ARGS: lintArgs, LINT_CWD: lintCwd },
    encoding: 'utf8',
  });
  assert.equal(result.status, 0, result.stderr);
  assert.deepEqual(readLines(lintArgs), ['run', '--new-from-rev=' + freshSha]);
  assert.deepEqual(readLines(lintCwd), [realpathSync(join(root, 'apps', 'gooseforum'))]);
  assert.equal(command('git', ['rev-parse', 'FETCH_HEAD'], { cwd: root }), freshSha);
  assert.equal(command('git', ['rev-parse', 'refs/remotes/central/dev'], { cwd: root }), freshSha);
});

import { spawnSync } from 'node:child_process';
import { readFileSync, existsSync } from 'node:fs';
import { pathToFileURL, fileURLToPath } from 'node:url';

const domains = ['governance', 'backend', 'postgres', 'frontend', 'contract', 'mobile'];
export function selectedScopes(paths) {
  const installed = new URL('./select_inputs.py', import.meta.url);
  const selector = existsSync(installed) ? installed : new URL('../../scripts/ci/select_inputs.py', import.meta.url);
  const result = spawnSync('python3', [fileURLToPath(selector), '--paths-json'], {input: JSON.stringify(paths), encoding: 'utf8'});
  if (result.error || result.status !== 0) throw new Error('Cannot establish shared validation scope');
  const plan = JSON.parse(result.stdout);
  return domains.filter(scope => scope === 'governance' || plan[scope]?.run);

}
function run(command, args, cwd = process.cwd(), extraEnv = {}) {
  const result = spawnSync(command, args, { cwd, env: { ...process.env, ...extraEnv }, stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${command} failed (${result.status ?? result.signal})`);
}
export function changedPaths(base) {
  const tracked = spawnSync('git', ['diff', '--name-only', '-z', base, '--'], { encoding: 'utf8' });
  const untracked = spawnSync('git', ['ls-files', '--others', '--exclude-standard', '-z'], { encoding: 'utf8' });
  if (tracked.status !== 0 || untracked.status !== 0) throw new Error('Cannot establish validation scope');
  return (tracked.stdout + untracked.stdout).split('\0').filter(Boolean);
}
function main() {
  const scope = process.argv[2];
  if (!domains.includes(scope)) throw new Error('Unknown validation scope');
  const base = readFileSync('/opt/oryn-validation/base-sha', 'utf8').trim();
  if (!/^[a-f0-9]{40}$/.test(base)) throw new Error('Invalid trusted base SHA');
  if (!selectedScopes(changedPaths(base)).includes(scope)) {
    console.log(`No ${scope} changes relative to the reviewed base.`);
    return;
  }
  const root = process.cwd();
  const backend = `${root}/apps/gooseforum`;
  const frontend = `${backend}/resource`;
  const contract = `${root}/packages/api-contract`;
  const mobile = `${root}/apps/mobile`;
  const install = cwd => {
    if (!existsSync(`${cwd}/node_modules/.modules.yaml`)) run('pnpm', ['install', '--frozen-lockfile'], cwd);
  };
  if (scope === 'governance') run('node', ['scripts/run-gates.mjs']);
  if (scope === 'backend') {
    run('go', ['vet', './...'], backend);
    run('go', ['test', './...'], backend);
    run('go', ['build', '-o', `${process.env.HOME}/yourtj-hub`, '.'], backend);
  }
  if (scope === 'postgres') {
    const env = { YOURTJ_TEST_PG_URL: 'host=127.0.0.1 port=5432 user=postgres password=postgres dbname=postgres sslmode=disable' };
    env.TEST_PG_DSN = env.YOURTJ_TEST_PG_URL;
    run('go', ['test', '-p', '1', '-parallel', '1', './app/...', '-run', 'PostgreSQL|Postgres', '-v'], backend, env);
  }
  if (scope === 'frontend') {
    install(frontend);
    for (const task of ['typecheck', 'test', 'check:i18n', 'build']) run('pnpm', [task], frontend);
  }
  if (scope === 'contract') {
    install(contract);
    run('pnpm', ['check'], contract);
  }
  if (scope === 'mobile') {
    // Flutter writes SDK locks/cache even for analysis; keep those writes in this ephemeral home.
    const sdk = `${process.env.HOME}/flutter`;
    if (!existsSync(sdk)) run('cp', ['-a', '/opt/oryn-flutter', sdk]);
    const env = { PATH: `${sdk}/bin:${process.env.PATH}`, FLUTTER_SUPPRESS_ANALYTICS: 'true', DART_SUPPRESS_ANALYTICS: 'true' };
    run('dart', ['pub', 'get'], mobile, env);
    run('dart', ['run', 'melos', 'bootstrap'], mobile, env);
    for (const task of ['analyze', 'test']) run('dart', ['run', 'melos', 'run', task], mobile, env);
  }
}
if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main();

import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';

const args = new Map();
for (let i = 2; i < process.argv.length; i += 2) {
  args.set(process.argv[i], process.argv[i + 1]);
}

const base = args.get('--base');
const candidate = args.get('--candidate');
const failures = [];
const check = (condition, message) => {
  if (!condition) failures.push(message);
};
const git = (...gitArgs) =>
  execFileSync('git', gitArgs, { encoding: 'utf8' }).trim();
const readAt = (revision, file) =>
  git('show', `${revision}:${file}`);
const sha256 = (value) =>
  createHash('sha256').update(value).digest('hex');

check(Boolean(base), '--base is required');
check(Boolean(candidate), '--candidate is required');
if (!base || !candidate) {
  console.error(`FAIL: ${failures.join('; ')}`);
  process.exit(1);
}

for (const revision of [base, candidate]) {
  try {
    git('cat-file', '-e', `${revision}^{commit}`);
  } catch {
    failures.push(`${revision} is not a commit`);
  }
}

check(git('status', '--porcelain') === '', 'working tree must be clean');

const registry = JSON.parse(
  readAt(candidate, 'docs/waves/wave-registry.json'),
);
const active = registry.waves.filter((wave) => wave.status === 'active');
const mc = registry.waves.find((wave) => wave.id === 'MC-W01');
const tc = registry.waves.find((wave) => wave.id === 'TC-W02');

check(active.length === 1, `expected one active Wave, got ${active.length}`);
check(registry.active_wave === 'MC-W01', 'active_wave must be MC-W01');
check(active[0]?.id === 'MC-W01', 'MC-W01 must be the active record');
check(mc?.document === 'multica-transition/mc-w01/plan-r001.md', 'MC-W01 plan mismatch');
check(mc?.product_code_changes_allowed === true, 'MC-W01 must allow scoped product changes');
check(tc?.status === 'superseded', 'TC-W02 must be superseded');
check(tc?.superseded_by === 'MC-W01', 'TC-W02 superseded_by mismatch');

const planPath = 'docs/waves/multica-transition/mc-w01/plan-r001.md';
const plan = readAt(candidate, planPath);
for (const value of [
  'plan_status: approved',
  'wave_state: active',
  'MC-W01-S01',
  'MC-W01-S05',
  'BACKUP-VERIFIED',
  'product_code_changes_allowed: true',
]) {
  check(plan.includes(value), `MC-W01 plan missing ${value}`);
}

const manifestPath =
  'docs/waves/multica-transition/mc-w01/POLICY_BUNDLE_SHA256SUMS-r001.txt';
const manifest = readAt(candidate, manifestPath);
for (const line of manifest.trim().split('\n')) {
  const match = line.match(/^([0-9a-f]{64})  (.+)$/);
  check(Boolean(match), `invalid manifest line: ${line}`);
  if (!match) continue;
  const [, expected, file] = match;
  const actual = sha256(readAt(candidate, file));
  check(actual === expected, `${file} checksum mismatch`);
}

const freezePath =
  'docs/waves/multica-transition/mc-w01/task-freeze-r001.md';
const freeze = readAt(candidate, freezePath);
for (const value of [
  base,
  git('rev-parse', `${base}^{tree}`),
  'MC-W01-BASELINE-001',
  'codex/backup-goclaw-pre-multica-20260729',
  'codex/multica-six-domain-baseline',
]) {
  check(freeze.includes(value), `task freeze missing ${value}`);
}

const changed = git('diff', '--name-only', `${base}...${candidate}`)
  .split('\n')
  .filter(Boolean);
check(changed.length > 0, 'base-to-candidate diff must be non-empty');
check(
  changed.every((file) => file.startsWith('docs/waves/')),
  `freeze candidate changed outside docs/waves: ${changed.join(', ')}`,
);

try {
  execFileSync('git', ['diff', '--check', `${base}...${candidate}`], {
    stdio: 'pipe',
  });
} catch {
  failures.push('git diff --check failed');
}

if (failures.length > 0) {
  console.error(`FAIL:\n- ${failures.join('\n- ')}`);
  process.exit(1);
}

console.log(
  `PASS: MC-W01 active; ${changed.length} freeze files changed; policy and tuple verified`,
);

#!/usr/bin/env node

import { execFileSync } from 'node:child_process';
import { posix as path } from 'node:path';

const root = execFileSync('git', ['rev-parse', '--show-toplevel'], {
  encoding: 'utf8',
}).trim();
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

function git(args, options = {}) {
  return execFileSync('git', args, {
    cwd: root,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'pipe'],
    ...options,
  });
}

function arg(name) {
  const index = process.argv.indexOf(name);
  return index >= 0 ? process.argv[index + 1] : '';
}

const baseInput = arg('--base');
const candidateInput = arg('--candidate');
check(Boolean(baseInput), '--base <frozen-base-commit> is required');
check(Boolean(candidateInput), '--candidate <candidate-commit> is required');

let base = '';
let candidate = '';
try {
  base = git(['rev-parse', '--verify', `${baseInput}^{commit}`]).trim();
} catch {
  failures.push(`base is not a commit: ${baseInput}`);
}
try {
  candidate = git(['rev-parse', '--verify', `${candidateInput}^{commit}`]).trim();
} catch {
  failures.push(`candidate is not a commit: ${candidateInput}`);
}

if (base && candidate) {
  try {
    git(['merge-base', '--is-ancestor', base, candidate]);
  } catch {
    failures.push('candidate does not descend from frozen base');
  }
}

const dirty = git(['status', '--porcelain=v1', '--untracked-files=all']).trim();
check(dirty === '', 'working tree must be clean for exact candidate verification');

function existsAt(relative) {
  try {
    git(['cat-file', '-e', `${candidate}:${relative}`]);
    return true;
  } catch {
    return false;
  }
}

function readAt(relative) {
  return git(['show', `${candidate}:${relative}`]);
}

function frontmatterValue(content, key) {
  const match = content.match(new RegExp(`^${key}:\\s*(.+)\\s*$`, 'm'));
  return match ? match[1].trim().replace(/^["']|["']$/g, '') : '';
}

let changed = [];
if (base && candidate) {
  changed = git(['diff', '--name-only', `${base}...${candidate}`])
    .trim()
    .split('\n')
    .filter(Boolean);
}
check(changed.length > 0, 'base→candidate diff must be non-empty');
for (const relative of changed) {
  check(relative.startsWith('docs/waves/'), `change outside docs/waves/**: ${relative}`);
}

let registry = { waves: [] };
if (candidate && existsAt('docs/waves/wave-registry.json')) {
  try {
    registry = JSON.parse(readAt('docs/waves/wave-registry.json'));
  } catch (error) {
    failures.push(`wave-registry.json is not valid JSON: ${error.message}`);
  }
} else {
  failures.push('candidate does not contain docs/waves/wave-registry.json');
}

const active = registry.waves.filter((wave) => wave.status === 'active');
check(active.length === 1, `expected exactly one active Wave, got ${active.length}`);
check(active[0]?.id === registry.active_wave, 'active_wave does not match active record');
check(registry.active_wave === 'TC-W02', 'TC-W02 must be the active Wave');

const ids = new Set();
for (const wave of registry.waves) {
  check(!ids.has(wave.id), `duplicate Wave ID ${wave.id}`);
  ids.add(wave.id);
  const relative = path.join('docs/waves', wave.document);
  check(existsAt(relative), `${wave.id} document does not exist: ${wave.document}`);
  if (!existsAt(relative)) continue;
  const content = readAt(relative);
  check(frontmatterValue(content, 'wave_id') === wave.id, `${wave.id} document wave_id mismatch`);
  if (['active', 'planned', 'proposed'].includes(wave.status)) {
    check(
      frontmatterValue(content, 'wave_state') === wave.status,
      `${wave.id} document wave_state does not match ${wave.status}`,
    );
  }
  for (const dependency of wave.depends_on ?? []) {
    check(registry.waves.some((item) => item.id === dependency), `${wave.id} missing dependency ${dependency}`);
  }
}

const tcw02 = registry.waves.find((wave) => wave.id === 'TC-W02');
check(tcw02?.document === 'team-runtime/tc-w02/plan-r002.md', 'TC-W02 must point to r002');
check(tcw02?.product_code_changes_allowed === false, 'TC-W02 must forbid product code changes');
check(
  JSON.stringify(tcw02?.allowed_change_scope) === JSON.stringify(['docs/waves/**']),
  'TC-W02 scope must be exactly docs/waves/**',
);

const rn = registry.waves.find((wave) => wave.id === 'RN-W01');
check(rn?.status === 'superseded', 'RN-W01 must be superseded');
check(rn?.superseded_by === 'TC-W02', 'RN-W01 must identify TC-W02 as replacement');
check(Boolean(rn?.superseded_reason?.includes('unfinished history')), 'RN-W01 must retain an unfinished reason');
check(rn?.product_code_changes_allowed === false, 'RN-W01 must not allow product changes');

const intWave = registry.waves.find((wave) => wave.id === 'INT-W01');
check(intWave?.status === 'superseded', 'INT-W01 must be superseded');
check(
  ['TC-W02', 'TC-W03', 'TC-W04', 'TC-W05', 'TC-W06'].every(
    (id) => intWave?.superseded_by?.includes(id),
  ),
  'INT-W01 replacement route is incomplete',
);

for (const id of ['TC-W03', 'TC-W04', 'TC-W05', 'TC-W06']) {
  const wave = registry.waves.find((item) => item.id === id);
  check(wave?.status === 'proposed', `${id} must remain proposed`);
  check(wave?.product_code_changes_allowed === false, `${id} must forbid product changes`);
  const content = existsAt(path.join('docs/waves', wave?.document ?? ''))
    ? readAt(path.join('docs/waves', wave.document))
    : '';
  check(/^approved_by:\s*$/m.test(content), `${id} draft missing approved_by`);
  for (const heading of [
    '## 目标',
    '## 权威输入',
    '## 入口门禁',
    '## 范围',
    '### 包含',
    '### 不包含',
    '## 问题与事实',
    '## 影响分析',
    '## 分步计划',
    '## 验证与证据计划',
    '## 风险与回滚',
    '## 退出门禁',
    '## 决策记录',
    '## Plan revision',
  ]) {
    check(content.includes(heading), `${id} draft missing ${heading}`);
  }
}

const successorDecisionMap = {
  'TC-W03': 'TC-DEC-005',
  'TC-W04': 'TC-DEC-004',
  'TC-W05': 'TC-DEC-006',
  'TC-W06': 'TC-DEC-007',
};
for (const [id, decisionId] of Object.entries(successorDecisionMap)) {
  const wave = registry.waves.find((item) => item.id === id);
  const content = readAt(path.join('docs/waves', wave.document));
  check(content.includes(`\`${decisionId}\``), `${id} must reference ${decisionId}`);
}
const tcw03Draft = readAt('docs/waves/team-runtime/tc-w03/plan-r001.md');
check(
  tcw03Draft.includes('non-executable shadow') &&
    tcw03Draft.includes('旧 Catalog 继续作为唯一 runtime writer'),
  'TC-W03 must not claim runtime authority cutover',
);
const decisionLog = readAt('docs/waves/decision-log.md');
check(
  decisionLog.includes('TC-DEC-007：客户端仅为投影，知识迁移分阶段且切换独立授权'),
  'TC-DEC-007 projection/migration decision missing',
);

const rel = registry.waves.find((wave) => wave.id === 'REL-W01');
check(JSON.stringify(rel?.depends_on) === JSON.stringify(['TC-W06']), 'REL-W01 must depend on TC-W06');
check(rel?.document === 'team-runtime/rel-w01/plan-r002.md', 'REL-W01 must point to r002');
check(rel?.product_code_changes_allowed === false, 'REL-W01 r002 must forbid product changes');

const planR1 = readAt('docs/waves/team-runtime/tc-w02/plan-r001.md');
const planR2 = readAt('docs/waves/team-runtime/tc-w02/plan-r002.md');
for (const heading of [
  '## 目标',
  '## 权威输入',
  '## 范围',
  '## 核心不变量',
  '## 分步计划',
  '## 验证与证据计划',
  '## 风险与回滚',
  '## 停止条件',
  '## 退出门禁',
]) {
  check(planR1.includes(heading), `TC-W02 r001 missing inherited ${heading}`);
}
check(planR2.includes('## Review findings 与必须修复'), 'TC-W02 r002 missing review findings');
check(planR2.includes('--base <frozen-base-commit>'), 'TC-W02 r002 missing exact validator base');
check(planR2.includes('--candidate <candidate-commit>'), 'TC-W02 r002 missing exact validator candidate');

const contracts = readAt('docs/waves/team-runtime/tc-w02/target-contracts.md');
for (const phrase of [
  'global defaults',
  'global mandatory constraints',
  'mandatory_set_epoch',
  'global_policy_approve',
  'global_memory_approve',
  'RFC 8785',
  'Stable citation v1',
  'execution_pack_sha256',
  'typed opaque ID',
  'monotonic',
  '## Knowledge 合同',
  '## Context Compiler 合同',
  '## Team Control MCP 合同',
  '## Evidence 与反馈闭环',
]) {
  check(contracts.includes(phrase), `target contracts missing ${phrase}`);
}

const matrix = readAt('docs/waves/team-runtime/tc-w02/current-state-responsibility-matrix.md');
for (const phrase of [
  'KnowledgeSource',
  'PolicyBundle',
  'ContextBundle',
  'ExecutionPack',
  'EvidenceBundle',
  'Memory Catalog',
  '## 当前 RPC/命令到目标合同的逐项映射',
  '`memory.catalog.candidate.approve`',
  '`knowledge.source.delete`',
  '`context.compile`',
  '`propose_project_memory`',
]) {
  check(matrix.includes(phrase), `current-state matrix missing ${phrase}`);
}

const migration = readAt('docs/waves/team-runtime/tc-w02/migration-and-wave-roadmap.md');
for (const phrase of [
  '`project_id="*"`',
  '一律拒绝',
  '### TC-W03',
  '### TC-W04',
  '### TC-W05',
  '### TC-W06',
  '`workstation/types.go`',
  'shadow import',
  'read cutover',
]) {
  check(migration.includes(phrase), `migration roadmap missing ${phrase}`);
}

let journalDiff = '';
if (base && candidate) {
  journalDiff = git([
    'diff',
    '--unified=0',
    `${base}...${candidate}`,
    '--',
    'docs/waves/**/journal.md',
  ]);
}
for (const line of journalDiff.split('\n')) {
  if (line.startsWith('-') && !line.startsWith('---')) {
    failures.push(`journal is not append-only: ${line.slice(0, 120)}`);
  }
}

const markdownFiles = changed.filter((relative) => relative.endsWith('.md') && existsAt(relative));
const linkPattern = /\[[^\]]+\]\(([^)]+)\)/g;
for (const relative of markdownFiles) {
  const content = readAt(relative);
  for (const match of content.matchAll(linkPattern)) {
    const target = match[1].trim();
    if (!target || target.startsWith('#') || /^[a-z]+:\/\//i.test(target)) continue;
    const withoutAnchor = target.split('#')[0];
    if (!withoutAnchor) continue;
    const resolved = path.normalize(path.join(path.dirname(relative), withoutAnchor));
    check(existsAt(resolved), `${relative} has broken link ${target}`);
  }
}

if (failures.length > 0) {
  for (const failure of failures) process.stderr.write(`FAIL: ${failure}\n`);
  process.exit(1);
}

process.stdout.write(
  `PASS: base ${base.slice(0, 12)} → candidate ${candidate.slice(0, 12)}; ` +
  `${registry.waves.length} Waves; one active (${registry.active_wave}); ` +
  `${changed.length} changed files all under docs/waves/**\n`,
);

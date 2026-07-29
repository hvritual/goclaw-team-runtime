#!/usr/bin/env node

import { execFileSync } from 'node:child_process';
import { existsSync, readFileSync } from 'node:fs';
import { dirname, join, normalize } from 'node:path';

const root = execFileSync('git', ['rev-parse', '--show-toplevel'], {
  encoding: 'utf8',
}).trim();
const registryPath = join(root, 'docs/waves/wave-registry.json');
const registry = JSON.parse(readFileSync(registryPath, 'utf8'));
const failures = [];

function check(condition, message) {
  if (!condition) failures.push(message);
}

function read(relative) {
  return readFileSync(join(root, relative), 'utf8');
}

function frontmatterValue(content, key) {
  const match = content.match(new RegExp(`^${key}:\\s*(.+)\\s*$`, 'm'));
  return match ? match[1].trim().replace(/^["']|["']$/g, '') : '';
}

const active = registry.waves.filter((wave) => wave.status === 'active');
check(active.length === 1, `expected exactly one active Wave, got ${active.length}`);
check(active[0]?.id === registry.active_wave, 'active_wave does not match active record');
check(registry.active_wave === 'TC-W02', 'TC-W02 must be the active Wave');

const ids = new Set();
for (const wave of registry.waves) {
  check(!ids.has(wave.id), `duplicate Wave ID ${wave.id}`);
  ids.add(wave.id);
  const documentPath = join(root, 'docs/waves', wave.document);
  check(existsSync(documentPath), `${wave.id} document does not exist: ${wave.document}`);
  if (!existsSync(documentPath)) continue;
  const content = readFileSync(documentPath, 'utf8');
  check(frontmatterValue(content, 'wave_id') === wave.id, `${wave.id} document wave_id mismatch`);
  if (['active', 'planned', 'proposed'].includes(wave.status)) {
    check(
      frontmatterValue(content, 'wave_state') === wave.status,
      `${wave.id} document wave_state does not match ${wave.status}`,
    );
  }
  for (const dependency of wave.depends_on ?? []) {
    check(registry.waves.some((candidate) => candidate.id === dependency), `${wave.id} missing dependency ${dependency}`);
  }
}

const tcw02 = registry.waves.find((wave) => wave.id === 'TC-W02');
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
  const wave = registry.waves.find((candidate) => candidate.id === id);
  check(wave?.status === 'proposed', `${id} must remain proposed`);
  check(wave?.product_code_changes_allowed === false, `${id} must forbid product changes`);
}

const rel = registry.waves.find((wave) => wave.id === 'REL-W01');
check(
  JSON.stringify(rel?.depends_on) === JSON.stringify(['TC-W06']),
  'REL-W01 must depend on TC-W06',
);
check(rel?.document === 'team-runtime/rel-w01/plan-r002.md', 'REL-W01 must point to r002');
check(rel?.product_code_changes_allowed === false, 'REL-W01 r002 must forbid product changes');

const plan = read('docs/waves/team-runtime/tc-w02/plan-r001.md');
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
  check(plan.includes(heading), `TC-W02 plan missing ${heading}`);
}

const contracts = read('docs/waves/team-runtime/tc-w02/target-contracts.md');
for (const phrase of [
  'global defaults',
  'global mandatory constraints',
  '## Knowledge 合同',
  '## Context Compiler 合同',
  '## Team Control MCP 合同',
  '## Evidence 与反馈闭环',
  'memory_approve',
]) {
  check(contracts.includes(phrase), `target contracts missing ${phrase}`);
}

const matrix = read('docs/waves/team-runtime/tc-w02/current-state-responsibility-matrix.md');
for (const phrase of [
  'KnowledgeSource',
  'PolicyBundle',
  'ContextBundle',
  'ExecutionPack',
  'EvidenceBundle',
  'Memory Catalog',
  'Gateway',
  'Team Web',
  'CLI',
]) {
  check(matrix.includes(phrase), `current-state matrix missing ${phrase}`);
}

const migration = read('docs/waves/team-runtime/tc-w02/migration-and-wave-roadmap.md');
for (const phrase of [
  '`project_id="*"`',
  '### TC-W03',
  '### TC-W04',
  '### TC-W05',
  '### TC-W06',
  'inventory',
  'shadow import',
  'read cutover',
]) {
  check(migration.includes(phrase), `migration roadmap missing ${phrase}`);
}

const statusLines = execFileSync(
  'git',
  ['status', '--porcelain=v1', '--untracked-files=all'],
  {
    cwd: root,
    encoding: 'utf8',
  },
).split('\n').filter(Boolean);
const changed = statusLines.map((line) => {
  const path = line.slice(3);
  const renameTarget = path.includes(' -> ') ? path.split(' -> ').at(-1) : path;
  return renameTarget.replace(/^"|"$/g, '');
});
for (const path of changed) {
  check(path.startsWith('docs/waves/'), `change outside docs/waves/**: ${path}`);
}

const journalDiff = execFileSync(
  'git',
  ['diff', '--unified=0', 'HEAD', '--', 'docs/waves/**/journal.md'],
  { cwd: root, encoding: 'utf8' },
);
for (const line of journalDiff.split('\n')) {
  if (line.startsWith('-') && !line.startsWith('---')) {
    failures.push(`journal is not append-only: ${line.slice(0, 120)}`);
  }
}

const markdownFiles = changed.filter((path) => path.endsWith('.md'));
const linkPattern = /\[[^\]]+\]\(([^)]+)\)/g;
for (const relative of markdownFiles) {
  const content = read(relative);
  for (const match of content.matchAll(linkPattern)) {
    const target = match[1].trim();
    if (!target || target.startsWith('#') || /^[a-z]+:\/\//i.test(target)) continue;
    const withoutAnchor = target.split('#')[0];
    if (!withoutAnchor) continue;
    const resolved = normalize(join(root, dirname(relative), withoutAnchor));
    check(existsSync(resolved), `${relative} has broken link ${target}`);
  }
}

if (failures.length > 0) {
  for (const failure of failures) process.stderr.write(`FAIL: ${failure}\n`);
  process.exit(1);
}

process.stdout.write(
  `PASS: ${registry.waves.length} Waves, one active (${registry.active_wave}), ` +
  `${changed.length} changed files all under docs/waves/**\n`,
);

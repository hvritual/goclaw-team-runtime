import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync(new URL('../src/team/TeamPage.tsx', import.meta.url), 'utf8');

test('team projection loads and renders the central control summary', () => {
  assert.match(source, /client\.rpc<TeamControlSummary>\('control\.summary'/);
  assert.match(source, /中央上下文治理/);
  assert.match(source, /Token 预算/);
  assert.match(source, /尚未登记中央预算、知识、Skill 或 Runner release/);
});

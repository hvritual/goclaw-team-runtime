import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { createServer } from 'vite';

const uiRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

async function load() {
  const vite = await createServer({
    root: uiRoot,
    configFile: false,
    envFile: false,
    appType: 'custom',
    logLevel: 'silent',
    server: { middlewareMode: true },
  });
  try {
    return await vite.ssrLoadModule('/src/team/control-summary-state.ts');
  } finally {
    await vite.close();
  }
}

const populated = {
  project_id: 'alpha',
  budget_count: 1,
  limit_tokens: 100,
  used_tokens: 10,
  knowledge_count: 1,
  approved_knowledge: 1,
  skill_count: 1,
  approved_skills: 1,
  runner_release_count: 1,
  context_bundle_count: 1,
};

test('central control projection exposes executable loading and empty states', async () => {
  const { controlSummaryState } = await load();
  assert.equal(controlSummaryState({
    data: null,
    error: null,
    loading: true,
  }).kind, 'loading');
  assert.equal(controlSummaryState({
    data: Object.fromEntries(
      Object.entries(populated).map(([key, value]) =>
        [key, typeof value === 'number' ? 0 : value]),
    ),
    error: null,
    loading: false,
  }).kind, 'empty');
});

test('central control projection distinguishes denied, error, and ready', async () => {
  const { controlSummaryState } = await load();
  assert.equal(controlSummaryState({
    data: null,
    error: new Error('403 forbidden'),
    loading: false,
  }).kind, 'denied');
  assert.equal(controlSummaryState({
    data: null,
    error: new Error('gateway unavailable'),
    loading: false,
  }).kind, 'error');
  assert.equal(controlSummaryState({
    data: populated,
    error: null,
    loading: false,
  }).kind, 'ready');
});

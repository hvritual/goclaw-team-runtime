import assert from 'node:assert/strict';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { createServer } from 'vite';

const uiRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

async function load(modulePath) {
  const vite = await createServer({
    root: uiRoot,
    configFile: false,
    envFile: false,
    appType: 'custom',
    logLevel: 'silent',
    server: { middlewareMode: true },
  });
  try {
    return await vite.ssrLoadModule(modulePath);
  } finally {
    await vite.close();
  }
}

test('chat reducer isolates scope and rejects duplicate or reordered run sequences', async () => {
  const {
    applyChatEvent,
    chatScope,
    emptyChatState,
  } = await load('/src/team/chat-state.ts');
  const alpha = chatScope('alpha', 'inbox');
  let state = emptyChatState(alpha);
  state = applyChatEvent(state, alpha, {
    run_id: 'run-1',
    seq: 1,
    state: 'delta',
    content: 'Hello',
  });
  const duplicate = applyChatEvent(state, alpha, {
    run_id: 'run-1',
    seq: 1,
    state: 'delta',
    content: ' duplicated',
  });
  assert.equal(duplicate, state);
  const reordered = applyChatEvent(state, alpha, {
    run_id: 'run-1',
    seq: 0,
    state: 'delta',
    content: ' reordered',
  });
  assert.equal(reordered, state);
  assert.equal(state.messages[0].content, 'Hello');

  const beta = chatScope('beta', 'inbox');
  const switched = applyChatEvent(state, beta, {
    run_id: 'run-2',
    seq: 1,
    state: 'final',
    content: 'Beta only',
  });
  assert.deepEqual(switched.messages.map((message) => message.content), ['Beta only']);
});

test('chat event scope matching fails closed for missing or mismatched scope', async () => {
  const { chatEventMatches } = await load('/src/team/chat-state.ts');
  const scoped = {
    run_id: 'run-1',
    seq: 1,
    state: 'delta',
    project_id: 'alpha',
    topic_id: 'inbox',
  };

  assert.equal(chatEventMatches(scoped, 'alpha', 'inbox'), true);
  assert.equal(chatEventMatches({ ...scoped, project_id: undefined }, 'alpha', 'inbox'), false);
  assert.equal(chatEventMatches({ ...scoped, topic_id: undefined }, 'alpha', 'inbox'), false);
  assert.equal(chatEventMatches({ ...scoped, project_id: 'beta' }, 'alpha', 'inbox'), false);
  assert.equal(chatEventMatches({ ...scoped, topic_id: 'release' }, 'alpha', 'inbox'), false);
});

test('chat history replaces matching optimistic messages without dropping unmatched live runs', async () => {
  const {
    appendTransientMessage,
    emptyChatState,
    mergeChatHistory,
  } = await load('/src/team/chat-state.ts');
  const scope = 'alpha\u0000inbox';
  let state = emptyChatState(scope);
  state = appendTransientMessage(state, scope, {
    id: 'local-user',
    role: 'user',
    content: 'same question',
  });
  state = appendTransientMessage(state, scope, {
    id: 'live-assistant',
    role: 'assistant',
    content: 'still streaming',
    pending: true,
  });
  state = mergeChatHistory(state, scope, [{
    id: 'history-user',
    role: 'user',
    content: 'same question',
  }]);
  assert.deepEqual(
    state.messages.map((message) => message.id),
    ['history-user', 'live-assistant'],
  );
});

test('canonical issue and work item transition tables expose only server-valid moves', async () => {
  const {
    nextIssueStatuses,
    nextWorkItemStatuses,
  } = await load('/src/team/workflow-state.ts');
  assert.deepEqual(nextIssueStatuses('new'), ['triaged', 'cancelled']);
  assert.deepEqual(nextIssueStatuses('verifying'), ['resolved', 'in_progress', 'blocked']);
  assert.deepEqual(nextIssueStatuses('closed'), ['reopened']);
  assert.deepEqual(nextIssueStatuses('cancelled'), []);
  assert.deepEqual(nextWorkItemStatuses('ready'), ['in_progress', 'blocked', 'cancelled']);
  assert.deepEqual(nextWorkItemStatuses('verifying'), ['done', 'in_progress', 'blocked']);
  assert.deepEqual(nextWorkItemStatuses('done'), []);
});

test('dependency identity comparison detects a scope change synchronously', async () => {
  const { sameDependencies } = await load('/src/team/use-data.ts');
  const loader = () => undefined;
  assert.equal(sameDependencies([loader, 'alpha'], [loader, 'alpha']), true);
  assert.equal(sameDependencies([loader, 'beta'], [loader, 'alpha']), false);
  assert.equal(sameDependencies([loader], [loader, 'alpha']), false);
});

test('TeamClient publishes 401 expiry and closes event transport', {
  concurrency: false,
}, async () => {
  const originals = {
    fetch: globalThis.fetch,
    WebSocket: globalThis.WebSocket,
    window: globalThis.window,
  };
  let closeCount = 0;
  let requestCount = 0;
  class FakeWebSocket {
    static CONNECTING = 0;
    static OPEN = 1;
    static CLOSING = 2;
    static CLOSED = 3;

    constructor() {
      this.readyState = FakeWebSocket.CONNECTING;
    }

    close() {
      closeCount += 1;
      this.readyState = FakeWebSocket.CLOSED;
    }
  }

  try {
    globalThis.window = {
      location: {
        protocol: 'http:',
        host: '127.0.0.1:5173',
      },
      setTimeout,
      clearTimeout,
    };
    globalThis.WebSocket = FakeWebSocket;
    globalThis.fetch = async () => {
      requestCount += 1;
      if (requestCount === 1) {
        return new Response(JSON.stringify({
          principal_id: 'pilot-alice',
          csrf_token: 'csrf',
          expires_at: '2026-07-27T12:00:00Z',
        }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      return new Response(null, { status: 401 });
    };

    const { TeamClient } = await load('/src/team/client.ts');
    const client = new TeamClient();
    const sessions = [];
    client.onSession((session) => sessions.push(session?.principal_id ?? null));
    await client.login({ gatewayToken: 'outer', userToken: 'personal' });
    await assert.rejects(
      client.rpc('project.list'),
      /团队会话已失效/,
    );
    assert.deepEqual(sessions, ['pilot-alice', null]);
    assert.equal(client.activeSession, null);
    assert.equal(closeCount, 1);
  } finally {
    globalThis.fetch = originals.fetch;
    globalThis.WebSocket = originals.WebSocket;
    if (originals.window === undefined) delete globalThis.window;
    else globalThis.window = originals.window;
  }
});

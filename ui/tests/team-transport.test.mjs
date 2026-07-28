import assert from 'node:assert/strict';
import http from 'node:http';
import net from 'node:net';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { createServer, loadConfigFromFile } from 'vite';

const uiRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');

function listen(server) {
  return new Promise((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => {
      server.off('error', reject);
      const address = server.address();
      if (!address || typeof address === 'string') {
        reject(new Error('loopback server did not expose a TCP port'));
        return;
      }
      resolve(address.port);
    });
  });
}

function closeServer(server) {
  return new Promise((resolve, reject) => {
    if (!server.listening) {
      resolve();
      return;
    }
    server.close((error) => error ? reject(error) : resolve());
  });
}

function sameOriginHost(request) {
  const origin = request.headers.origin;
  if (!origin || !request.headers.host) return false;
  try {
    return new URL(origin).host.toLowerCase() === request.headers.host.toLowerCase();
  } catch {
    return false;
  }
}

function requestUpgrade(port, pageHost) {
  return new Promise((resolve, reject) => {
    const socket = net.connect({ host: '127.0.0.1', port });
    let response = '';
    const timeout = setTimeout(() => {
      socket.destroy();
      reject(new Error('WebSocket upgrade probe timed out'));
    }, 5_000);

    socket.once('error', (error) => {
      clearTimeout(timeout);
      reject(error);
    });
    socket.once('connect', () => {
      socket.write([
        'GET /ws HTTP/1.1',
        `Host: ${pageHost}`,
        'Connection: Upgrade',
        'Upgrade: websocket',
        'Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==',
        'Sec-WebSocket-Version: 13',
        `Origin: http://${pageHost}`,
        '',
        '',
      ].join('\r\n'));
    });
    socket.on('data', (chunk) => {
      response += chunk.toString('utf8');
      if (!response.includes('\r\n\r\n')) return;
      clearTimeout(timeout);
      const status = Number(response.match(/^HTTP\/1\.1\s+(\d{3})/)?.[1] ?? 0);
      socket.destroy();
      resolve(status);
    });
  });
}

test('TeamClient uses the same-origin WebSocket without token-bearing protocols', {
  concurrency: false,
}, async () => {
  const vite = await createServer({
    root: uiRoot,
    configFile: false,
    envFile: false,
    appType: 'custom',
    logLevel: 'silent',
    server: { middlewareMode: true },
  });
  const originals = {
    fetch: globalThis.fetch,
    WebSocket: globalThis.WebSocket,
    window: globalThis.window,
  };
  let capturedURL = '';
  let capturedProtocols = [];

  class FakeWebSocket {
    static CONNECTING = 0;
    static OPEN = 1;
    static CLOSING = 2;
    static CLOSED = 3;

    constructor(url, protocols) {
      capturedURL = String(url);
      capturedProtocols = [...protocols];
      this.readyState = FakeWebSocket.CONNECTING;
    }

    close() {
      this.readyState = FakeWebSocket.CLOSED;
    }
  }

  try {
    globalThis.window = {
      location: {
        protocol: 'http:',
        hostname: '127.0.0.1',
        host: '127.0.0.1:5173',
      },
      setTimeout,
      clearTimeout,
    };
    globalThis.WebSocket = FakeWebSocket;
    globalThis.fetch = async () => new Response(JSON.stringify({
      principal_id: 'test-principal',
      csrf_token: 'test-csrf',
      expires_at: '2026-07-26T12:00:00Z',
    }), {
      status: 201,
      headers: { 'Content-Type': 'application/json' },
    });

    const { TeamClient } = await vite.ssrLoadModule('/src/team/client.ts');
    const client = new TeamClient();
    await client.login({
      gatewayToken: 'gateway-secret-sentinel',
      userToken: 'team-secret-sentinel',
    });

    assert.equal(capturedURL, 'ws://127.0.0.1:5173/ws');
    assert.deepEqual(capturedProtocols, ['goclaw.v1']);
    assert.doesNotMatch(
      `${capturedURL} ${capturedProtocols.join(' ')}`,
      /gateway-secret-sentinel|team-secret-sentinel/,
    );
    client.disconnectEvents();
  } finally {
    globalThis.fetch = originals.fetch;
    globalThis.WebSocket = originals.WebSocket;
    if (originals.window === undefined) delete globalThis.window;
    else globalThis.window = originals.window;
    await vite.close();
  }
});
test('Vite auth and WebSocket proxies preserve the browser origin host', {
  concurrency: false,
}, async () => {
  const loaded = await loadConfigFromFile(
    { command: 'serve', mode: 'development' },
    path.join(uiRoot, 'vite.config.ts'),
    uiRoot,
  );
  assert.ok(loaded, 'vite.config.ts must be loadable');
  const configuredProxy = loaded.config.server?.proxy;
  assert.ok(configuredProxy && !Array.isArray(configuredProxy));
  const authConfig = configuredProxy['/auth'];
  const wsConfig = configuredProxy['/ws'];
  assert.equal(typeof authConfig, 'object');
  assert.equal(typeof wsConfig, 'object');

  const observations = [];
  const authTarget = http.createServer((request, response) => {
    const observation = {
      method: request.method,
      path: request.url,
      host: request.headers.host,
      origin: request.headers.origin,
    };
    observations.push({ kind: 'auth', ...observation });
    response.writeHead(sameOriginHost(request) ? 204 : 403);
    response.end();
  });
  const wsTarget = http.createServer();
  wsTarget.on('upgrade', (request, socket) => {
    const observation = {
      method: request.method,
      path: request.url,
      host: request.headers.host,
      origin: request.headers.origin,
    };
    observations.push({ kind: 'ws', ...observation });
    if (!sameOriginHost(request)) {
      socket.end('HTTP/1.1 403 Forbidden\r\nConnection: close\r\n\r\n');
      return;
    }
    socket.end([
      'HTTP/1.1 101 Switching Protocols',
      'Connection: Upgrade',
      'Upgrade: websocket',
      'Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=',
      '',
      '',
    ].join('\r\n'));
  });

  let vite;
  try {
    const authPort = await listen(authTarget);
    const wsPort = await listen(wsTarget);
    vite = await createServer({
      root: uiRoot,
      configFile: false,
      envFile: false,
      appType: 'custom',
      logLevel: 'silent',
      server: {
        host: '127.0.0.1',
        port: 0,
        strictPort: false,
        proxy: {
          '/auth': {
            ...authConfig,
            target: `http://127.0.0.1:${authPort}`,
          },
          '/ws': {
            ...wsConfig,
            target: `ws://127.0.0.1:${wsPort}`,
          },
        },
      },
    });
    await vite.listen();
    const address = vite.httpServer?.address();
    assert.ok(address && typeof address !== 'string');
    const pageHost = `127.0.0.1:${address.port}`;
    const origin = `http://${pageHost}`;

    const authResponse = await fetch(`${origin}/auth/session`, {
      method: 'POST',
      headers: { Origin: origin },
    });
    const wsStatus = await requestUpgrade(address.port, pageHost);
    const authObservation = observations.find((entry) => entry.kind === 'auth');
    const wsObservation = observations.find((entry) => entry.kind === 'ws');

    assert.equal(authResponse.status, 204, JSON.stringify(authObservation));
    assert.equal(authObservation?.host, new URL(authObservation?.origin ?? '').host);
    assert.equal(wsStatus, 101, JSON.stringify(wsObservation));
    assert.equal(wsObservation?.host, new URL(wsObservation?.origin ?? '').host);
    assert.equal(wsConfig.changeOrigin, false);
  } finally {
    if (vite) await vite.close();
    await Promise.all([closeServer(authTarget), closeServer(wsTarget)]);
  }
});

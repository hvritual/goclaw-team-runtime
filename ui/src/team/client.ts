import { LoginInput, RPCMessage, WebSession } from './types';

type NotificationListener = (payload: unknown) => void;
type ConnectionListener = (state: 'connecting' | 'connected' | 'disconnected' | 'error') => void;
type SessionListener = (session: WebSession | null) => void;

const CSRF_HEADER = 'X-GoClaw-CSRF';

function websocketURL(): string {
  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${scheme}//${window.location.host}/ws`;
}

export class TeamClient {
  private session: WebSession | null = null;
  private socket: WebSocket | null = null;
  private sequence = 0;
  private pending = new Map<string, {
    resolve: (value: unknown) => void;
    reject: (error: Error) => void;
    timeout: number;
  }>();
  private notifications = new Map<string, Set<NotificationListener>>();
  private connectionListeners = new Set<ConnectionListener>();
  private sessionListeners = new Set<SessionListener>();
  private reconnectTimer: number | null = null;
  private reconnectAttempt = 0;
  private manualClose = false;

  get activeSession(): WebSession | null {
    return this.session;
  }

  async resume(): Promise<WebSession | null> {
    try {
      const response = await fetch('/auth/session', {
        credentials: 'same-origin',
        headers: { Accept: 'application/json' },
      });
      if (response.status === 401) {
        this.expireSession();
        return null;
      }
      if (!response.ok) throw new Error(`恢复会话失败（${response.status}）`);
      this.setSession(await response.json() as WebSession);
      this.connectEvents();
      return this.session;
    } catch (error) {
      this.expireSession();
      throw error;
    }
  }

  async login(input: LoginInput): Promise<WebSession> {
    const response = await fetch('/auth/session', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify({
        gateway_token: input.gatewayToken,
        user_token: input.userToken,
      }),
    });
    if (!response.ok) {
      if (response.status === 401) this.expireSession();
      throw new Error(response.status === 401
        ? 'Gateway Token 或个人 Token 无效'
        : `登录失败（${response.status}）`);
    }
    this.setSession(await response.json() as WebSession);
    this.connectEvents();
    return this.session as WebSession;
  }

  async logout(): Promise<void> {
    const headers: Record<string, string> = {};
    if (this.session) headers[CSRF_HEADER] = this.session.csrf_token;
    try {
      await fetch('/auth/session', {
        method: 'DELETE',
        credentials: 'same-origin',
        headers,
      }).catch(() => undefined);
    } finally {
      this.expireSession();
    }
  }

  async rpc<T>(method: string, params: Record<string, unknown> = {}, timeoutMs = 30_000): Promise<T> {
    if (!this.session) throw new Error('团队会话未登录');
    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), timeoutMs);
    try {
      const response = await fetch('/rpc', {
        method: 'POST',
        credentials: 'same-origin',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          [CSRF_HEADER]: this.session.csrf_token,
        },
        body: JSON.stringify({
          jsonrpc: '2.0',
          id: `web-${Date.now()}-${++this.sequence}`,
          method,
          params,
        }),
        signal: controller.signal,
      });
      if (response.status === 401) {
        this.expireSession();
        throw new Error('团队会话已失效，请重新登录');
      }
      if (response.status === 403) throw new Error('当前身份无权执行此操作，或 CSRF 校验失败');
      if (!response.ok) throw new Error(`${method} 请求失败（${response.status}）`);
      const message = await response.json() as RPCMessage;
      if (message.error) throw new Error(message.error.message);
      return message.result as T;
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        throw new Error(`${method} 请求超时`);
      }
      throw error;
    } finally {
      window.clearTimeout(timeout);
    }
  }

  on(method: string, listener: NotificationListener): () => void {
    const listeners = this.notifications.get(method) ?? new Set<NotificationListener>();
    listeners.add(listener);
    this.notifications.set(method, listeners);
    return () => listeners.delete(listener);
  }

  onConnection(listener: ConnectionListener): () => void {
    this.connectionListeners.add(listener);
    return () => this.connectionListeners.delete(listener);
  }

  onSession(listener: SessionListener): () => void {
    this.sessionListeners.add(listener);
    return () => this.sessionListeners.delete(listener);
  }

  connectEvents(): void {
    if (!this.session || this.socket?.readyState === WebSocket.OPEN ||
      this.socket?.readyState === WebSocket.CONNECTING) return;
    this.manualClose = false;
    this.emitConnection('connecting');
    const socket = new WebSocket(websocketURL(), ['goclaw.v1']);
    this.socket = socket;
    socket.onopen = () => {
      this.reconnectAttempt = 0;
      this.emitConnection('connected');
    };
    socket.onmessage = (event) => this.handleSocketMessage(String(event.data));
    socket.onerror = () => this.emitConnection('error');
    socket.onclose = (event) => {
      if (this.socket === socket) this.socket = null;
      this.rejectPending(new Error('Gateway WebSocket 已断开'));
      this.emitConnection('disconnected');
      if (!this.manualClose && this.session) {
        if ([1008, 4001, 4401].includes(event.code)) {
          this.expireSession();
        } else {
          this.scheduleReconnect();
        }
      }
    };
  }

  disconnectEvents(): void {
    this.manualClose = true;
    if (this.reconnectTimer !== null) window.clearTimeout(this.reconnectTimer);
    this.reconnectTimer = null;
    this.socket?.close(1000, 'web logout');
    this.socket = null;
    this.rejectPending(new Error('Gateway WebSocket 已关闭'));
    this.emitConnection('disconnected');
  }

  private handleSocketMessage(raw: string): void {
    let message: RPCMessage;
    try {
      message = JSON.parse(raw) as RPCMessage;
    } catch {
      return;
    }
    if (message.id && this.pending.has(message.id)) {
      const pending = this.pending.get(message.id);
      if (!pending) return;
      this.pending.delete(message.id);
      window.clearTimeout(pending.timeout);
      if (message.error) pending.reject(new Error(message.error.message));
      else pending.resolve(message.result);
      return;
    }
    if (!message.method) return;
    const payload = message.params?.data ?? message.params;
    this.notifications.get(message.method)?.forEach((listener) => listener(payload));
  }

  private scheduleReconnect(): void {
    if (this.reconnectTimer !== null) return;
    const delay = Math.min(30_000, 1000 * 2 ** this.reconnectAttempt++);
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.connectEvents();
    }, delay);
  }

  private emitConnection(state: 'connecting' | 'connected' | 'disconnected' | 'error'): void {
    this.connectionListeners.forEach((listener) => listener(state));
  }

  private setSession(session: WebSession | null): void {
    this.session = session;
    this.sessionListeners.forEach((listener) => listener(session));
  }

  private expireSession(): void {
    this.setSession(null);
    this.disconnectEvents();
  }

  private rejectPending(error: Error): void {
    this.pending.forEach((pending) => {
      window.clearTimeout(pending.timeout);
      pending.reject(error);
    });
    this.pending.clear();
  }
}

export const teamClient = new TeamClient();

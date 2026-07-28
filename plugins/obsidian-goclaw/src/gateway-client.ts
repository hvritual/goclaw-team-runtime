import {
  ChatEvent,
  ConnectionState,
  RPCMessage,
  TeamControlRPC
} from "./types";
import {
  encodeTokenProtocol,
  encodeUserTokenProtocol
} from "./util";

type NotificationListener = (payload: any) => void;
type StateListener = (state: ConnectionState, detail?: string) => void;

interface PendingCall {
  resolve(value: unknown): void;
  reject(error: Error): void;
  timeout: number;
}

export const TEAM_CONTROL_RPC = {
  members: "team.members",
  workItems: "work.items",
  issues: "issue.list",
  runners: "runner.list",
  policy: "policy.status",
  docs: "docs.summary",
  components: "components.summary"
} as const satisfies Record<string, keyof TeamControlRPC>;

export class GatewayClient {
  private socket: WebSocket | null = null;
  private pending = new Map<string, PendingCall>();
  private notifications = new Map<string, Set<NotificationListener>>();
  private stateListeners = new Set<StateListener>();
  private sequence = 0;
  private state: ConnectionState = "disconnected";
  private manualClose = false;
  private reconnectTimer: number | null = null;
  private reconnectAttempt = 0;
  private url = "";
  private token = "";
  private userToken = "";

  get connectionState(): ConnectionState {
    return this.state;
  }

  onState(listener: StateListener): () => void {
    this.stateListeners.add(listener);
    listener(this.state);
    return () => this.stateListeners.delete(listener);
  }

  on(method: "chat.event", listener: (payload: ChatEvent) => void): () => void;
  on(method: string, listener: NotificationListener): () => void;
  on(method: string, listener: NotificationListener): () => void {
    const listeners = this.notifications.get(method) ?? new Set();
    listeners.add(listener);
    this.notifications.set(method, listeners);
    return () => listeners.delete(listener);
  }

  async connect(url: string, token: string, userToken = ""): Promise<void> {
    this.disconnect();
    this.manualClose = false;
    this.url = url;
    this.token = token;
    this.userToken = userToken;
    this.setState("connecting");
    await new Promise<void>((resolve, reject) => {
      let protocols = ["goclaw.v1"];
      if (token) {
        protocols = [...protocols, encodeTokenProtocol(token)];
      }
      if (userToken) {
        protocols = [...protocols, encodeUserTokenProtocol(userToken)];
      }
      const socket = new WebSocket(url, protocols);
      this.socket = socket;
      const initialFailure = window.setTimeout(() => {
        reject(new Error("Gateway connection timed out"));
        socket.close();
      }, 10_000);
      socket.onopen = () => {
        window.clearTimeout(initialFailure);
        this.reconnectAttempt = 0;
        this.setState("connected");
        resolve();
      };
      socket.onmessage = (event) => this.handleMessage(String(event.data));
      socket.onerror = () => {
        window.clearTimeout(initialFailure);
        const error = new Error("Gateway WebSocket error");
        this.setState("error", error.message);
        reject(error);
      };
      socket.onclose = (event) => {
        window.clearTimeout(initialFailure);
        if (this.socket === socket) this.socket = null;
        this.rejectPending(new Error(`Gateway disconnected (${event.code})`));
        this.setState("disconnected", event.reason);
        if (!this.manualClose) this.scheduleReconnect();
      };
    });
  }

  disconnect(): void {
    this.manualClose = true;
    if (this.reconnectTimer !== null) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.socket?.close(1000, "plugin disconnect");
    this.socket = null;
    this.rejectPending(new Error("Gateway disconnected"));
    this.setState("disconnected");
  }

  rpc<T>(method: string, params: Record<string, unknown> = {}, timeoutMs = 20_000): Promise<T> {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return Promise.reject(new Error("Gateway is not connected"));
    }
    const id = `obsidian-${Date.now()}-${++this.sequence}`;
    return new Promise<T>((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`${method} timed out`));
      }, timeoutMs);
      this.pending.set(id, {
        resolve: resolve as (value: unknown) => void,
        reject,
        timeout
      });
      this.socket?.send(JSON.stringify({
        jsonrpc: "2.0",
        id,
        method,
        params
      }));
    });
  }

  controlRpc<M extends keyof TeamControlRPC>(
    method: M,
    params: TeamControlRPC[M]["params"],
    timeoutMs = 20_000
  ): Promise<TeamControlRPC[M]["result"]> {
    return this.rpc<TeamControlRPC[M]["result"]>(
      method,
      params as unknown as Record<string, unknown>,
      timeoutMs
    );
  }

  private handleMessage(raw: string): void {
    let message: RPCMessage;
    try {
      message = JSON.parse(raw) as RPCMessage;
    } catch {
      return;
    }
    if (message.id && this.pending.has(message.id)) {
      const pending = this.pending.get(message.id)!;
      this.pending.delete(message.id);
      window.clearTimeout(pending.timeout);
      if (message.error) {
        pending.reject(new Error(message.error.message));
      } else {
        pending.resolve(message.result);
      }
      return;
    }
    if (!message.method) return;
    const payload = message.params?.data ?? message.params;
    this.notifications.get(message.method)?.forEach((listener) => listener(payload));
  }

  private scheduleReconnect(): void {
    if (!this.url || this.reconnectTimer !== null) return;
    const delay = Math.min(30_000, 1000 * 2 ** this.reconnectAttempt++);
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      void this.connect(this.url, this.token, this.userToken).catch(() => undefined);
    }, delay);
  }

  private setState(state: ConnectionState, detail?: string): void {
    this.state = state;
    this.stateListeners.forEach((listener) => listener(state, detail));
  }

  private rejectPending(error: Error): void {
    this.pending.forEach((pending) => {
      window.clearTimeout(pending.timeout);
      pending.reject(error);
    });
    this.pending.clear();
  }
}

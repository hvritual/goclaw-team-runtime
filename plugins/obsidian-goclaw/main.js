"use strict";
var __defProp = Object.defineProperty;
var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
var __getOwnPropNames = Object.getOwnPropertyNames;
var __hasOwnProp = Object.prototype.hasOwnProperty;
var __export = (target, all) => {
  for (var name in all)
    __defProp(target, name, { get: all[name], enumerable: true });
};
var __copyProps = (to, from, except, desc) => {
  if (from && typeof from === "object" || typeof from === "function") {
    for (let key of __getOwnPropNames(from))
      if (!__hasOwnProp.call(to, key) && key !== except)
        __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
  }
  return to;
};
var __toCommonJS = (mod) => __copyProps(__defProp({}, "__esModule", { value: true }), mod);

// src/main.ts
var main_exports = {};
__export(main_exports, {
  default: () => GoClawPlugin
});
module.exports = __toCommonJS(main_exports);
var import_obsidian3 = require("obsidian");

// src/util.ts
function encodeTokenProtocol(token) {
  const bytes = new TextEncoder().encode(token);
  let binary = "";
  bytes.forEach((value) => {
    binary += String.fromCharCode(value);
  });
  return `goclaw.bearer.${btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "")}`;
}
function encodeUserTokenProtocol(token) {
  const bytes = new TextEncoder().encode(token);
  let binary = "";
  bytes.forEach((value) => {
    binary += String.fromCharCode(value);
  });
  return `goclaw.user.${btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/g, "")}`;
}
function shortHash(value) {
  return value ? value.slice(0, 8) : "new file";
}
function relativeTime(value) {
  const date = new Date(value);
  const seconds = Math.round((Date.now() - date.getTime()) / 1e3);
  if (!Number.isFinite(seconds)) return "";
  if (seconds < 60) return `${Math.max(0, seconds)} \u79D2\u524D`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)} \u5206\u949F\u524D`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} \u5C0F\u65F6\u524D`;
  return `${Math.floor(seconds / 86400)} \u5929\u524D`;
}
function safeExcerpt(value, max = 120) {
  const normalized = (value ?? "").replace(/\s+/g, " ").trim();
  return normalized.length > max ? `${normalized.slice(0, max)}\u2026` : normalized;
}
function collectionItems(value) {
  if (Array.isArray(value)) return value;
  return Array.isArray(value?.items) ? value.items : [];
}
function clampPercent(value) {
  if (!Number.isFinite(value)) return 0;
  return Math.max(0, Math.min(100, Math.round(value ?? 0)));
}
function displayInitial(value) {
  const normalized = (value ?? "").trim();
  return normalized ? Array.from(normalized)[0].toUpperCase() : "?";
}

// src/gateway-client.ts
var TEAM_CONTROL_RPC = {
  members: "team.members",
  workItems: "work.items",
  issues: "issue.list",
  runners: "runner.list",
  policy: "policy.status",
  docs: "docs.summary",
  components: "components.summary"
};
var GatewayClient = class {
  socket = null;
  pending = /* @__PURE__ */ new Map();
  notifications = /* @__PURE__ */ new Map();
  stateListeners = /* @__PURE__ */ new Set();
  sequence = 0;
  state = "disconnected";
  manualClose = false;
  reconnectTimer = null;
  reconnectAttempt = 0;
  url = "";
  token = "";
  userToken = "";
  get connectionState() {
    return this.state;
  }
  onState(listener) {
    this.stateListeners.add(listener);
    listener(this.state);
    return () => this.stateListeners.delete(listener);
  }
  on(method, listener) {
    const listeners = this.notifications.get(method) ?? /* @__PURE__ */ new Set();
    listeners.add(listener);
    this.notifications.set(method, listeners);
    return () => listeners.delete(listener);
  }
  async connect(url, token, userToken = "") {
    this.disconnect();
    this.manualClose = false;
    this.url = url;
    this.token = token;
    this.userToken = userToken;
    this.setState("connecting");
    await new Promise((resolve, reject) => {
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
      }, 1e4);
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
  disconnect() {
    this.manualClose = true;
    if (this.reconnectTimer !== null) {
      window.clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.socket?.close(1e3, "plugin disconnect");
    this.socket = null;
    this.rejectPending(new Error("Gateway disconnected"));
    this.setState("disconnected");
  }
  rpc(method, params = {}, timeoutMs = 2e4) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return Promise.reject(new Error("Gateway is not connected"));
    }
    const id = `obsidian-${Date.now()}-${++this.sequence}`;
    return new Promise((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`${method} timed out`));
      }, timeoutMs);
      this.pending.set(id, {
        resolve,
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
  controlRpc(method, params, timeoutMs = 2e4) {
    return this.rpc(
      method,
      params,
      timeoutMs
    );
  }
  handleMessage(raw) {
    let message;
    try {
      message = JSON.parse(raw);
    } catch {
      return;
    }
    if (message.id && this.pending.has(message.id)) {
      const pending = this.pending.get(message.id);
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
  scheduleReconnect() {
    if (!this.url || this.reconnectTimer !== null) return;
    const delay = Math.min(3e4, 1e3 * 2 ** this.reconnectAttempt++);
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      void this.connect(this.url, this.token, this.userToken).catch(() => void 0);
    }, delay);
  }
  setState(state, detail) {
    this.state = state;
    this.stateListeners.forEach((listener) => listener(state, detail));
  }
  rejectPending(error) {
    this.pending.forEach((pending) => {
      window.clearTimeout(pending.timeout);
      pending.reject(error);
    });
    this.pending.clear();
  }
};

// src/settings.ts
var import_obsidian = require("obsidian");
function getSecretStorage(app) {
  const storage = app.secretStorage;
  return storage ?? null;
}
var GoClawSettingTab = class extends import_obsidian.PluginSettingTab {
  constructor(app, plugin) {
    super(app, plugin);
    this.plugin = plugin;
  }
  display() {
    const { containerEl } = this;
    containerEl.empty();
    containerEl.createEl("h2", { text: "GoClaw Project Console" });
    new import_obsidian.Setting(containerEl).setName("Gateway WebSocket").setDesc("Use wss:// through a TLS reverse proxy for non-local connections.").addText((text) => text.setPlaceholder("ws://127.0.0.1:28789/ws").setValue(this.plugin.settings.gatewayUrl).onChange(async (value) => {
      this.plugin.settings.gatewayUrl = value.trim();
      await this.plugin.saveSettings();
    }));
    new import_obsidian.Setting(containerEl).setName("Project ID").setDesc("Shared routing boundary used by Obsidian, Feishu and gateway chat.").addText((text) => text.setValue(this.plugin.settings.projectId).onChange(async (value) => {
      this.plugin.settings.projectId = value.trim() || "default";
      await this.plugin.saveSettings();
    }));
    new import_obsidian.Setting(containerEl).setName("Topic ID").setDesc("Conversation topic inside the selected project.").addText((text) => text.setValue(this.plugin.settings.topicId).onChange(async (value) => {
      this.plugin.settings.topicId = value.trim() || "inbox";
      await this.plugin.saveSettings();
    }));
    const secretStorage = getSecretStorage(this.app);
    const secret = secretStorage?.getSecret(this.plugin.settings.secretKey) ?? "";
    new import_obsidian.Setting(containerEl).setName("Gateway token").setDesc(secretStorage ? "Stored in Obsidian SecretStorage; it is never written to data.json." : "This Obsidian version does not expose SecretStorage. Upgrade Obsidian before storing a token.").addText((text) => {
      text.inputEl.type = "password";
      text.setPlaceholder(secret ? "Token is stored" : "Paste gateway token").onChange((value) => {
        if (!secretStorage) {
          new import_obsidian.Notice("SecretStorage is unavailable; token was not saved.");
          return;
        }
        secretStorage.setSecret(this.plugin.settings.secretKey, value);
      });
    });
    const userSecret = secretStorage?.getSecret(
      this.plugin.settings.userSecretKey
    ) ?? "";
    new import_obsidian.Setting(containerEl).setName("Team user token").setDesc(secretStorage ? "Personal team identity token. It is stored in SecretStorage and is separate from the shared Gateway token." : "SecretStorage is unavailable. Upgrade Obsidian before enabling team mode.").addText((text) => {
      text.inputEl.type = "password";
      text.setPlaceholder(userSecret ? "User token is stored" : "Paste personal user token").onChange((value) => {
        if (!secretStorage) {
          new import_obsidian.Notice("SecretStorage is unavailable; user token was not saved.");
          return;
        }
        secretStorage.setSecret(this.plugin.settings.userSecretKey, value);
      });
    });
    new import_obsidian.Setting(containerEl).setName("Reviewer ID").setDesc("Legacy single-user mode only. In team mode, approval identity is bound to the personal team token.").addText((text) => text.setPlaceholder("alice").setValue(this.plugin.settings.reviewerId).onChange(async (value) => {
      this.plugin.settings.reviewerId = value.trim();
      await this.plugin.saveSettings();
    }));
    const reviewerSecret = secretStorage?.getSecret(
      this.plugin.settings.reviewerSecretKey
    ) ?? "";
    new import_obsidian.Setting(containerEl).setName("Reviewer token").setDesc(secretStorage ? "Stored separately in Obsidian SecretStorage and sent only on decision RPCs." : "SecretStorage is unavailable; authenticated approvals are disabled on this device.").addText((text) => {
      text.inputEl.type = "password";
      text.setPlaceholder(reviewerSecret ? "Reviewer token is stored" : "Paste reviewer token").onChange((value) => {
        if (!secretStorage) {
          new import_obsidian.Notice("SecretStorage is unavailable; reviewer token was not saved.");
          return;
        }
        secretStorage.setSecret(this.plugin.settings.reviewerSecretKey, value);
      });
    });
    new import_obsidian.Setting(containerEl).setName("Connect automatically").setDesc("Reconnect with exponential backoff when Obsidian starts.").addToggle((toggle) => toggle.setValue(this.plugin.settings.autoConnect).onChange(async (value) => {
      this.plugin.settings.autoConnect = value;
      await this.plugin.saveSettings();
    }));
    new import_obsidian.Setting(containerEl).setName("Reconnect now").addButton((button) => button.setButtonText("Reconnect").onClick(() => void this.plugin.connectGateway()));
  }
};

// src/types.ts
var DEFAULT_SETTINGS = {
  gatewayUrl: "ws://127.0.0.1:28789/ws",
  projectId: "default",
  topicId: "inbox",
  autoConnect: true,
  secretKey: "goclaw.gateway.token",
  userSecretKey: "goclaw.team.user-token",
  reviewerId: "obsidian-user",
  reviewerSecretKey: "goclaw.governance.reviewer-token"
};

// src/view.ts
var import_obsidian2 = require("obsidian");

// src/team-presenter.ts
var memberStates = {
  active: { label: "\u5728\u7EBF", tone: "success" },
  away: { label: "\u6682\u79BB", tone: "warning" },
  offline: { label: "\u79BB\u7EBF", tone: "muted" },
  disabled: { label: "\u505C\u7528", tone: "danger" }
};
var workStates = {
  backlog: { label: "\u5F85\u89C4\u5212", tone: "muted" },
  ready: { label: "\u53EF\u6267\u884C", tone: "accent" },
  in_progress: { label: "\u8FDB\u884C\u4E2D", tone: "success" },
  blocked: { label: "\u53D7\u963B", tone: "danger" },
  in_review: { label: "\u8BC4\u5BA1\u4E2D", tone: "warning" },
  done: { label: "\u5DF2\u5B8C\u6210", tone: "muted" },
  cancelled: { label: "\u5DF2\u53D6\u6D88", tone: "muted" }
};
var issueStates = {
  open: { label: "\u5F85\u5904\u7406", tone: "danger" },
  triaged: { label: "\u5DF2\u5206\u8BCA", tone: "warning" },
  in_progress: { label: "\u4FEE\u590D\u4E2D", tone: "accent" },
  in_review: { label: "\u9A8C\u8BC1\u4E2D", tone: "warning" },
  resolved: { label: "\u5DF2\u89E3\u51B3", tone: "success" },
  closed: { label: "\u5DF2\u5173\u95ED", tone: "muted" },
  reopened: { label: "\u91CD\u65B0\u6253\u5F00", tone: "danger" }
};
var runnerStates = {
  online: { label: "\u5728\u7EBF", tone: "success" },
  busy: { label: "\u6267\u884C\u4E2D", tone: "accent" },
  draining: { label: "\u6392\u7A7A\u4E2D", tone: "warning" },
  offline: { label: "\u79BB\u7EBF", tone: "danger" }
};
var severityStates = {
  critical: { label: "\u81F4\u547D", tone: "danger" },
  high: { label: "\u9AD8", tone: "danger" },
  medium: { label: "\u4E2D", tone: "warning" },
  low: { label: "\u4F4E", tone: "muted" }
};
function memberState(value) {
  return memberStates[value] ?? { label: value, tone: "muted" };
}
function workState(value) {
  return workStates[value] ?? { label: value, tone: "muted" };
}
function issueState(value) {
  return issueStates[value] ?? { label: value, tone: "muted" };
}
function runnerState(value) {
  return runnerStates[value] ?? { label: value, tone: "muted" };
}
function severityState(value) {
  return severityStates[value] ?? { label: value, tone: "muted" };
}
function leaseState(expiresAt, now = Date.now()) {
  if (!expiresAt) return { label: "\u65E0\u79DF\u7EA6", tone: "muted" };
  const expires = new Date(expiresAt).getTime();
  if (!Number.isFinite(expires)) return { label: "\u79DF\u7EA6\u65F6\u95F4\u65E0\u6548", tone: "danger" };
  const remaining = expires - now;
  if (remaining <= 0) return { label: "\u79DF\u7EA6\u5DF2\u8FC7\u671F", tone: "danger" };
  if (remaining <= 5 * 6e4) return { label: "\u79DF\u7EA6\u5373\u5C06\u5230\u671F", tone: "warning" };
  return { label: "\u79DF\u7EA6\u6709\u6548", tone: "success" };
}
function teamStateClass(presentation) {
  return `goclaw-team-state is-${presentation.tone}`;
}

// src/view.ts
var VIEW_TYPE_GOCLAW = "goclaw-project-console";
var GoClawView = class extends import_obsidian2.ItemView {
  constructor(leaf, plugin, client) {
    super(leaf);
    this.plugin = plugin;
    this.client = client;
  }
  activeTab = "chat";
  bodyEl = null;
  statusEl = null;
  chatMessages = [];
  disposers = [];
  getViewType() {
    return VIEW_TYPE_GOCLAW;
  }
  getDisplayText() {
    return "GoClaw";
  }
  getIcon() {
    return "bot-message-square";
  }
  async onOpen() {
    this.disposers.push(
      this.client.onState((state, detail) => this.updateStatus(state, detail)),
      this.client.on("chat.event", (payload) => this.onChatEvent(payload))
    );
    this.renderShell();
  }
  async onClose() {
    this.disposers.splice(0).forEach((dispose) => dispose());
  }
  refresh() {
    this.renderShell();
  }
  renderShell() {
    const root = this.containerEl.children[1];
    root.empty();
    root.addClass("goclaw-console");
    const header = root.createDiv({ cls: "goclaw-header" });
    const brand = header.createDiv({ cls: "goclaw-brand" });
    const brandIcon = brand.createSpan({ cls: "goclaw-brand-icon" });
    (0, import_obsidian2.setIcon)(brandIcon, "bot");
    brand.createSpan({ text: "GoClaw" });
    const reconnect = header.createEl("button", {
      cls: "clickable-icon goclaw-icon-button",
      attr: { "aria-label": "Reconnect gateway" }
    });
    (0, import_obsidian2.setIcon)(reconnect, "refresh-cw");
    reconnect.addEventListener("click", () => void this.plugin.connectGateway());
    const project = root.createDiv({ cls: "goclaw-project-row" });
    const folderIcon = project.createSpan();
    (0, import_obsidian2.setIcon)(folderIcon, "folder-kanban");
    project.createSpan({
      cls: "goclaw-project-name",
      text: this.plugin.settings.projectId
    });
    const status = project.createSpan({ cls: "goclaw-connection" });
    status.createSpan({ cls: "goclaw-status-dot" });
    this.statusEl = status.createSpan({ text: "\u672A\u8FDE\u63A5" });
    this.updateStatus(this.client.connectionState);
    const tabs = root.createDiv({ cls: "goclaw-tabs", attr: { role: "tablist" } });
    [
      ["chat", "\u804A\u5929"],
      ["spec", "\u89C4\u683C"],
      ["memory", "\u8BB0\u5FC6"],
      ["approvals", "\u5BA1\u6279"],
      ["development", "\u5F00\u53D1"],
      ["team", "\u56E2\u961F"],
      ["progress", "\u8FDB\u5EA6"],
      ["harness", "Harness"]
    ].forEach(([id, label]) => {
      const button = tabs.createEl("button", {
        text: label,
        cls: this.activeTab === id ? "is-active" : "",
        attr: {
          role: "tab",
          "aria-selected": String(this.activeTab === id)
        }
      });
      button.addEventListener("click", () => {
        this.activeTab = id;
        this.renderShell();
      });
    });
    this.bodyEl = root.createDiv({ cls: "goclaw-body" });
    this.renderActiveTab();
    const footer = root.createDiv({ cls: "goclaw-footer" });
    const syncIcon = footer.createSpan();
    (0, import_obsidian2.setIcon)(syncIcon, "refresh-cw");
    footer.createSpan({ text: "Vault \u5C31\u7EEA" });
    footer.createSpan({
      cls: "goclaw-footer-detail",
      text: "\u540C\u6B65\u7531 Obsidian \u7BA1\u7406",
      attr: { title: "\u63D2\u4EF6\u4E0D\u4F2A\u9020\u8FDC\u7AEF\u540C\u6B65\u72B6\u6001\uFF1B\u8BF7\u5728 Obsidian Sync \u4E2D\u786E\u8BA4\u8BBE\u5907\u72B6\u6001\u3002" }
    });
  }
  renderActiveTab() {
    if (!this.bodyEl) return;
    this.bodyEl.empty();
    if (this.activeTab === "chat") this.renderChat();
    if (this.activeTab === "spec") void this.renderSpec();
    if (this.activeTab === "memory") void this.renderMemory();
    if (this.activeTab === "approvals") void this.renderApprovals();
    if (this.activeTab === "development") void this.renderDevelopment();
    if (this.activeTab === "team") void this.renderTeam();
    if (this.activeTab === "progress") void this.renderProgress();
    if (this.activeTab === "harness") void this.renderHarness();
  }
  renderChat() {
    if (!this.bodyEl) return;
    const list = this.bodyEl.createDiv({ cls: "goclaw-chat-list" });
    if (this.chatMessages.length === 0) {
      const empty = list.createDiv({ cls: "goclaw-empty" });
      const icon = empty.createSpan();
      (0, import_obsidian2.setIcon)(icon, "message-square");
      empty.createEl("strong", { text: "\u9879\u76EE\u4F1A\u8BDD\u5C1A\u672A\u5F00\u59CB" });
      empty.createEl("p", { text: "\u8FD9\u91CC\u4E0E\u98DE\u4E66\u5171\u4EAB\u540C\u4E00 project_id\uFF1Btopic_id \u7528\u4E8E\u7EC6\u5206\u8BA8\u8BBA\u3002" });
    } else {
      this.chatMessages.forEach((message) => {
        const item = list.createDiv({
          cls: `goclaw-message is-${message.role}${message.pending ? " is-pending" : ""}${message.error ? " is-error" : ""}`
        });
        item.createDiv({
          cls: "goclaw-message-role",
          text: message.role === "user" ? "\u4F60" : message.role === "assistant" ? "GoClaw" : "\u7CFB\u7EDF"
        });
        item.createDiv({ cls: "goclaw-message-content", text: message.content || "\u2026" });
      });
    }
    const composer = this.bodyEl.createDiv({ cls: "goclaw-composer" });
    const textarea = composer.createEl("textarea", {
      attr: {
        placeholder: "\u7ED9\u5F53\u524D\u9879\u76EE\u53D1\u9001\u6D88\u606F\u2026",
        rows: "3",
        "aria-label": "GoClaw message"
      }
    });
    const send = composer.createEl("button", { cls: "mod-cta", text: "\u53D1\u9001" });
    const submit = async () => {
      const content = textarea.value.trim();
      if (!content) return;
      textarea.value = "";
      this.chatMessages.push({
        id: crypto.randomUUID(),
        role: "user",
        content
      });
      this.renderChat();
      try {
        await this.client.rpc("agent", {
          content,
          project_id: this.plugin.settings.projectId,
          topic_id: this.plugin.settings.topicId
        });
      } catch (error) {
        this.chatMessages.push({
          id: crypto.randomUUID(),
          role: "system",
          content: error instanceof Error ? error.message : String(error),
          error: true
        });
        this.renderChat();
      }
    };
    send.addEventListener("click", () => void submit());
    textarea.addEventListener("keydown", (event) => {
      if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
        event.preventDefault();
        void submit();
      }
    });
  }
  onChatEvent(payload) {
    if (!payload || !payload.run_id) return;
    if (payload.project_id && payload.project_id !== this.plugin.settings.projectId) return;
    if (payload.topic_id && payload.topic_id !== this.plugin.settings.topicId) return;
    let message = this.chatMessages.find((item) => item.id === payload.run_id);
    if (!message) {
      message = {
        id: payload.run_id,
        role: "assistant",
        content: "",
        pending: true
      };
      this.chatMessages.push(message);
    }
    if (payload.state === "delta") message.content += payload.content ?? "";
    if (payload.state === "thinking" && !message.content) message.content = "\u6B63\u5728\u5206\u6790\u2026";
    if (payload.state === "tool") message.content = payload.content || "\u6B63\u5728\u8C03\u7528\u5DE5\u5177\u2026";
    if (payload.state === "final") {
      message.content = payload.content || message.content;
      message.pending = false;
    }
    if (payload.state === "error") {
      message.content = payload.content || "\u8FD0\u884C\u5931\u8D25";
      message.pending = false;
      message.error = true;
    }
    if (this.activeTab === "chat") this.renderActiveTab();
  }
  async renderSpec() {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "\u52A0\u8F7D Ouroboros \u89C4\u683C\u95ED\u73AF");
    try {
      const sessions = await this.client.rpc("ouroboros.sessions", {
        project_id: this.plugin.settings.projectId
      });
      if (this.activeTab !== "spec" || !this.bodyEl) return;
      target.empty();
      const intro = target.createDiv({ cls: "goclaw-spec-intro" });
      const title = intro.createDiv({ cls: "goclaw-review-title" });
      const icon = title.createSpan();
      (0, import_obsidian2.setIcon)(icon, "infinity");
      const copy = title.createDiv();
      copy.createEl("strong", { text: "\u5148\u7ED3\u6676\u89C4\u683C\uFF0C\u518D\u8FDB\u5165\u53D7\u63A7\u5F00\u53D1" });
      copy.createSpan({ text: "interview \u2192 Seed \u2192 approval \u2192 compile \u2192 evaluate \u2192 evolve" });
      intro.createEl("p", {
        text: "\u804A\u5929\u8BB0\u5FC6\u53EA\u63D0\u4F9B\u4E0A\u4E0B\u6587\uFF1B\u4E0D\u53EF\u53D8 Seed\u3001\u9A8C\u8BC1\u8BC1\u636E\u548C Go \u4E8B\u4EF6\u94FE\u624D\u662F\u6267\u884C\u4F9D\u636E\u3002"
      });
      const form = target.createEl("form", { cls: "goclaw-spec-form" });
      const request = form.createEl("textarea", {
        attr: {
          rows: "4",
          placeholder: "\u63CF\u8FF0\u8981\u5F00\u53D1\u7684\u76EE\u6807\u3001\u7EA6\u675F\u4E0E\u9A8C\u6536\u65B9\u5F0F\u2026",
          "aria-label": "Ouroboros development request"
        }
      });
      const options = form.createDiv({ cls: "goclaw-spec-options" });
      const brownfieldLabel = options.createEl("label");
      const brownfield = brownfieldLabel.createEl("input", { type: "checkbox" });
      brownfield.checked = true;
      brownfieldLabel.createSpan({ text: "\u73B0\u6709\u4EE3\u7801\u5E93" });
      const start = options.createEl("button", {
        cls: "mod-cta",
        text: "\u5F00\u59CB\u89C4\u683C\u8BBF\u8C08",
        attr: { type: "submit" }
      });
      form.addEventListener("submit", (event) => {
        event.preventDefault();
        const rawRequest = request.value.trim();
        if (!rawRequest) {
          new import_obsidian2.Notice("\u8BF7\u5148\u8F93\u5165\u5F00\u53D1\u9700\u6C42");
          return;
        }
        start.disabled = true;
        void this.client.rpc("ouroboros.session.start", {
          project_id: this.plugin.settings.projectId,
          topic_id: this.plugin.settings.topicId,
          raw_request: rawRequest,
          brownfield: brownfield.checked,
          base_ref: "HEAD",
          created_by: this.plugin.actorId()
        }, 24e4).then(async () => {
          request.value = "";
          new import_obsidian2.Notice("\u89C4\u683C\u8BBF\u8C08\u5DF2\u5F00\u59CB");
          await this.renderSpec();
        }).catch((error) => {
          new import_obsidian2.Notice(error instanceof Error ? error.message : String(error));
        }).finally(() => {
          start.disabled = false;
        });
      });
      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "\u89C4\u683C", sessions.length);
      this.metric(summary, "\u5F85\u6F84\u6E05", sessions.filter((session) => ["interviewing", "clarification_required"].includes(session.status)).length);
      this.metric(summary, "\u5F85\u5BA1\u6279", sessions.filter((session) => ["awaiting_seed_approval", "evolution_pending"].includes(session.status)).length);
      const list = target.createDiv({ cls: "goclaw-spec-list" });
      sessions.forEach((session) => list.appendChild(this.renderSpecSession(session)));
      if (sessions.length === 0) {
        this.renderEmpty(list, "infinity", "\u8FD8\u6CA1\u6709\u89C4\u683C\u4F1A\u8BDD", "\u8F93\u5165\u5F00\u53D1\u76EE\u6807\uFF0COuroboros \u4F1A\u4F18\u5148\u8FFD\u95EE\u9AD8\u4FE1\u606F\u589E\u76CA\u95EE\u9898\u3002");
      }
    } catch (error) {
      if (this.activeTab === "spec") this.renderError(target, error);
    }
  }
  renderSpecSession(session) {
    const card = document.createElement("article");
    card.addClass("goclaw-spec-card", `is-${session.status}`);
    const heading = card.createDiv({ cls: "goclaw-spec-heading" });
    const info = heading.createDiv();
    info.createEl("strong", { text: session.title });
    info.createSpan({ text: `${session.id} \xB7 ${relativeTime(session.updated_at)}` });
    heading.createSpan({ cls: `goclaw-state is-${session.status}`, text: session.status });
    card.createEl("p", { text: safeExcerpt(session.raw_request, 180) });
    const latest = session.rounds?.at(-1);
    if (latest) {
      const meter = card.createDiv({ cls: "goclaw-ambiguity" });
      const label = meter.createDiv();
      label.createSpan({ text: "\u6B67\u4E49\u5EA6" });
      label.createEl("strong", {
        text: `${Math.round(latest.assessment.overall * 100)}% / \u9608\u503C ${Math.round(latest.assessment.threshold * 100)}%`
      });
      const track = meter.createDiv({ cls: "goclaw-ambiguity-track" });
      const fill = track.createDiv({ cls: "goclaw-ambiguity-fill" });
      fill.style.width = `${Math.min(100, Math.max(0, latest.assessment.overall * 100))}%`;
      meter.createSpan({
        cls: "goclaw-spec-note",
        text: `${latest.assessment.summary} \xB7 ready ${latest.assessment.ready_streak}/${latest.assessment.required_ready_streak} \xB7 \u5206\u6B67 ${Math.round((latest.assessment.score_spread ?? 0) * 100)}%${latest.assessment.gray_zone ? " \xB7 \u7070\u533A" : ""}`
      });
    }
    const answered = new Set((latest?.answers ?? []).map((answer) => answer.question_id));
    const questions = (latest?.questions ?? []).filter((question) => !answered.has(question.id));
    if (questions.length > 0 && ["interviewing", "clarification_required"].includes(session.status)) {
      const questionForm = card.createEl("form", { cls: "goclaw-question-list" });
      const inputs = /* @__PURE__ */ new Map();
      questions.forEach((question) => {
        const item = questionForm.createDiv({ cls: "goclaw-question" });
        const questionLabel = item.createEl("label", {
          text: `${question.blocking ? "\u5FC5\u7B54 \xB7 " : ""}${question.text}`
        });
        if (question.why) questionLabel.setAttribute("title", question.why);
        const input = item.createEl("textarea", {
          attr: { rows: "2", placeholder: "\u8F93\u5165\u660E\u786E\u7B54\u6848\u2026" }
        });
        inputs.set(question.id, input);
      });
      const actions2 = questionForm.createDiv({ cls: "goclaw-actions" });
      const submit = actions2.createEl("button", {
        cls: "mod-cta",
        text: "\u63D0\u4EA4\u7B54\u6848\u5E76\u91CD\u8BC4",
        attr: { type: "submit" }
      });
      questionForm.addEventListener("submit", (event) => {
        event.preventDefault();
        const answers = Array.from(inputs.entries()).map(([questionId, input]) => ({ question_id: questionId, text: input.value.trim() })).filter((answer) => answer.text.length > 0);
        if (answers.length === 0) {
          new import_obsidian2.Notice("\u81F3\u5C11\u56DE\u7B54\u4E00\u4E2A\u95EE\u9898");
          return;
        }
        submit.disabled = true;
        void this.client.rpc("ouroboros.session.answer", {
          id: session.id,
          answers,
          actor: this.plugin.actorId()
        }, 24e4).then(() => this.renderSpec()).catch((error) => {
          new import_obsidian2.Notice(error instanceof Error ? error.message : String(error));
        }).finally(() => {
          submit.disabled = false;
        });
      });
    }
    if (session.last_error) {
      const error = card.createDiv({ cls: "goclaw-safety is-error" });
      (0, import_obsidian2.setIcon)(error.createSpan(), "circle-x");
      error.createSpan({ text: safeExcerpt(session.last_error, 160) });
    }
    if (session.status === "awaiting_seed_approval") {
      const note = card.createDiv({ cls: "goclaw-safety" });
      (0, import_obsidian2.setIcon)(note.createSpan(), "shield-check");
      note.createSpan({ text: `Seed ${shortHash(session.pending_seed_hash)} \u7B49\u5F85\u4EBA\u5DE5\u5BA1\u6279` });
    }
    if (session.status === "evolution_pending") {
      const note = card.createDiv({ cls: "goclaw-safety" });
      (0, import_obsidian2.setIcon)(note.createSpan(), "git-compare-arrows");
      note.createSpan({
        text: `\u5019\u9009 G${session.pending_evolution?.candidate_generation ?? "?"} \xB7 \u672C\u4F53\u76F8\u4F3C\u5EA6 ${Math.round((session.pending_evolution?.ontology_similarity ?? 0) * 100)}%`
      });
    }
    const actions = card.createDiv({ cls: "goclaw-actions" });
    if (["interviewing", "clarification_required"].includes(session.status)) {
      this.actionButton(actions, "\u91CD\u65B0\u8BC4\u4F30", "", async () => {
        await this.client.rpc("ouroboros.session.reassess", {
          id: session.id,
          actor: this.plugin.actorId()
        }, 24e4);
        await this.renderSpec();
      });
    }
    if (session.status === "seed_ready") {
      this.actionButton(actions, "\u751F\u6210\u4E0D\u53EF\u53D8 Seed", "mod-cta", async () => {
        await this.client.rpc("ouroboros.session.crystallize", {
          id: session.id,
          actor: this.plugin.actorId()
        }, 24e4);
        new import_obsidian2.Notice("Seed \u5DF2\u751F\u6210\uFF0C\u8BF7\u5230\u5BA1\u6279\u9875\u590D\u6838");
        await this.renderSpec();
      });
    }
    if (session.status === "approved") {
      this.actionButton(actions, "\u7F16\u8BD1\u4E3A\u5F00\u53D1\u4EFB\u52A1", "mod-cta", async () => {
        await this.client.rpc("ouroboros.session.compile", {
          id: session.id,
          actor: this.plugin.actorId()
        });
        new import_obsidian2.Notice("\u5DF2\u7F16\u8BD1\u4E3A Orchestrator Lite \u4EFB\u52A1\uFF0C\u4ECD\u9700\u56DB\u7C7B\u5BA1\u6279");
        await this.renderSpec();
      });
    }
    if (session.status === "compiled") {
      const task = session.compiled_tasks?.at(-1);
      if (task) {
        this.actionButton(actions, "\u4F9D\u636E\u8BC1\u636E\u8BC4\u4F30", "mod-cta", async () => {
          await this.client.rpc("ouroboros.session.evaluate", {
            id: session.id,
            task_id: task.task_id,
            actor: this.plugin.actorId()
          }, 9e5);
          await this.renderSpec();
        });
      }
    }
    if (session.status === "evaluated") {
      this.actionButton(actions, "\u751F\u6210\u6F14\u5316\u5019\u9009", "mod-cta", async () => {
        await this.client.rpc("ouroboros.session.evolve", {
          id: session.id,
          actor: this.plugin.actorId()
        }, 24e4);
        await this.renderSpec();
      });
    }
    if (!["converged", "cancelled", "rejected"].includes(session.status)) {
      const cancelBlock = card.createDiv({ cls: "goclaw-question" });
      const fields = this.reviewFields(cancelBlock, "\u53D6\u6D88 Ouroboros \u4F1A\u8BDD");
      this.actionButton(cancelBlock, "\u53D6\u6D88\u4F1A\u8BDD", "mod-warning", async () => {
        await this.client.rpc("ouroboros.session.cancel", {
          id: session.id,
          reason: fields.rationale.value.trim(),
          ...this.reviewPayload(
            fields,
            [`session:${session.id}`, `status:${session.status}`],
            true
          )
        });
        await this.renderSpec();
      });
    }
    return card;
  }
  async renderApprovals() {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "\u52A0\u8F7D\u5BA1\u6279\u961F\u5217");
    try {
      const [knowledge, memoryCandidates, experiments, tasks, ouroborosSessions] = await Promise.all([
        this.client.rpc("knowledge.proposals", { status: "pending" }),
        this.client.rpc("memory.catalog.list", {
          project_id: this.plugin.settings.projectId,
          status: "pending",
          limit: 100
        }).catch(() => []),
        this.client.rpc("harness.experiments"),
        this.client.rpc("dev.tasks", { project_id: this.plugin.settings.projectId }).catch(() => []),
        this.client.rpc("ouroboros.sessions", {
          project_id: this.plugin.settings.projectId
        }).catch(() => [])
      ]);
      if (this.activeTab !== "approvals" || !this.bodyEl) return;
      target.empty();
      this.renderApprovalSection(target, "\u77E5\u8BC6\u63D0\u6848", knowledge.length, knowledge, (proposal) => this.renderKnowledgeProposal(proposal));
      this.renderApprovalSection(
        target,
        "\u8BB0\u5FC6\u7F16\u76EE\u5019\u9009",
        memoryCandidates.length,
        memoryCandidates,
        (record) => this.renderCatalogCandidate(record)
      );
      const reviewable = experiments.filter((experiment) => ["validated", "human_approved"].includes(experiment.status));
      this.renderApprovalSection(target, "Harness \u5B9E\u9A8C", reviewable.length, reviewable, (experiment) => this.renderExperiment(experiment));
      const developmentReviewable = tasks.filter((task) => ["review_pending", "ready_to_freeze", "blocked", "awaiting_acceptance"].includes(task.status));
      this.renderApprovalSection(
        target,
        "\u5F00\u53D1\u4EFB\u52A1",
        developmentReviewable.length,
        developmentReviewable,
        (task) => this.renderDevelopmentApproval(task)
      );
      const seedReviewable = ouroborosSessions.filter((session) => session.status === "awaiting_seed_approval");
      const evolutionReviewable = ouroborosSessions.filter((session) => session.status === "evolution_pending");
      const cognitiveReviewable = ouroborosSessions.filter((session) => {
        const latest = session.rounds?.at(-1)?.assessment;
        return Boolean(
          latest?.human_decision_required || session.decision_conflicts?.some((conflict) => conflict.status === "open") || session.evaluations?.some((evaluation) => evaluation.human_decision_required)
        );
      });
      const seedHashes = Array.from(new Set([
        ...seedReviewable.map((session) => session.pending_seed_hash),
        ...evolutionReviewable.map((session) => session.pending_evolution?.candidate_seed_hash)
      ].filter((hash) => Boolean(hash))));
      const seedEntries = await Promise.all(seedHashes.map(async (hash) => {
        const seed = await this.client.rpc("ouroboros.seed.get", { hash }).catch(() => null);
        return [hash, seed];
      }));
      const seeds = new Map(seedEntries);
      if (this.activeTab !== "approvals" || !this.bodyEl) return;
      this.renderApprovalSection(
        target,
        "\u8BA4\u77E5\u5206\u6B67\u4E0E\u5229\u76CA\u76F8\u5173\u65B9\u51B2\u7A81",
        cognitiveReviewable.length,
        cognitiveReviewable,
        (session) => this.renderCognitiveEscalation(session)
      );
      this.renderApprovalSection(
        target,
        "Ouroboros Seed",
        seedReviewable.length,
        seedReviewable,
        (session) => this.renderOuroborosSeedApproval(
          session,
          session.pending_seed_hash ? seeds.get(session.pending_seed_hash) ?? null : null
        )
      );
      this.renderApprovalSection(
        target,
        "Ouroboros \u6F14\u5316\u5019\u9009",
        evolutionReviewable.length,
        evolutionReviewable,
        (session) => this.renderOuroborosEvolutionApproval(
          session,
          session.pending_evolution?.candidate_seed_hash ? seeds.get(session.pending_evolution.candidate_seed_hash) ?? null : null
        )
      );
      if (knowledge.length + memoryCandidates.length + reviewable.length + developmentReviewable.length + cognitiveReviewable.length + seedReviewable.length + evolutionReviewable.length === 0) {
        this.renderEmpty(target, "inbox", "\u6CA1\u6709\u5F85\u5BA1\u6279\u4E8B\u9879", "\u65B0\u63D0\u6848\u4E0E\u901A\u8FC7\u8BC4\u6D4B\u7684\u5B9E\u9A8C\u4F1A\u51FA\u73B0\u5728\u8FD9\u91CC\u3002");
      }
    } catch (error) {
      if (this.activeTab === "approvals") this.renderError(target, error);
    }
  }
  renderCognitiveEscalation(session) {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const latest = session.rounds?.at(-1);
    const disputedEvaluation = [...session.evaluations ?? []].reverse().find((evaluation) => evaluation.human_decision_required);
    const title = row.createDiv({ cls: "goclaw-review-title" });
    (0, import_obsidian2.setIcon)(title.createSpan(), "scale");
    const copy = title.createDiv();
    copy.createEl("strong", { text: session.title });
    copy.createSpan({
      text: disputedEvaluation ? `\u8BC1\u636E\u8BC4\u4F30\u4E89\u8BAE \xB7 ${disputedEvaluation.distinct_models ?? 0} \u4E2A\u6A21\u578B \xB7 \u5206\u5DEE ${Math.round((disputedEvaluation.score_spread ?? 0) * 100)}%` : `\u9700\u6C42\u8BC4\u4F30\u5206\u6B67 \xB7 ${latest?.assessment.distinct_models ?? 0} \u4E2A\u6A21\u578B \xB7 \u5206\u5DEE ${Math.round((latest?.assessment.score_spread ?? 0) * 100)}%${latest?.assessment.gray_zone ? " \xB7 \u9608\u503C\u7070\u533A" : ""}`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: safeExcerpt(
        disputedEvaluation?.consensus.summary ?? latest?.assessment.unresolved?.join("\uFF1B"),
        240
      ) || "\u591A\u4E2A\u8BC4\u4F30\u89C6\u89D2\u65E0\u6CD5\u7ED9\u51FA\u7A33\u5B9A\u7684\u81EA\u52A8\u7ED3\u8BBA\u3002"
    });
    const fields = this.reviewFields(row, "\u8BA4\u77E5\u5206\u6B67");
    const openConflicts = (session.decision_conflicts ?? []).filter((conflict) => conflict.status === "open");
    openConflicts.forEach((conflict) => {
      const block = row.createDiv({ cls: "goclaw-question" });
      block.createEl("strong", { text: conflict.description });
      const resolution = block.createEl("textarea", {
        attr: {
          rows: "2",
          placeholder: "\u660E\u786E\u9009\u62E9\u3001\u4F18\u5148\u7EA7\u6216\u53EF\u9A8C\u8BC1\u6298\u4E2D\u65B9\u6848\u2026",
          "aria-label": `Resolve conflict ${conflict.id}`
        }
      });
      this.actionButton(block, "\u89E3\u51B3\u6B64\u51B2\u7A81", "mod-cta", async () => {
        if (!resolution.value.trim()) throw new Error("\u8BF7\u586B\u5199\u660E\u786E\u7684\u51B2\u7A81\u89E3\u51B3\u65B9\u6848");
        await this.client.rpc("ouroboros.conflict.resolve", {
          id: session.id,
          conflict_id: conflict.id,
          resolution: resolution.value.trim(),
          ...this.reviewPayload(fields, [`conflict:${conflict.id}`], true)
        });
        await this.renderApprovals();
      });
    });
    if (disputedEvaluation) {
      const block = row.createDiv({ cls: "goclaw-question" });
      block.createEl("strong", { text: `\u8BC4\u4F30 ${disputedEvaluation.id}` });
      block.createDiv({
        text: [
          `\u673A\u68B0\u95E8\uFF1A${disputedEvaluation.mechanical.passed ? "\u901A\u8FC7" : "\u5931\u8D25"}`,
          `\u8BED\u4E49\u95E8\uFF1A${disputedEvaluation.semantic.passed ? "\u901A\u8FC7" : "\u5931\u8D25"}`,
          `\u5171\u8BC6\u95E8\uFF1A${disputedEvaluation.consensus.passed ? "\u901A\u8FC7" : "\u4E89\u8BAE"}`
        ].join(" \xB7 ")
      });
      block.createDiv({
        cls: "goclaw-review-reason",
        text: "\u8FD9\u91CC\u4EC5\u88C1\u51B3\u8BC1\u636E\u4E89\u8BAE\uFF0C\u4E0D\u4EE3\u8868\u5F00\u53D1\u4EFB\u52A1\u9A8C\u6536\u3001\u90E8\u7F72\u6388\u6743\u6216 Harness \u664B\u7EA7\u3002"
      });
      const actions = block.createDiv({ cls: "goclaw-actions" });
      this.actionButton(actions, "\u9A73\u56DE\u4E89\u8BAE\u8BC4\u4F30", "mod-warning", async () => {
        await this.client.rpc("ouroboros.evaluation.resolve", {
          id: session.id,
          evaluation_id: disputedEvaluation.id,
          accepted: false,
          ...this.reviewPayload(
            fields,
            [`evaluation:${disputedEvaluation.id}`],
            true
          )
        });
        await this.renderApprovals();
      });
      this.actionButton(actions, "\u63A5\u53D7\u8BC1\u636E\u7ED3\u8BBA\uFF08\u975E\u9A8C\u6536\uFF09", "mod-cta", async () => {
        await this.client.rpc("ouroboros.evaluation.resolve", {
          id: session.id,
          evaluation_id: disputedEvaluation.id,
          accepted: true,
          ...this.reviewPayload(
            fields,
            [`evaluation:${disputedEvaluation.id}`],
            true
          )
        });
        await this.renderApprovals();
      });
    }
    if (latest?.assessment.human_decision_required && openConflicts.length === 0) {
      const actions = row.createDiv({ cls: "goclaw-actions" });
      this.actionButton(actions, "\u5224\u5B9A\u4ECD\u9700\u6F84\u6E05", "mod-warning", async () => {
        await this.client.rpc("ouroboros.readiness.resolve", {
          id: session.id,
          ready: false,
          ...this.reviewPayload(fields, [`assessment:${session.id}:${latest.number}`], true)
        });
        await this.renderApprovals();
      });
      this.actionButton(actions, "\u5224\u5B9A\u53EF\u7ED3\u6676", "mod-cta", async () => {
        await this.client.rpc("ouroboros.readiness.resolve", {
          id: session.id,
          ready: true,
          ...this.reviewPayload(fields, [`assessment:${session.id}:${latest.number}`], true)
        });
        await this.renderApprovals();
      });
    }
    return row;
  }
  renderApprovalSection(parent, title, count, items, render) {
    const section = parent.createDiv({ cls: "goclaw-section" });
    const heading = section.createDiv({ cls: "goclaw-section-heading" });
    heading.createEl("h3", { text: title });
    heading.createSpan({ cls: "goclaw-count", text: String(count) });
    items.forEach((item) => section.appendChild(render(item)));
  }
  renderKnowledgeProposal(proposal) {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    const icon = title.createSpan();
    (0, import_obsidian2.setIcon)(icon, "file-check-2");
    const titleText = title.createDiv();
    titleText.createEl("strong", { text: proposal.target_path });
    titleText.createSpan({ text: relativeTime(proposal.created_at) });
    row.createDiv({ cls: "goclaw-review-reason", text: proposal.reason });
    const safety = row.createDiv({ cls: "goclaw-safety" });
    const shield = safety.createSpan();
    (0, import_obsidian2.setIcon)(shield, "shield-check");
    safety.createSpan({
      text: `\u57FA\u4E8E\u521B\u5EFA\u65F6\u7248\u672C \xB7 SHA ${shortHash(proposal.base_sha256)}`
    });
    const fields = this.reviewFields(row, "\u77E5\u8BC6\u63D0\u6848");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "\u62D2\u7EDD", "mod-warning", async () => {
      await this.client.rpc("knowledge.proposal.reject", {
        id: proposal.id,
        ...this.reviewPayload(fields, [`knowledge-proposal:${proposal.id}`], false)
      });
      new import_obsidian2.Notice("\u77E5\u8BC6\u63D0\u6848\u5DF2\u62D2\u7EDD");
      await this.renderApprovals();
    });
    this.actionButton(actions, "\u6279\u51C6", "mod-cta", async () => {
      await this.client.rpc("knowledge.proposal.approve", {
        id: proposal.id,
        ...this.reviewPayload(fields, [`knowledge-proposal:${proposal.id}`], true)
      });
      new import_obsidian2.Notice("\u77E5\u8BC6\u63D0\u6848\u5DF2\u5E94\u7528\u5230 Vault");
      await this.renderApprovals();
    });
    return row;
  }
  renderCatalogCandidate(record) {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    (0, import_obsidian2.setIcon)(title.createSpan(), "library");
    const titleText = title.createDiv();
    titleText.createEl("strong", { text: record.title });
    titleText.createSpan({
      text: `${record.kind} \xB7 v${record.version} \xB7 ${relativeTime(record.created_at)}`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: record.abstract || safeExcerpt(record.content, 240)
    });
    const safety = row.createDiv({ cls: "goclaw-safety" });
    (0, import_obsidian2.setIcon)(safety.createSpan(), "fingerprint");
    safety.createSpan({
      text: `${record.provenance.source_uri} \xB7 SHA ${shortHash(record.checksum)}`
    });
    const fields = this.reviewFields(row, "\u8BB0\u5FC6\u5019\u9009");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "\u62D2\u7EDD", "mod-warning", async () => {
      await this.client.rpc("memory.catalog.candidate.reject", {
        id: record.id,
        ...this.reviewPayload(fields, [`catalog:${record.id}@v${record.version}`], false)
      });
      new import_obsidian2.Notice("\u8BB0\u5FC6\u5019\u9009\u5DF2\u62D2\u7EDD\uFF0C\u672A\u8FDB\u5165\u68C0\u7D22\u4E0A\u4E0B\u6587");
      await this.renderApprovals();
    });
    this.actionButton(actions, "\u6279\u51C6\u5165\u85CF", "mod-cta", async () => {
      await this.client.rpc("memory.catalog.candidate.approve", {
        id: record.id,
        ...this.reviewPayload(fields, [`catalog:${record.id}@v${record.version}`], true)
      });
      new import_obsidian2.Notice("\u8BB0\u5FC6\u5DF2\u6279\u51C6\u5165\u85CF");
      await this.renderApprovals();
    });
    return row;
  }
  async renderMemory() {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "\u52A0\u8F7D\u8BB0\u5FC6\u76EE\u5F55");
    try {
      const [stats, active, pending] = await Promise.all([
        this.client.rpc("memory.catalog.status", {
          project_id: this.plugin.settings.projectId
        }),
        this.client.rpc("memory.catalog.list", {
          project_id: this.plugin.settings.projectId,
          status: "active",
          limit: 30
        }),
        this.client.rpc("memory.catalog.list", {
          project_id: this.plugin.settings.projectId,
          status: "pending",
          limit: 30
        })
      ]);
      if (this.activeTab !== "memory" || !this.bodyEl) return;
      target.empty();
      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "\u5728\u85CF", stats.by_status.active ?? 0);
      this.metric(summary, "\u5F85\u7F16\u76EE", stats.by_status.pending ?? 0, (stats.by_status.pending ?? 0) > 0);
      this.metric(summary, "\u5F85\u590D\u6838", stats.review_due, stats.review_due > 0);
      this.metric(
        summary,
        "\u51B2\u7A81",
        stats.unresolved_contradictions,
        stats.unresolved_contradictions > 0
      );
      const search = target.createDiv({ cls: "goclaw-memory-search" });
      const input = search.createEl("input", {
        type: "search",
        attr: {
          placeholder: "\u68C0\u7D22\u9879\u76EE\u51B3\u7B56\u3001\u7EA6\u675F\u3001\u4E8B\u5B9E\u6216\u504F\u597D\u2026",
          "aria-label": "Search catalog memory"
        }
      });
      const button = search.createEl("button", { text: "\u68C0\u7D22", cls: "mod-cta" });
      const resultsTarget = target.createDiv({ cls: "goclaw-section" });
      const renderResults = (results) => {
        resultsTarget.empty();
        const heading = resultsTarget.createDiv({ cls: "goclaw-section-heading" });
        heading.createEl("h3", { text: "\u68C0\u7D22\u7ED3\u679C" });
        heading.createSpan({ cls: "goclaw-count", text: String(results.length) });
        results.forEach((result) => resultsTarget.appendChild(this.renderCatalogRecord(result.record, result)));
        if (results.length === 0) {
          this.renderEmpty(resultsTarget, "search-x", "\u6CA1\u6709\u5339\u914D\u7684\u5DF2\u6279\u51C6\u8BB0\u5FC6", "\u5019\u9009\u8BB0\u5F55\u4E0D\u4F1A\u51FA\u73B0\u5728\u68C0\u7D22\u7ED3\u679C\u4E2D\u3002");
        }
      };
      const runSearch = async () => {
        const query = input.value.trim();
        if (!query) {
          renderResults(active.map((record) => ({
            record,
            score: 0.5,
            citation: `catalog:${record.id}@v${record.version}`,
            review_due: Boolean(record.review_at && Date.parse(record.review_at) <= Date.now()),
            expired: Boolean(record.expires_at && Date.parse(record.expires_at) <= Date.now())
          })));
          return;
        }
        button.disabled = true;
        try {
          const results = await this.client.rpc(
            "memory.catalog.search",
            {
              project_id: this.plugin.settings.projectId,
              query,
              include_shared: true,
              limit: 30
            }
          );
          renderResults(results);
        } finally {
          button.disabled = false;
        }
      };
      button.addEventListener("click", () => void runSearch());
      input.addEventListener("keydown", (event) => {
        if (event.key === "Enter") void runSearch();
      });
      const pendingSection = target.createDiv({ cls: "goclaw-section" });
      const pendingHeading = pendingSection.createDiv({ cls: "goclaw-section-heading" });
      pendingHeading.createEl("h3", { text: "\u5F85\u7F16\u76EE" });
      pendingHeading.createSpan({ cls: "goclaw-count", text: String(pending.length) });
      pending.slice(0, 8).forEach((record) => pendingSection.appendChild(this.renderCatalogCandidate(record)));
      if (pending.length === 0) {
        this.renderEmpty(pendingSection, "archive-restore", "\u6CA1\u6709\u5F85\u7F16\u76EE\u8BB0\u5F55", "Agent \u63D0\u6848\u4E0E Vault \u6444\u53D6\u7ED3\u679C\u4F1A\u5148\u8FDB\u5165\u8FD9\u91CC\u3002");
      }
      renderResults(active.map((record) => ({
        record,
        score: 0.5,
        citation: `catalog:${record.id}@v${record.version}`,
        review_due: Boolean(record.review_at && Date.parse(record.review_at) <= Date.now()),
        expired: Boolean(record.expires_at && Date.parse(record.expires_at) <= Date.now())
      })));
    } catch (error) {
      if (this.activeTab === "memory") this.renderError(target, error);
    }
  }
  renderCatalogRecord(record, result) {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    (0, import_obsidian2.setIcon)(title.createSpan(), result.review_due ? "clock-alert" : "book-check");
    const copy = title.createDiv();
    copy.createEl("strong", { text: record.title });
    copy.createSpan({
      text: `${record.kind} \xB7 v${record.version} \xB7 ${Math.round(result.score * 100)}%`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: safeExcerpt(record.content, 260)
    });
    const source = row.createDiv({ cls: "goclaw-catalog-source" });
    source.createSpan({ text: result.citation });
    if ((result.warnings ?? []).length > 0) {
      source.createSpan({
        cls: "is-warning",
        text: (result.warnings ?? []).join(" \xB7 ")
      });
    }
    if (result.review_due) {
      const fields = this.reviewFields(row, "\u8BB0\u5FC6\u590D\u6838");
      const actions = row.createDiv({ cls: "goclaw-actions" });
      this.actionButton(actions, "\u786E\u8BA4\u5E76\u5EF6\u957F 90 \u5929", "mod-cta", async () => {
        await this.client.rpc("memory.catalog.review.renew", {
          id: record.id,
          days: 90,
          ...this.reviewPayload(fields, [result.citation], true)
        });
        new import_obsidian2.Notice("\u8BB0\u5FC6\u590D\u6838\u5468\u671F\u5DF2\u66F4\u65B0");
        await this.renderMemory();
      });
      this.actionButton(actions, "\u9000\u85CF", "mod-warning", async () => {
        await this.client.rpc("memory.catalog.withdraw", {
          id: record.id,
          ...this.reviewPayload(fields, [result.citation], false)
        });
        new import_obsidian2.Notice("\u8BB0\u5F55\u5DF2\u9000\u85CF\uFF0C\u4E0D\u518D\u8FDB\u5165\u81EA\u52A8\u4E0A\u4E0B\u6587");
        await this.renderMemory();
      });
    }
    return row;
  }
  renderExperiment(experiment) {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    const icon = title.createSpan();
    (0, import_obsidian2.setIcon)(icon, "flask-conical");
    const text = title.createDiv();
    text.createEl("strong", { text: experiment.candidate_version });
    text.createSpan({ text: experiment.change_manifest.change_summary });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: `\u6839\u56E0\uFF1A${experiment.change_manifest.root_cause}`
    });
    const fields = this.reviewFields(row, "Harness \u5B9E\u9A8C");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "\u62D2\u7EDD", "mod-warning", async () => {
      await this.client.rpc("harness.experiment.reject", {
        id: experiment.id,
        ...this.reviewPayload(fields, [`experiment:${experiment.id}`], false)
      });
      await this.renderApprovals();
    });
    if (experiment.status === "validated") {
      this.actionButton(actions, "\u6279\u51C6\u5B9E\u9A8C", "mod-cta", async () => {
        await this.client.rpc("harness.experiment.approve", {
          id: experiment.id,
          ...this.reviewPayload(fields, [`experiment:${experiment.id}`], true)
        });
        await this.renderApprovals();
      });
    } else {
      this.actionButton(actions, "\u63D0\u5347\u4E3A\u5F53\u524D\u7248\u672C", "mod-cta", async () => {
        await this.client.rpc("harness.experiment.promote", {
          id: experiment.id,
          ...this.reviewPayload(fields, [`experiment:${experiment.id}`], true)
        });
        new import_obsidian2.Notice("Harness \u5DF2\u63D0\u5347\uFF1B\u540E\u7EED\u4F1A\u8BDD\u5C06\u4F7F\u7528\u65B0\u7248\u672C");
        await this.renderApprovals();
      });
    }
    return row;
  }
  renderOuroborosSeedApproval(session, seed) {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    (0, import_obsidian2.setIcon)(title.createSpan(), "scan-text");
    const info = title.createDiv();
    info.createEl("strong", { text: session.title });
    info.createSpan({
      text: `Seed ${shortHash(session.pending_seed_hash)} \xB7 ${relativeTime(session.updated_at)}`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: seed?.goal ?? safeExcerpt(session.raw_request, 180)
    });
    this.renderOuroborosSeedDetails(row, seed);
    const safety = row.createDiv({ cls: "goclaw-safety" });
    (0, import_obsidian2.setIcon)(safety.createSpan(), "shield-check");
    safety.createSpan({
      text: "\u6279\u51C6\u4EC5\u6388\u6743\u8FDB\u5165\u4EFB\u52A1\u7F16\u8BD1\uFF1B\u4E0D\u4EE3\u8868\u5B9E\u73B0\u6B63\u786E\uFF0C\u4E5F\u4E0D\u4F1A\u76F4\u63A5\u6267\u884C\u4EE3\u7801\u3002"
    });
    const fields = this.reviewFields(row, "Seed");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "\u62D2\u7EDD Seed", "mod-warning", async () => {
      await this.client.rpc("ouroboros.seed.reject", {
        id: session.id,
        ...this.reviewPayload(fields, [`seed:${session.pending_seed_hash ?? ""}`], false)
      });
      await this.renderApprovals();
    });
    if (seed) {
      this.actionButton(actions, "\u6279\u51C6 Seed", "mod-cta", async () => {
        await this.client.rpc("ouroboros.seed.approve", {
          id: session.id,
          ...this.reviewPayload(fields, [`seed:${session.pending_seed_hash ?? ""}`], true)
        });
        new import_obsidian2.Notice("Seed \u5DF2\u6279\u51C6\uFF0C\u53EF\u5728\u89C4\u683C\u9875\u7F16\u8BD1\u4EFB\u52A1");
        await this.renderApprovals();
      });
    }
    return row;
  }
  renderOuroborosEvolutionApproval(session, seed) {
    const proposal = session.pending_evolution;
    const row = document.createElement("article");
    row.addClass("goclaw-review-item");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    (0, import_obsidian2.setIcon)(title.createSpan(), "git-compare-arrows");
    const info = title.createDiv();
    info.createEl("strong", { text: `${session.title} \xB7 G${proposal?.candidate_generation ?? "?"}` });
    info.createSpan({
      text: `\u672C\u4F53\u76F8\u4F3C\u5EA6 ${Math.round((proposal?.ontology_similarity ?? 0) * 100)}%`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: safeExcerpt(proposal?.reasons?.join("\uFF1B"), 200) || "\u6A21\u578B\u63D0\u51FA\u4E86\u4E0B\u4E00\u4EE3\u5019\u9009 Seed\u3002"
    });
    this.renderOuroborosSeedDetails(row, seed);
    const safety = row.createDiv({ cls: "goclaw-safety" });
    (0, import_obsidian2.setIcon)(safety.createSpan(), proposal?.oscillation_detected ? "triangle-alert" : "shield-check");
    safety.createSpan({
      text: proposal?.oscillation_detected ? "\u68C0\u6D4B\u5230\u672C\u4F53\u632F\u8361\uFF0C\u5FC5\u987B\u4EBA\u5DE5\u51B3\u5B9A\u3002" : "\u5019\u9009\u5C1A\u672A\u751F\u6548\uFF1B\u6279\u51C6\u540E\u53EA\u5207\u6362 active Seed\uFF0C\u4E0D\u76F4\u63A5\u4FEE\u6539\u4EE3\u7801\u6216\u77E5\u8BC6\u5E93\u3002"
    });
    const fields = this.reviewFields(row, "\u6F14\u5316\u5019\u9009");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "\u62D2\u7EDD\u5019\u9009", "mod-warning", async () => {
      await this.client.rpc("ouroboros.evolution.reject", {
        id: session.id,
        ...this.reviewPayload(
          fields,
          [`seed:${proposal?.candidate_seed_hash ?? ""}`, `evolution:${proposal?.id ?? ""}`],
          false
        )
      });
      await this.renderApprovals();
    });
    if (seed) {
      this.actionButton(actions, "\u6279\u51C6\u5019\u9009", "mod-cta", async () => {
        await this.client.rpc("ouroboros.evolution.approve", {
          id: session.id,
          ...this.reviewPayload(
            fields,
            [`seed:${proposal?.candidate_seed_hash ?? ""}`, `evolution:${proposal?.id ?? ""}`],
            true
          )
        });
        new import_obsidian2.Notice("\u5019\u9009 Seed \u5DF2\u5207\u6362\u4E3A active\uFF1B\u9700\u91CD\u65B0\u7F16\u8BD1\u65B0\u4E00\u4EE3\u4EFB\u52A1");
        await this.renderApprovals();
      });
    }
    return row;
  }
  renderOuroborosSeedDetails(parent, seed) {
    if (!seed) {
      parent.createDiv({
        cls: "goclaw-review-reason",
        text: "Seed \u8BE6\u60C5\u8BFB\u53D6\u5931\u8D25\uFF1B\u4E0D\u8981\u5728\u672A\u6838\u5BF9\u5B8C\u6574\u89C4\u683C\u65F6\u6279\u51C6\u3002"
      });
      return;
    }
    const details = parent.createEl("details", { cls: "goclaw-seed-details" });
    details.createEl("summary", {
      text: `\u67E5\u770B\u5B8C\u6574\u5BA1\u6279\u6458\u8981 \xB7 G${seed.generation} \xB7 ${shortHash(seed.hash)}`
    });
    const body = details.createDiv();
    body.createEl("strong", { text: "\u7EA6\u675F" });
    const constraints = body.createEl("ul");
    seed.constraints.forEach((constraint) => constraints.createEl("li", { text: constraint }));
    body.createEl("strong", { text: "\u9A8C\u6536\u6807\u51C6\u4E0E\u547D\u4EE4" });
    const criteria = body.createEl("ol");
    seed.acceptance_criteria.forEach((criterion) => {
      const item = criteria.createEl("li");
      item.createSpan({ text: criterion.description });
      if (criterion.verify_command?.length) {
        item.createEl("code", { text: criterion.verify_command.join(" ") });
      }
    });
    body.createEl("strong", { text: "\u5907\u9009\u65B9\u6848\u4E0E\u4E0D\u884C\u52A8\u6210\u672C" });
    const alternatives = body.createEl("ul");
    (seed.alternatives ?? []).forEach((alternative) => {
      alternatives.createEl("li", {
        text: `${alternative.selected ? "\u5DF2\u9009" : "\u672A\u9009"} \xB7 ${alternative.title}\uFF1A${alternative.summary}`
      });
    });
    const inaction = body.createEl("ul");
    (seed.cost_of_inaction ?? []).forEach((cost) => inaction.createEl("li", { text: cost }));
    body.createEl("strong", { text: "\u53CD\u8BC1\u4E0E\u505C\u6B62\u6761\u4EF6" });
    const falsifiers = body.createEl("ul");
    (seed.falsifiers ?? []).forEach((falsifier) => falsifiers.createEl("li", {
      text: `${falsifier.criterion_id} \xB7 ${falsifier.condition} \xB7 \u8BC1\u636E\uFF1A${falsifier.evidence_required}`
    }));
    const kills = body.createEl("ul");
    (seed.kill_conditions ?? []).forEach((condition) => kills.createEl("li", {
      text: `${condition.id} \xB7 ${condition.metric} > ${condition.threshold} \u2192 ${condition.action}`
    }));
    body.createEl("strong", { text: "\u9884\u6CE8\u518C\u9884\u6D4B" });
    const predictions = body.createEl("ul");
    (seed.predictions ?? []).forEach((prediction) => predictions.createEl("li", {
      text: `${prediction.horizon} \xB7 ${Math.round(prediction.confidence * 100)}% \xB7 ${prediction.expected_outcome}`
    }));
    body.createEl("strong", { text: "\u6267\u884C\u8FB9\u754C" });
    body.createEl("p", {
      text: `${seed.scope.allowed_paths.join(", ")} \xB7 \u6700\u591A ${seed.scope.max_changed_files} \u6587\u4EF6 / ${seed.scope.max_changed_lines} \u884C \xB7 \u98CE\u9669 ${seed.risk.level} \xB7 \u4FEE\u590D ${seed.cost.max_repair_attempts} \u6B21`
    });
    body.createEl("strong", { text: "\u56DE\u6EDA" });
    body.createEl("p", { text: seed.risk.rollback });
  }
  renderDevelopmentApproval(task) {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "goclaw-dev-review");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    const icon = title.createSpan();
    (0, import_obsidian2.setIcon)(icon, "git-pull-request-draft");
    const text = title.createDiv();
    text.createEl("strong", { text: task.title });
    text.createSpan({ text: `${task.status} \xB7 rev ${task.compile.revision}` });
    row.createDiv({ cls: "goclaw-review-reason", text: safeExcerpt(task.goal.objective, 160) });
    const reviewLabels = {
      scenario: "\u573A\u666F",
      capacity: "\u5BB9\u91CF",
      risk: "\u98CE\u9669",
      cost: "\u6210\u672C"
    };
    const reviewGrid = row.createDiv({ cls: "goclaw-dev-review-grid" });
    Object.keys(reviewLabels).forEach((kind) => {
      const record = task.reviews[kind];
      const item = reviewGrid.createDiv({
        cls: `goclaw-dev-review-chip is-${record?.decision ?? "pending"}`
      });
      item.createSpan({ text: reviewLabels[kind] });
      item.createEl("strong", { text: record?.decision ?? "pending" });
    });
    const safety = row.createDiv({ cls: "goclaw-safety" });
    const shield = safety.createSpan();
    (0, import_obsidian2.setIcon)(shield, "shield-check");
    safety.createSpan({
      text: `\u6700\u591A ${task.scope.max_changed_files} \u6587\u4EF6 / ${task.scope.max_changed_lines} \u884C`
    });
    const fields = this.reviewFields(row, "\u5F00\u53D1\u4EFB\u52A1");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    if (task.status === "review_pending" || task.status === "blocked") {
      Object.keys(reviewLabels).filter((kind) => task.reviews[kind]?.decision !== "approved").forEach((kind) => {
        this.actionButton(actions, `\u6279\u51C6${reviewLabels[kind]}`, "", async () => {
          await this.client.rpc("dev.task.review", {
            id: task.id,
            kind,
            decision: "approved",
            ...this.reviewPayload(fields, [`task:${task.id}`, `review-kind:${kind}`], true)
          });
          await this.renderApprovals();
        });
      });
    }
    if (task.status === "ready_to_freeze") {
      this.actionButton(actions, "\u51BB\u7ED3\u6267\u884C\u5305", "mod-cta", async () => {
        await this.client.rpc("dev.task.freeze", {
          id: task.id,
          actor: this.plugin.actorId()
        });
        new import_obsidian2.Notice("\u5F00\u53D1\u4EFB\u52A1\u5DF2\u51BB\u7ED3\uFF0C\u53EF\u8FDB\u5165\u5F00\u53D1\u9875\u6267\u884C");
        await this.renderApprovals();
      });
    }
    if (task.status === "awaiting_acceptance") {
      this.actionButton(actions, "\u6700\u7EC8\u9A8C\u6536", "mod-cta", async () => {
        await this.client.rpc("dev.task.accept", {
          id: task.id,
          ...this.reviewPayload(fields, [`task:${task.id}`, task.last_evidence ?? ""], true)
        });
        new import_obsidian2.Notice("\u5F00\u53D1\u4EFB\u52A1\u5DF2\u901A\u8FC7\u6700\u7EC8\u9A8C\u6536");
        await this.renderApprovals();
      });
    }
    return row;
  }
  async renderDevelopment() {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "\u52A0\u8F7D\u5F00\u53D1\u4EFB\u52A1");
    try {
      const tasks = await this.client.rpc("dev.tasks", {
        project_id: this.plugin.settings.projectId
      });
      if (this.activeTab !== "development") return;
      target.empty();
      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "\u4EFB\u52A1", tasks.length);
      this.metric(summary, "\u6267\u884C\u4E2D", tasks.filter((task) => ["running", "checking"].includes(task.status)).length);
      this.metric(summary, "\u5F85\u9A8C\u6536", tasks.filter((task) => task.status === "awaiting_acceptance").length);
      const list = target.createDiv({ cls: "goclaw-dev-list" });
      tasks.forEach((task) => {
        const card = list.createEl("article", { cls: `goclaw-dev-card is-${task.status}` });
        const heading = card.createDiv({ cls: "goclaw-dev-card-heading" });
        const info = heading.createDiv();
        info.createEl("strong", { text: task.title });
        info.createSpan({ text: `${task.id} \xB7 rev ${task.compile.revision}` });
        heading.createSpan({ cls: `goclaw-state is-${task.status}`, text: task.status });
        card.createEl("p", { text: safeExcerpt(task.goal.objective, 150) });
        const meta = card.createDiv({ cls: "goclaw-dev-meta" });
        meta.createSpan({ text: task.branch || task.compile.base_ref });
        meta.createSpan({ text: `${task.scope.max_changed_files} \u6587\u4EF6\u4E0A\u9650` });
        meta.createSpan({ text: relativeTime(task.updated_at) });
        if (task.last_gate) {
          const gate = card.createDiv({
            cls: `goclaw-safety${task.last_gate.passed ? "" : " is-error"}`
          });
          const gateIcon = gate.createSpan();
          (0, import_obsidian2.setIcon)(gateIcon, task.last_gate.passed ? "badge-check" : "circle-x");
          gate.createSpan({
            text: task.last_gate.passed ? "DoneGate \u5DF2\u901A\u8FC7" : safeExcerpt(task.last_gate.reasons?.join("\uFF1B"), 120)
          });
        }
        const actions = card.createDiv({ cls: "goclaw-actions" });
        if (task.status === "frozen") {
          this.actionButton(actions, "\u5F00\u59CB\u6267\u884C", "mod-cta", async () => {
            await this.client.rpc("dev.task.enqueue", {
              task_id: task.id,
              capabilities: ["codex"]
            });
            new import_obsidian2.Notice("\u51BB\u7ED3\u7248\u672C\u5DF2\u8FDB\u5165\u5DE5\u4F5C\u7AD9\u961F\u5217");
            await this.renderDevelopment();
          });
        }
        if (["repair_pending", "failed"].includes(task.status)) {
          this.actionButton(actions, "\u521B\u5EFA\u4FEE\u590D\u7248\u672C", "mod-cta", async () => {
            await this.client.rpc("dev.task.revise", {
              id: task.id,
              expected_revision: task.compile.revision,
              reason: "Workstation DoneGate \u672A\u901A\u8FC7\uFF0C\u521B\u5EFA\u65B0 revision \u5E76\u91CD\u65B0\u5C65\u884C\u56DB\u7C7B\u8BC4\u5BA1\u3002"
            });
            new import_obsidian2.Notice("\u5DF2\u521B\u5EFA\u65B0 revision\uFF1B\u91CD\u65B0\u5B8C\u6210\u56DB\u7C7B\u8BC4\u5BA1\u3001\u51BB\u7ED3\u5E76\u5165\u961F");
            await this.renderDevelopment();
          });
        }
      });
      if (tasks.length === 0) {
        this.renderEmpty(target, "git-branch-plus", "\u8FD8\u6CA1\u6709\u5F00\u53D1\u4EFB\u52A1", "\u5148\u4F7F\u7528 goclaw dev create \u7F16\u8BD1\u4EFB\u52A1\u5951\u7EA6\u3002");
      }
    } catch (error) {
      if (this.activeTab === "development") this.renderError(target, error);
    }
  }
  async renderTeam() {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    const project = { project_id: this.plugin.settings.projectId };
    this.renderLoading(target, "\u52A0\u8F7D\u56E2\u961F\u63A7\u5236\u9762");
    const [
      membersResult,
      workResult,
      issuesResult,
      runnersResult,
      policyResult,
      docsResult,
      componentsResult
    ] = await Promise.allSettled([
      this.client.controlRpc(TEAM_CONTROL_RPC.members, project),
      this.client.controlRpc(TEAM_CONTROL_RPC.workItems, { ...project, limit: 40 }),
      this.client.controlRpc(TEAM_CONTROL_RPC.issues, { ...project, limit: 40 }),
      this.client.controlRpc(TEAM_CONTROL_RPC.runners, project),
      this.client.controlRpc(TEAM_CONTROL_RPC.policy, project),
      this.client.controlRpc(TEAM_CONTROL_RPC.docs, { ...project, limit: 8 }),
      this.client.controlRpc(TEAM_CONTROL_RPC.components, { ...project, limit: 8 })
    ]);
    if (this.activeTab !== "team") return;
    target.empty();
    const members = membersResult.status === "fulfilled" ? collectionItems(membersResult.value) : [];
    const work = workResult.status === "fulfilled" ? collectionItems(workResult.value) : [];
    const issues = issuesResult.status === "fulfilled" ? collectionItems(issuesResult.value) : [];
    const runners = runnersResult.status === "fulfilled" ? collectionItems(runnersResult.value) : [];
    const policy = policyResult.status === "fulfilled" ? policyResult.value : null;
    const docs = docsResult.status === "fulfilled" ? docsResult.value : null;
    const components = componentsResult.status === "fulfilled" ? componentsResult.value : null;
    const intro = target.createDiv({ cls: "goclaw-team-intro" });
    const introIcon = intro.createSpan();
    (0, import_obsidian2.setIcon)(introIcon, "users-round");
    const introText = intro.createDiv();
    introText.createEl("strong", { text: "\u56E2\u961F\u53EA\u8BFB\u63A7\u5236\u53F0" });
    introText.createEl("p", {
      text: "\u6210\u5458\u3001\u4EFB\u52A1\u3001Bug\u3001Runner\u3001\u7B56\u7565\u3001\u6587\u6863\u4E0E\u7EC4\u4EF6\u5747\u7531 Gateway \u6309\u5F53\u524D\u9879\u76EE\u6388\u6743\u540E\u8FD4\u56DE\u3002"
    });
    const activeWork = work.filter((item) => ["ready", "in_progress", "in_review", "blocked"].includes(item.status)).length;
    const openIssues = issues.filter((issue) => !["resolved", "closed"].includes(issue.status)).length;
    const onlineRunners = runners.filter((runner) => ["online", "busy", "draining"].includes(runner.status)).length;
    const summary = target.createDiv({ cls: "goclaw-team-summary" });
    this.metric(summary, "\u6210\u5458", members.length);
    this.metric(summary, "\u6D3B\u52A8\u4EFB\u52A1", activeWork, work.some((item) => item.status === "blocked"));
    this.metric(summary, "\u672A\u5173\u95ED Bug", openIssues, issues.some((issue) => ["critical", "high"].includes(issue.severity) && !["resolved", "closed"].includes(issue.status)));
    this.metric(summary, "\u5728\u7EBF Runner", onlineRunners, runners.some((runner) => {
      const leaseTone = leaseState(runner.lease?.expires_at).tone;
      return runner.status !== "offline" && (!runner.lease || leaseTone === "warning" || leaseTone === "danger");
    }));
    const failures = [];
    [
      ["\u6210\u5458\u8D1F\u8F7D", membersResult],
      ["\u9879\u76EE\u4EFB\u52A1", workResult],
      ["Bug", issuesResult],
      ["Runner", runnersResult],
      ["\u7B56\u7565", policyResult],
      ["\u6587\u6863", docsResult],
      ["\u7EC4\u4EF6", componentsResult]
    ].forEach(([label, result]) => {
      if (result.status === "rejected") failures.push([label, result]);
    });
    if (failures.length > 0) {
      const warning = target.createDiv({ cls: "goclaw-team-warning" });
      const warningIcon = warning.createSpan();
      (0, import_obsidian2.setIcon)(warningIcon, "triangle-alert");
      const warningText = warning.createDiv();
      warningText.createEl("strong", { text: `${failures.length} \u4E2A\u6A21\u5757\u6682\u4E0D\u53EF\u7528` });
      warningText.createEl("p", {
        text: failures.map(
          ([label, result]) => `${label}\uFF1A${result.reason instanceof Error ? result.reason.message : String(result.reason)}`
        ).join("\uFF1B")
      });
    }
    const memberNames = new Map(members.map((member) => [member.id, member.display_name]));
    this.renderTeamMembers(target, members, membersResult.status === "rejected");
    this.renderTeamWork(target, work, memberNames, workResult.status === "rejected");
    this.renderTeamIssues(target, issues, memberNames, issuesResult.status === "rejected");
    this.renderTeamRunners(target, runners, memberNames, runnersResult.status === "rejected");
    this.renderTeamPolicy(target, policy, policyResult.status === "rejected");
    this.renderTeamDocs(target, docs, docsResult.status === "rejected");
    this.renderTeamComponents(target, components, componentsResult.status === "rejected");
  }
  renderTeamMembers(parent, members, failed) {
    const list = this.teamSection(parent, "\u6210\u5458\u8D1F\u8F7D", members.length);
    if (failed) return this.renderTeamModuleError(list, "\u6210\u5458\u8D1F\u8F7D");
    members.forEach((member) => {
      const card = list.createEl("article", { cls: "goclaw-team-member" });
      card.createSpan({
        cls: "goclaw-team-avatar",
        text: displayInitial(member.display_name)
      });
      const body = card.createDiv({ cls: "goclaw-team-member-body" });
      const heading = body.createDiv({ cls: "goclaw-team-card-heading" });
      const name = heading.createDiv();
      name.createEl("strong", { text: member.display_name || member.id });
      name.createSpan({
        text: [
          member.role,
          member.business_domains?.slice(0, 2).join(" / ")
        ].filter(Boolean).join(" \xB7 ") || member.id
      });
      const state = memberState(member.status ?? "offline");
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
      const utilization = clampPercent(member.capacity?.utilization_percent);
      const load = body.createDiv({ cls: "goclaw-team-load" });
      const loadLabel = load.createDiv();
      loadLabel.createSpan({
        text: `${member.capacity?.active_work ?? 0} \u8FDB\u884C\u4E2D \xB7 ${member.capacity?.queued_work ?? 0} \u6392\u961F`
      });
      loadLabel.createEl("strong", { text: `${utilization}%` });
      const track = load.createDiv({ cls: "goclaw-team-load-track" });
      track.createDiv({ cls: "goclaw-team-load-fill" }).style.setProperty("width", `${utilization}%`);
      if ((member.capacity?.blocked_work ?? 0) > 0) {
        body.createDiv({
          cls: "goclaw-team-note is-danger",
          text: `${member.capacity?.blocked_work} \u4E2A\u4EFB\u52A1\u53D7\u963B`
        });
      } else if (member.last_seen_at) {
        body.createDiv({
          cls: "goclaw-team-note",
          text: `\u6700\u540E\u6D3B\u52A8 ${relativeTime(member.last_seen_at)}`
        });
      }
    });
    if (members.length === 0) {
      this.renderTeamEmpty(list, "\u5C1A\u672A\u767B\u8BB0\u9879\u76EE\u6210\u5458");
    }
  }
  renderTeamWork(parent, work, memberNames, failed) {
    const active = work.filter((item) => item.status !== "done" && item.status !== "cancelled");
    const list = this.teamSection(parent, "\u9879\u76EE\u4EFB\u52A1", active.length);
    if (failed) return this.renderTeamModuleError(list, "\u9879\u76EE\u4EFB\u52A1");
    active.slice(0, 10).forEach((item) => {
      const card = list.createEl("article", { cls: "goclaw-team-row" });
      const heading = card.createDiv({ cls: "goclaw-team-card-heading" });
      const text = heading.createDiv();
      text.createEl("strong", { text: item.title });
      text.createSpan({
        text: [
          item.kind,
          item.business_domain,
          item.assignee_id ? memberNames.get(item.assignee_id) ?? item.assignee_id : "\u672A\u5206\u914D"
        ].filter(Boolean).join(" \xB7 ")
      });
      const state = workState(item.status);
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
      const links = [
        item.id,
        item.task_id,
        item.issue_id,
        ...(item.source_refs ?? []).slice(0, 2)
      ].filter(Boolean);
      card.createDiv({ cls: "goclaw-team-links", text: links.join(" \xB7 ") });
      if (item.blocked_reason) {
        card.createDiv({
          cls: "goclaw-team-note is-danger",
          text: safeExcerpt(item.blocked_reason, 120)
        });
      } else if (item.updated_at) {
        card.createDiv({ cls: "goclaw-team-note", text: relativeTime(item.updated_at) });
      }
    });
    if (active.length > 10) this.renderTeamMore(list, active.length - 10);
    if (active.length === 0) this.renderTeamEmpty(list, "\u5F53\u524D\u6CA1\u6709\u6D3B\u52A8\u4EFB\u52A1");
  }
  renderTeamIssues(parent, issues, memberNames, failed) {
    const open = issues.filter((issue) => issue.status !== "resolved" && issue.status !== "closed");
    const list = this.teamSection(parent, "Bug \u72B6\u6001", open.length);
    if (failed) return this.renderTeamModuleError(list, "Bug");
    open.slice(0, 8).forEach((issue) => {
      const card = list.createEl("article", { cls: "goclaw-team-row" });
      const heading = card.createDiv({ cls: "goclaw-team-card-heading" });
      const text = heading.createDiv();
      text.createEl("strong", { text: issue.title });
      text.createSpan({
        text: `${issue.id} \xB7 ${issue.owner_id ? memberNames.get(issue.owner_id) ?? issue.owner_id : "\u672A\u5206\u914D"}`
      });
      const status = issueState(issue.status);
      heading.createSpan({ cls: teamStateClass(status), text: status.label });
      const metadata = card.createDiv({ cls: "goclaw-team-meta-line" });
      const severity = severityState(issue.severity);
      metadata.createSpan({ cls: teamStateClass(severity), text: `\u4E25\u91CD\u5EA6 ${severity.label}` });
      if (issue.work_item_id) metadata.createSpan({ text: issue.work_item_id });
      if (issue.regression_case_id) metadata.createSpan({ text: `\u56DE\u5F52 ${issue.regression_case_id}` });
      if (issue.updated_at) {
        card.createDiv({ cls: "goclaw-team-note", text: relativeTime(issue.updated_at) });
      }
    });
    if (open.length > 8) this.renderTeamMore(list, open.length - 8);
    if (open.length === 0) this.renderTeamEmpty(list, "\u6CA1\u6709\u672A\u5173\u95ED Bug");
  }
  renderTeamRunners(parent, runners, memberNames, failed) {
    const list = this.teamSection(parent, "Runner \u5728\u7EBF\u4E0E\u79DF\u7EA6", runners.length);
    if (failed) return this.renderTeamModuleError(list, "Runner");
    runners.forEach((runner) => {
      const card = list.createEl("article", { cls: "goclaw-team-runner" });
      const status = runnerState(runner.status);
      card.createSpan({ cls: `goclaw-team-runner-dot is-${status.tone}` });
      const body = card.createDiv();
      body.createEl("strong", { text: runner.display_name || runner.id });
      body.createSpan({
        text: [
          runner.member_id ? memberNames.get(runner.member_id) ?? runner.member_id : "",
          runner.current_work_id,
          runner.capabilities?.slice(0, 2).join(" / ")
        ].filter(Boolean).join(" \xB7 ") || "\u672A\u7ED1\u5B9A\u4EFB\u52A1"
      });
      const badges = card.createDiv({ cls: "goclaw-team-runner-states" });
      badges.createSpan({ cls: teamStateClass(status), text: status.label });
      const lease = leaseState(runner.lease?.expires_at);
      badges.createSpan({ cls: teamStateClass(lease), text: lease.label });
      if (runner.lease?.expires_at) {
        badges.setAttribute("title", `\u79DF\u7EA6\u5230\u671F\uFF1A${runner.lease.expires_at}`);
      }
    });
    if (runners.length === 0) this.renderTeamEmpty(list, "\u5F53\u524D\u6CA1\u6709\u767B\u8BB0\u7684 Runner");
  }
  renderTeamPolicy(parent, policy, failed) {
    const list = this.teamSection(parent, "\u751F\u6548\u7B56\u7565", policy?.layers?.length ?? 0);
    if (failed) return this.renderTeamModuleError(list, "\u7B56\u7565");
    if (!policy) return this.renderTeamEmpty(list, "\u5F53\u524D\u9879\u76EE\u6CA1\u6709\u7B56\u7565\u72B6\u6001");
    const facts = list.createDiv({ cls: "goclaw-team-facts" });
    this.fact(facts, "\u6709\u6548\u7248\u672C", policy.effective_version || "\u672A\u9501\u5B9A");
    this.fact(facts, "\u5408\u89C4", policy.compliant ? "\u901A\u8FC7" : "\u672A\u901A\u8FC7");
    this.fact(facts, "\u6F02\u79FB", String(policy.drift_count ?? 0));
    this.fact(facts, "\u68C0\u67E5", policy.checked_at ? relativeTime(policy.checked_at) : "\u672A\u77E5");
    (policy.layers ?? []).forEach((layer) => {
      const row = list.createDiv({ cls: "goclaw-team-policy-layer" });
      row.createSpan({ text: layer.scope });
      row.createEl("strong", { text: `${layer.id}@${layer.version}` });
      const state = layer.compliant === false ? { label: "\u6F02\u79FB", tone: "danger" } : { label: "\u4E00\u81F4", tone: "success" };
      row.createSpan({ cls: teamStateClass(state), text: state.label });
    });
    (policy.violations ?? []).slice(0, 5).forEach((violation) => {
      list.createDiv({
        cls: `goclaw-team-note is-${violation.severity === "warning" ? "warning" : "danger"}`,
        text: `${violation.code} \xB7 ${violation.message}`
      });
    });
  }
  renderTeamDocs(parent, docs, failed) {
    const list = this.teamSection(parent, "\u65B9\u6848\u6587\u6863", docs?.total ?? 0);
    if (failed) return this.renderTeamModuleError(list, "\u6587\u6863");
    if (!docs) return this.renderTeamEmpty(list, "\u5F53\u524D\u9879\u76EE\u6CA1\u6709\u6587\u6863\u6982\u89C8");
    const facts = list.createDiv({ cls: "goclaw-team-facts" });
    this.fact(facts, "\u5DF2\u6279\u51C6", String(docs.approved ?? 0));
    this.fact(facts, "\u5F85\u590D\u6838", String(docs.review_due ?? 0));
    this.fact(facts, "\u9648\u65E7", String(docs.stale ?? 0));
    this.fact(facts, "\u672A\u5173\u8054\u4EFB\u52A1", String(docs.unlinked ?? 0));
    (docs.items ?? []).slice(0, 6).forEach((document2) => this.renderTeamDocument(list, document2));
    if ((docs.items?.length ?? 0) === 0) {
      this.renderTeamEmpty(list, "\u6CA1\u6709\u8FD4\u56DE\u6700\u8FD1\u6587\u6863");
    }
  }
  renderTeamDocument(parent, document2) {
    const row = parent.createEl("article", { cls: "goclaw-team-row" });
    const heading = row.createDiv({ cls: "goclaw-team-card-heading" });
    const text = heading.createDiv();
    text.createEl("strong", { text: document2.title || document2.path });
    text.createSpan({
      text: [
        document2.kind,
        document2.owner_id,
        `${document2.linked_work_ids?.length ?? 0} \u4E2A\u4EFB\u52A1\u5173\u8054`
      ].filter(Boolean).join(" \xB7 ")
    });
    if (document2.status) {
      const state = document2.status === "approved" ? { label: "\u5DF2\u6279\u51C6", tone: "success" } : document2.status === "stale" ? { label: "\u9648\u65E7", tone: "danger" } : { label: document2.status === "review" ? "\u8BC4\u5BA1\u4E2D" : "\u8349\u7A3F", tone: "warning" };
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
    }
    row.createDiv({ cls: "goclaw-team-links", text: document2.path });
  }
  renderTeamComponents(parent, components, failed) {
    const list = this.teamSection(parent, "\u5171\u4EAB\u7EC4\u4EF6", components?.total ?? 0);
    if (failed) return this.renderTeamModuleError(list, "\u7EC4\u4EF6");
    if (!components) return this.renderTeamEmpty(list, "\u5F53\u524D\u9879\u76EE\u6CA1\u6709\u7EC4\u4EF6\u6982\u89C8");
    const facts = list.createDiv({ cls: "goclaw-team-facts" });
    this.fact(facts, "\u53EF\u590D\u7528", String(components.reusable ?? 0));
    this.fact(facts, "\u5F85\u8BC4\u5BA1", String(components.pending_review ?? 0));
    this.fact(facts, "\u5DF2\u5F03\u7528", String(components.deprecated ?? 0));
    this.fact(facts, "\u603B\u8BA1", String(components.total ?? 0));
    (components.items ?? []).slice(0, 6).forEach((component) => this.renderTeamComponent(list, component));
    if ((components.items?.length ?? 0) === 0) {
      this.renderTeamEmpty(list, "\u6CA1\u6709\u8FD4\u56DE\u63A8\u8350\u7EC4\u4EF6");
    }
  }
  renderTeamComponent(parent, component) {
    const row = parent.createEl("article", { cls: "goclaw-team-row" });
    const heading = row.createDiv({ cls: "goclaw-team-card-heading" });
    const text = heading.createDiv();
    text.createEl("strong", { text: component.name });
    text.createSpan({
      text: [
        component.kind,
        component.version,
        component.owner_id
      ].filter(Boolean).join(" \xB7 ") || component.id
    });
    if (component.status) {
      const state = component.status === "active" ? { label: "\u53EF\u7528", tone: "success" } : component.status === "deprecated" ? { label: "\u5F03\u7528", tone: "danger" } : { label: "\u5B9E\u9A8C", tone: "warning" };
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
    }
    row.createDiv({
      cls: "goclaw-team-links",
      text: `${component.id} \xB7 \u5DF2\u590D\u7528 ${component.reuse_count ?? 0} \u6B21`
    });
  }
  teamSection(parent, title, count) {
    const section = parent.createDiv({ cls: "goclaw-section goclaw-team-section" });
    const heading = section.createDiv({ cls: "goclaw-section-heading" });
    heading.createEl("h3", { text: title });
    heading.createSpan({ cls: "goclaw-count", text: String(count) });
    return section.createDiv({ cls: "goclaw-team-list" });
  }
  renderTeamModuleError(parent, label) {
    parent.createDiv({
      cls: "goclaw-team-empty is-error",
      text: `${label}\u63A5\u53E3\u6682\u4E0D\u53EF\u7528\uFF1B\u5176\u4ED6\u56E2\u961F\u6A21\u5757\u4ECD\u53EF\u7EE7\u7EED\u67E5\u770B\u3002`
    });
  }
  renderTeamEmpty(parent, text) {
    parent.createDiv({ cls: "goclaw-team-empty", text });
  }
  renderTeamMore(parent, count) {
    parent.createDiv({ cls: "goclaw-team-more", text: `\u8FD8\u6709 ${count} \u9879\uFF0C\u8BF7\u5728\u5BF9\u5E94\u7CFB\u7EDF\u67E5\u770B\u5B8C\u6574\u5217\u8868` });
  }
  async renderProgress() {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "\u52A0\u8F7D\u8FD0\u884C\u8F68\u8FF9");
    try {
      const traces = await this.client.rpc("harness.traces", {
        project_id: this.plugin.settings.projectId,
        limit: 40
      });
      if (this.activeTab !== "progress") return;
      target.empty();
      const complete = traces.filter((trace) => trace.status === "completed").length;
      const failed = traces.filter((trace) => trace.status !== "completed").length;
      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "\u8FD0\u884C", traces.length);
      this.metric(summary, "\u5B8C\u6210", complete);
      this.metric(summary, "\u5F02\u5E38", failed, failed > 0);
      const list = target.createDiv({ cls: "goclaw-run-list" });
      traces.forEach((trace) => {
        const row = list.createDiv({ cls: `goclaw-run is-${trace.status}` });
        const dot = row.createSpan({ cls: "goclaw-run-dot" });
        dot.setAttribute("aria-label", trace.status);
        const content = row.createDiv({ cls: "goclaw-run-content" });
        content.createEl("strong", {
          text: safeExcerpt(trace.output || trace.input || trace.error, 90) || "\u65E0\u6458\u8981"
        });
        content.createSpan({
          text: `${trace.harness_version || "unversioned"} \xB7 ${trace.duration_ms} ms \xB7 ${relativeTime(trace.started_at)}`
        });
      });
      if (traces.length === 0) {
        this.renderEmpty(target, "activity", "\u8FD8\u6CA1\u6709\u8FD0\u884C\u8F68\u8FF9", "\u4ECE Obsidian \u6216\u98DE\u4E66\u53D1\u8D77\u4E00\u6B21\u9879\u76EE\u4F1A\u8BDD\u5373\u53EF\u751F\u6210\u3002");
      }
    } catch (error) {
      if (this.activeTab === "progress") this.renderError(target, error);
    }
  }
  async renderHarness() {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "\u52A0\u8F7D Harness \u72B6\u6001");
    try {
      const [status, experiments] = await Promise.all([
        this.client.rpc("harness.status"),
        this.client.rpc("harness.experiments")
      ]);
      if (this.activeTab !== "harness") return;
      target.empty();
      const hero = target.createDiv({ cls: "goclaw-harness-hero" });
      hero.createSpan({ text: "\u5F53\u524D\u7248\u672C" });
      hero.createEl("strong", { text: status.active.version });
      hero.createEl("p", { text: status.manifest.description || status.manifest.name });
      const facts = target.createDiv({ cls: "goclaw-facts" });
      this.fact(facts, "\u6A21\u578B", status.manifest.model_profile || "\u914D\u7F6E\u9ED8\u8BA4\u503C");
      this.fact(facts, "Golden \u95E8\u69DB", `${Math.round(status.manifest.minimum_golden * 100)}%`);
      this.fact(facts, "Holdout \u95E8\u69DB", `${Math.round(status.manifest.minimum_holdout * 100)}%`);
      this.fact(facts, "\u7EC4\u4EF6", String(Object.keys(status.manifest.components).length));
      const section = target.createDiv({ cls: "goclaw-section" });
      section.createEl("h3", { text: "\u6700\u8FD1\u5B9E\u9A8C" });
      experiments.slice(0, 8).forEach((experiment) => {
        const row = section.createDiv({ cls: "goclaw-experiment-row" });
        const info = row.createDiv();
        info.createEl("strong", { text: experiment.candidate_version });
        info.createSpan({ text: safeExcerpt(experiment.change_manifest.change_summary, 80) });
        row.createSpan({ cls: `goclaw-state is-${experiment.status}`, text: experiment.status });
      });
      if (status.active.previous_version) {
        const fields = this.reviewFields(target, "Harness \u56DE\u6EDA");
        this.actionButton(target, `\u56DE\u6EDA\u5230 ${status.active.previous_version}`, "", async () => {
          await this.client.rpc("harness.rollback", {
            ...this.reviewPayload(
              fields,
              [`harness:${status.active.version}`, `rollback:${status.active.previous_version}`],
              true
            )
          });
          new import_obsidian2.Notice("Harness \u5DF2\u56DE\u6EDA");
          await this.renderHarness();
        });
      }
    } catch (error) {
      if (this.activeTab === "harness") this.renderError(target, error);
    }
  }
  actionButton(parent, label, cls, action) {
    const button = parent.createEl("button", { text: label, cls });
    button.addEventListener("click", () => {
      button.disabled = true;
      void action().catch((error) => {
        new import_obsidian2.Notice(error instanceof Error ? error.message : String(error));
      }).finally(() => {
        button.disabled = false;
      });
    });
  }
  reviewFields(parent, subject) {
    const rationale = parent.createEl("textarea", {
      cls: "goclaw-review-comment",
      attr: {
        rows: "2",
        placeholder: `\u5FC5\u586B\uFF1A${subject}\u51B3\u7B56\u4F9D\u636E\u3001\u5DF2\u6838\u5BF9\u8BC1\u636E\u4E0E\u8FB9\u754C\u2026`,
        "aria-label": `${subject} review rationale`
      }
    });
    const counterargument = parent.createEl("textarea", {
      cls: "goclaw-review-comment",
      attr: {
        rows: "2",
        placeholder: "\u6279\u51C6\u65F6\u5FC5\u586B\uFF1A\u8FD9\u4E2A\u51B3\u5B9A\u6700\u53EF\u80FD\u9519\u5728\u54EA\u91CC\uFF1F",
        "aria-label": `${subject} strongest counterargument`
      }
    });
    return { rationale, counterargument };
  }
  reviewPayload(fields, evidenceRefs, approval) {
    const rationale = fields.rationale.value.trim();
    const counterargument = fields.counterargument.value.trim();
    if (!rationale) throw new Error("\u8BF7\u586B\u5199\u53EF\u5BA1\u8BA1\u7684\u51B3\u7B56\u4F9D\u636E");
    if (approval && !counterargument) {
      throw new Error("\u6279\u51C6\u524D\u8BF7\u586B\u5199\u6700\u5F3A\u53CD\u5BF9\u7406\u7531\uFF0C\u907F\u514D\u786E\u8BA4\u504F\u5DEE");
    }
    return this.plugin.governanceParams(
      rationale,
      counterargument,
      evidenceRefs.filter(Boolean)
    );
  }
  metric(parent, label, value, warning = false) {
    const item = parent.createDiv({ cls: `goclaw-metric${warning ? " is-warning" : ""}` });
    item.createEl("strong", { text: String(value) });
    item.createSpan({ text: label });
  }
  fact(parent, label, value) {
    const item = parent.createDiv({ cls: "goclaw-fact" });
    item.createSpan({ text: label });
    item.createEl("strong", { text: value });
  }
  renderLoading(parent, label) {
    parent.empty();
    const loading = parent.createDiv({ cls: "goclaw-loading" });
    const icon = loading.createSpan();
    (0, import_obsidian2.setIcon)(icon, "loader-circle");
    loading.createSpan({ text: label });
  }
  renderError(parent, error) {
    parent.empty();
    this.renderEmpty(
      parent,
      "circle-alert",
      "\u52A0\u8F7D\u5931\u8D25",
      error instanceof Error ? error.message : String(error)
    );
  }
  renderEmpty(parent, iconName, title, description) {
    const empty = parent.createDiv({ cls: "goclaw-empty" });
    const icon = empty.createSpan();
    (0, import_obsidian2.setIcon)(icon, iconName);
    empty.createEl("strong", { text: title });
    empty.createEl("p", { text: description });
  }
  updateStatus(state, detail) {
    if (!this.statusEl) return;
    const status = this.statusEl.parentElement;
    status?.removeClass("is-connected", "is-error", "is-connecting");
    if (state === "connected") {
      status?.addClass("is-connected");
      this.statusEl.setText("\u5DF2\u8FDE\u63A5");
    } else if (state === "connecting") {
      status?.addClass("is-connecting");
      this.statusEl.setText("\u8FDE\u63A5\u4E2D");
    } else if (state === "error") {
      status?.addClass("is-error");
      this.statusEl.setText("\u8FDE\u63A5\u9519\u8BEF");
    } else {
      this.statusEl.setText("\u672A\u8FDE\u63A5");
    }
    this.statusEl.setAttribute("title", detail ?? "");
  }
};

// src/main.ts
var GoClawPlugin = class extends import_obsidian3.Plugin {
  settings = DEFAULT_SETTINGS;
  client = new GatewayClient();
  async onload() {
    await this.loadSettings();
    this.registerView(VIEW_TYPE_GOCLAW, (leaf) => new GoClawView(leaf, this, this.client));
    this.addRibbonIcon("bot-message-square", "Open GoClaw", () => {
      void this.activateView();
    });
    this.addCommand({
      id: "open-project-console",
      name: "Open project console",
      callback: () => void this.activateView()
    });
    this.addCommand({
      id: "reconnect-gateway",
      name: "Reconnect gateway",
      callback: () => void this.connectGateway()
    });
    this.addSettingTab(new GoClawSettingTab(this.app, this));
    if (this.settings.autoConnect) {
      this.app.workspace.onLayoutReady(() => void this.connectGateway(false));
    }
  }
  onunload() {
    this.client.disconnect();
  }
  async loadSettings() {
    this.settings = Object.assign({}, DEFAULT_SETTINGS, await this.loadData());
  }
  async saveSettings() {
    await this.saveData(this.settings);
    this.app.workspace.getLeavesOfType(VIEW_TYPE_GOCLAW).forEach((leaf) => leaf.view.refresh());
  }
  async connectGateway(showNotice = true) {
    const secretStorage = getSecretStorage(this.app);
    const token = secretStorage?.getSecret(this.settings.secretKey) ?? "";
    const userToken = secretStorage?.getSecret(this.settings.userSecretKey) ?? "";
    try {
      await this.client.connect(this.settings.gatewayUrl, token, userToken);
      if (showNotice) new import_obsidian3.Notice("GoClaw Gateway \u5DF2\u8FDE\u63A5");
    } catch (error) {
      if (showNotice) {
        new import_obsidian3.Notice(error instanceof Error ? error.message : String(error));
      }
    }
  }
  governanceParams(rationale, counterargument = "", evidenceRefs = []) {
    const secretStorage = getSecretStorage(this.app);
    return {
      reviewer_id: this.settings.reviewerId,
      reviewer_token: secretStorage?.getSecret(this.settings.reviewerSecretKey) ?? "",
      rationale,
      counterargument,
      evidence_refs: evidenceRefs
    };
  }
  actorId() {
    return this.settings.reviewerId || "obsidian-user";
  }
  async activateView() {
    const { workspace } = this.app;
    let leaf = workspace.getLeavesOfType(VIEW_TYPE_GOCLAW)[0] ?? null;
    if (!leaf) {
      leaf = workspace.getRightLeaf(false);
      if (!leaf) return;
      await leaf.setViewState({
        type: VIEW_TYPE_GOCLAW,
        active: true
      });
    }
    await workspace.revealLeaf(leaf);
  }
};

import {
  Notice,
  Plugin,
  WorkspaceLeaf
} from "obsidian";
import { GatewayClient } from "./gateway-client";
import {
  getSecretStorage,
  GoClawSettingTab
} from "./settings";
import {
  DEFAULT_SETTINGS,
  GoClawSettings
} from "./types";
import {
  GoClawView,
  VIEW_TYPE_GOCLAW
} from "./view";

export default class GoClawPlugin extends Plugin {
  settings: GoClawSettings = DEFAULT_SETTINGS;
  readonly client = new GatewayClient();

  async onload(): Promise<void> {
    await this.loadSettings();
    this.registerView(VIEW_TYPE_GOCLAW, (leaf) =>
      new GoClawView(leaf, this, this.client));
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

  onunload(): void {
    this.client.disconnect();
  }

  async loadSettings(): Promise<void> {
    this.settings = Object.assign({}, DEFAULT_SETTINGS, await this.loadData());
  }

  async saveSettings(): Promise<void> {
    await this.saveData(this.settings);
    this.app.workspace.getLeavesOfType(VIEW_TYPE_GOCLAW)
      .forEach((leaf) => (leaf.view as GoClawView).refresh());
  }

  async connectGateway(showNotice = true): Promise<void> {
    const secretStorage = getSecretStorage(this.app);
    const token = secretStorage?.getSecret(this.settings.secretKey) ?? "";
    const userToken = secretStorage?.getSecret(this.settings.userSecretKey) ?? "";
    try {
      await this.client.connect(this.settings.gatewayUrl, token, userToken);
      if (showNotice) new Notice("GoClaw Gateway 已连接");
    } catch (error) {
      if (showNotice) {
        new Notice(error instanceof Error ? error.message : String(error));
      }
    }
  }

  governanceParams(
    rationale: string,
    counterargument = "",
    evidenceRefs: string[] = []
  ): Record<string, unknown> {
    const secretStorage = getSecretStorage(this.app);
    return {
      reviewer_id: this.settings.reviewerId,
      reviewer_token: secretStorage?.getSecret(this.settings.reviewerSecretKey) ?? "",
      rationale,
      counterargument,
      evidence_refs: evidenceRefs
    };
  }

  actorId(): string {
    return this.settings.reviewerId || "obsidian-user";
  }

  async activateView(): Promise<void> {
    const { workspace } = this.app;
    let leaf: WorkspaceLeaf | null = workspace.getLeavesOfType(VIEW_TYPE_GOCLAW)[0] ?? null;
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
}

import {
  App,
  Notice,
  PluginSettingTab,
  Setting
} from "obsidian";
import GoClawPlugin from "./main";

export interface SecretStorageLike {
  getSecret(key: string): string | null;
  setSecret(key: string, value: string): void;
}

export function getSecretStorage(app: App): SecretStorageLike | null {
  const storage = (app as App & { secretStorage?: SecretStorageLike }).secretStorage;
  return storage ?? null;
}

export class GoClawSettingTab extends PluginSettingTab {
  constructor(app: App, private readonly plugin: GoClawPlugin) {
    super(app, plugin);
  }

  display(): void {
    const { containerEl } = this;
    containerEl.empty();
    containerEl.createEl("h2", { text: "GoClaw Project Console" });

    new Setting(containerEl)
      .setName("Gateway WebSocket")
      .setDesc("Use wss:// through a TLS reverse proxy for non-local connections.")
      .addText((text) => text
        .setPlaceholder("ws://127.0.0.1:28789/ws")
        .setValue(this.plugin.settings.gatewayUrl)
        .onChange(async (value) => {
          this.plugin.settings.gatewayUrl = value.trim();
          await this.plugin.saveSettings();
        }));

    new Setting(containerEl)
      .setName("Project ID")
      .setDesc("Shared routing boundary used by Obsidian, Feishu and gateway chat.")
      .addText((text) => text
        .setValue(this.plugin.settings.projectId)
        .onChange(async (value) => {
          this.plugin.settings.projectId = value.trim() || "default";
          await this.plugin.saveSettings();
        }));

    new Setting(containerEl)
      .setName("Topic ID")
      .setDesc("Conversation topic inside the selected project.")
      .addText((text) => text
        .setValue(this.plugin.settings.topicId)
        .onChange(async (value) => {
          this.plugin.settings.topicId = value.trim() || "inbox";
          await this.plugin.saveSettings();
        }));

    const secretStorage = getSecretStorage(this.app);
    const secret = secretStorage?.getSecret(this.plugin.settings.secretKey) ?? "";
    new Setting(containerEl)
      .setName("Gateway token")
      .setDesc(secretStorage
        ? "Stored in Obsidian SecretStorage; it is never written to data.json."
        : "This Obsidian version does not expose SecretStorage. Upgrade Obsidian before storing a token.")
      .addText((text) => {
        text.inputEl.type = "password";
        text.setPlaceholder(secret ? "Token is stored" : "Paste gateway token")
          .onChange((value) => {
            if (!secretStorage) {
              new Notice("SecretStorage is unavailable; token was not saved.");
              return;
            }
            secretStorage.setSecret(this.plugin.settings.secretKey, value);
          });
      });

    const userSecret = secretStorage?.getSecret(
      this.plugin.settings.userSecretKey
    ) ?? "";
    new Setting(containerEl)
      .setName("Team user token")
      .setDesc(secretStorage
        ? "Personal team identity token. It is stored in SecretStorage and is separate from the shared Gateway token."
        : "SecretStorage is unavailable. Upgrade Obsidian before enabling team mode.")
      .addText((text) => {
        text.inputEl.type = "password";
        text.setPlaceholder(userSecret ? "User token is stored" : "Paste personal user token")
          .onChange((value) => {
            if (!secretStorage) {
              new Notice("SecretStorage is unavailable; user token was not saved.");
              return;
            }
            secretStorage.setSecret(this.plugin.settings.userSecretKey, value);
          });
      });

    new Setting(containerEl)
      .setName("Reviewer ID")
      .setDesc("Legacy single-user mode only. In team mode, approval identity is bound to the personal team token.")
      .addText((text) => text
        .setPlaceholder("alice")
        .setValue(this.plugin.settings.reviewerId)
        .onChange(async (value) => {
          this.plugin.settings.reviewerId = value.trim();
          await this.plugin.saveSettings();
        }));

    const reviewerSecret = secretStorage?.getSecret(
      this.plugin.settings.reviewerSecretKey
    ) ?? "";
    new Setting(containerEl)
      .setName("Reviewer token")
      .setDesc(secretStorage
        ? "Stored separately in Obsidian SecretStorage and sent only on decision RPCs."
        : "SecretStorage is unavailable; authenticated approvals are disabled on this device.")
      .addText((text) => {
        text.inputEl.type = "password";
        text.setPlaceholder(reviewerSecret ? "Reviewer token is stored" : "Paste reviewer token")
          .onChange((value) => {
            if (!secretStorage) {
              new Notice("SecretStorage is unavailable; reviewer token was not saved.");
              return;
            }
            secretStorage.setSecret(this.plugin.settings.reviewerSecretKey, value);
          });
      });

    new Setting(containerEl)
      .setName("Connect automatically")
      .setDesc("Reconnect with exponential backoff when Obsidian starts.")
      .addToggle((toggle) => toggle
        .setValue(this.plugin.settings.autoConnect)
        .onChange(async (value) => {
          this.plugin.settings.autoConnect = value;
          await this.plugin.saveSettings();
        }));

    new Setting(containerEl)
      .setName("Reconnect now")
      .addButton((button) => button
        .setButtonText("Reconnect")
        .onClick(() => void this.plugin.connectGateway()));
  }
}

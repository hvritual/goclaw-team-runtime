import {
  ItemView,
  Notice,
  WorkspaceLeaf,
  setIcon
} from "obsidian";
import {
  GatewayClient,
  TEAM_CONTROL_RPC
} from "./gateway-client";
import GoClawPlugin from "./main";
import {
  issueState,
  leaseState,
  memberState,
  runnerState,
  severityState,
  teamStateClass,
  workState
} from "./team-presenter";
import {
  ChatEvent,
  ChatMessage,
  CatalogMemoryRecord,
  CatalogMemorySearchResult,
  CatalogMemoryStats,
  ConnectionState,
  DevReviewKind,
  DevTask,
  HarnessExperiment,
  HarnessStatus,
  HarnessTrace,
  KnowledgeProposal,
  OuroborosSeed,
  OuroborosSession,
  TeamComponent,
  TeamComponentsSummary,
  TeamDocsSummary,
  TeamDocument,
  TeamIssue,
  TeamMember,
  TeamPolicyStatus,
  TeamRunner,
  TeamWorkItem
} from "./types";
import {
  clampPercent,
  collectionItems,
  displayInitial,
  relativeTime,
  safeExcerpt,
  shortHash
} from "./util";

export const VIEW_TYPE_GOCLAW = "goclaw-project-console";
type ViewTab =
  | "chat"
  | "spec"
  | "memory"
  | "approvals"
  | "development"
  | "team"
  | "progress"
  | "harness";
type ReviewFields = {
  rationale: HTMLTextAreaElement;
  counterargument: HTMLTextAreaElement;
};

export class GoClawView extends ItemView {
  private activeTab: ViewTab = "chat";
  private bodyEl: HTMLElement | null = null;
  private statusEl: HTMLElement | null = null;
  private chatMessages: ChatMessage[] = [];
  private disposers: Array<() => void> = [];

  constructor(
    leaf: WorkspaceLeaf,
    private readonly plugin: GoClawPlugin,
    private readonly client: GatewayClient
  ) {
    super(leaf);
  }

  getViewType(): string {
    return VIEW_TYPE_GOCLAW;
  }

  getDisplayText(): string {
    return "GoClaw";
  }

  getIcon(): string {
    return "bot-message-square";
  }

  async onOpen(): Promise<void> {
    this.disposers.push(
      this.client.onState((state, detail) => this.updateStatus(state, detail)),
      this.client.on("chat.event", (payload) => this.onChatEvent(payload))
    );
    this.renderShell();
  }

  async onClose(): Promise<void> {
    this.disposers.splice(0).forEach((dispose) => dispose());
  }

  refresh(): void {
    this.renderShell();
  }

  private renderShell(): void {
    const root = this.containerEl.children[1] as HTMLElement;
    root.empty();
    root.addClass("goclaw-console");

    const header = root.createDiv({ cls: "goclaw-header" });
    const brand = header.createDiv({ cls: "goclaw-brand" });
    const brandIcon = brand.createSpan({ cls: "goclaw-brand-icon" });
    setIcon(brandIcon, "bot");
    brand.createSpan({ text: "GoClaw" });
    const reconnect = header.createEl("button", {
      cls: "clickable-icon goclaw-icon-button",
      attr: { "aria-label": "Reconnect gateway" }
    });
    setIcon(reconnect, "refresh-cw");
    reconnect.addEventListener("click", () => void this.plugin.connectGateway());

    const project = root.createDiv({ cls: "goclaw-project-row" });
    const folderIcon = project.createSpan();
    setIcon(folderIcon, "folder-kanban");
    project.createSpan({
      cls: "goclaw-project-name",
      text: this.plugin.settings.projectId
    });
    const status = project.createSpan({ cls: "goclaw-connection" });
    status.createSpan({ cls: "goclaw-status-dot" });
    this.statusEl = status.createSpan({ text: "未连接" });
    this.updateStatus(this.client.connectionState);

    const tabs = root.createDiv({ cls: "goclaw-tabs", attr: { role: "tablist" } });
    ([
      ["chat", "聊天"],
      ["spec", "规格"],
      ["memory", "记忆"],
      ["approvals", "审批"],
      ["development", "开发"],
      ["team", "团队"],
      ["progress", "进度"],
      ["harness", "Harness"]
    ] as Array<[ViewTab, string]>).forEach(([id, label]) => {
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
    setIcon(syncIcon, "refresh-cw");
    footer.createSpan({ text: "Vault 就绪" });
    footer.createSpan({
      cls: "goclaw-footer-detail",
      text: "同步由 Obsidian 管理",
      attr: { title: "插件不伪造远端同步状态；请在 Obsidian Sync 中确认设备状态。" }
    });
  }

  private renderActiveTab(): void {
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

  private renderChat(): void {
    if (!this.bodyEl) return;
    const list = this.bodyEl.createDiv({ cls: "goclaw-chat-list" });
    if (this.chatMessages.length === 0) {
      const empty = list.createDiv({ cls: "goclaw-empty" });
      const icon = empty.createSpan();
      setIcon(icon, "message-square");
      empty.createEl("strong", { text: "项目会话尚未开始" });
      empty.createEl("p", { text: "这里与飞书共享同一 project_id；topic_id 用于细分讨论。" });
    } else {
      this.chatMessages.forEach((message) => {
        const item = list.createDiv({
          cls: `goclaw-message is-${message.role}${message.pending ? " is-pending" : ""}${message.error ? " is-error" : ""}`
        });
        item.createDiv({
          cls: "goclaw-message-role",
          text: message.role === "user" ? "你" : message.role === "assistant" ? "GoClaw" : "系统"
        });
        item.createDiv({ cls: "goclaw-message-content", text: message.content || "…" });
      });
    }

    const composer = this.bodyEl.createDiv({ cls: "goclaw-composer" });
    const textarea = composer.createEl("textarea", {
      attr: {
        placeholder: "给当前项目发送消息…",
        rows: "3",
        "aria-label": "GoClaw message"
      }
    });
    const send = composer.createEl("button", { cls: "mod-cta", text: "发送" });
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

  private onChatEvent(payload: ChatEvent): void {
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
    if (payload.state === "thinking" && !message.content) message.content = "正在分析…";
    if (payload.state === "tool") message.content = payload.content || "正在调用工具…";
    if (payload.state === "final") {
      message.content = payload.content || message.content;
      message.pending = false;
    }
    if (payload.state === "error") {
      message.content = payload.content || "运行失败";
      message.pending = false;
      message.error = true;
    }
    if (this.activeTab === "chat") this.renderActiveTab();
  }

  private async renderSpec(): Promise<void> {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "加载 Ouroboros 规格闭环");
    try {
      const sessions = await this.client.rpc<OuroborosSession[]>("ouroboros.sessions", {
        project_id: this.plugin.settings.projectId
      });
      if (this.activeTab !== "spec" || !this.bodyEl) return;
      target.empty();

      const intro = target.createDiv({ cls: "goclaw-spec-intro" });
      const title = intro.createDiv({ cls: "goclaw-review-title" });
      const icon = title.createSpan();
      setIcon(icon, "infinity");
      const copy = title.createDiv();
      copy.createEl("strong", { text: "先结晶规格，再进入受控开发" });
      copy.createSpan({ text: "interview → Seed → approval → compile → evaluate → evolve" });
      intro.createEl("p", {
        text: "聊天记忆只提供上下文；不可变 Seed、验证证据和 Go 事件链才是执行依据。"
      });

      const form = target.createEl("form", { cls: "goclaw-spec-form" });
      const request = form.createEl("textarea", {
        attr: {
          rows: "4",
          placeholder: "描述要开发的目标、约束与验收方式…",
          "aria-label": "Ouroboros development request"
        }
      });
      const options = form.createDiv({ cls: "goclaw-spec-options" });
      const brownfieldLabel = options.createEl("label");
      const brownfield = brownfieldLabel.createEl("input", { type: "checkbox" });
      brownfield.checked = true;
      brownfieldLabel.createSpan({ text: "现有代码库" });
      const start = options.createEl("button", {
        cls: "mod-cta",
        text: "开始规格访谈",
        attr: { type: "submit" }
      });
      form.addEventListener("submit", (event) => {
        event.preventDefault();
        const rawRequest = request.value.trim();
        if (!rawRequest) {
          new Notice("请先输入开发需求");
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
        }, 240_000).then(async () => {
          request.value = "";
          new Notice("规格访谈已开始");
          await this.renderSpec();
        }).catch((error) => {
          new Notice(error instanceof Error ? error.message : String(error));
        }).finally(() => {
          start.disabled = false;
        });
      });

      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "规格", sessions.length);
      this.metric(summary, "待澄清", sessions.filter((session) =>
        ["interviewing", "clarification_required"].includes(session.status)).length);
      this.metric(summary, "待审批", sessions.filter((session) =>
        ["awaiting_seed_approval", "evolution_pending"].includes(session.status)).length);

      const list = target.createDiv({ cls: "goclaw-spec-list" });
      sessions.forEach((session) => list.appendChild(this.renderSpecSession(session)));
      if (sessions.length === 0) {
        this.renderEmpty(list, "infinity", "还没有规格会话", "输入开发目标，Ouroboros 会优先追问高信息增益问题。");
      }
    } catch (error) {
      if (this.activeTab === "spec") this.renderError(target, error);
    }
  }

  private renderSpecSession(session: OuroborosSession): HTMLElement {
    const card = document.createElement("article");
    card.addClass("goclaw-spec-card", `is-${session.status}`);
    const heading = card.createDiv({ cls: "goclaw-spec-heading" });
    const info = heading.createDiv();
    info.createEl("strong", { text: session.title });
    info.createSpan({ text: `${session.id} · ${relativeTime(session.updated_at)}` });
    heading.createSpan({ cls: `goclaw-state is-${session.status}`, text: session.status });
    card.createEl("p", { text: safeExcerpt(session.raw_request, 180) });

    const latest = session.rounds?.at(-1);
    if (latest) {
      const meter = card.createDiv({ cls: "goclaw-ambiguity" });
      const label = meter.createDiv();
      label.createSpan({ text: "歧义度" });
      label.createEl("strong", {
        text: `${Math.round(latest.assessment.overall * 100)}% / 阈值 ${Math.round(latest.assessment.threshold * 100)}%`
      });
      const track = meter.createDiv({ cls: "goclaw-ambiguity-track" });
      const fill = track.createDiv({ cls: "goclaw-ambiguity-fill" });
      fill.style.width = `${Math.min(100, Math.max(0, latest.assessment.overall * 100))}%`;
      meter.createSpan({
        cls: "goclaw-spec-note",
        text: `${latest.assessment.summary} · ready ${latest.assessment.ready_streak}/${latest.assessment.required_ready_streak} · 分歧 ${Math.round((latest.assessment.score_spread ?? 0) * 100)}%${latest.assessment.gray_zone ? " · 灰区" : ""}`
      });
    }

    const answered = new Set((latest?.answers ?? []).map((answer) => answer.question_id));
    const questions = (latest?.questions ?? []).filter((question) => !answered.has(question.id));
    if (questions.length > 0 && ["interviewing", "clarification_required"].includes(session.status)) {
      const questionForm = card.createEl("form", { cls: "goclaw-question-list" });
      const inputs = new Map<string, HTMLTextAreaElement>();
      questions.forEach((question) => {
        const item = questionForm.createDiv({ cls: "goclaw-question" });
        const questionLabel = item.createEl("label", {
          text: `${question.blocking ? "必答 · " : ""}${question.text}`
        });
        if (question.why) questionLabel.setAttribute("title", question.why);
        const input = item.createEl("textarea", {
          attr: { rows: "2", placeholder: "输入明确答案…" }
        });
        inputs.set(question.id, input);
      });
      const actions = questionForm.createDiv({ cls: "goclaw-actions" });
      const submit = actions.createEl("button", {
        cls: "mod-cta",
        text: "提交答案并重评",
        attr: { type: "submit" }
      });
      questionForm.addEventListener("submit", (event) => {
        event.preventDefault();
        const answers = Array.from(inputs.entries())
          .map(([questionId, input]) => ({ question_id: questionId, text: input.value.trim() }))
          .filter((answer) => answer.text.length > 0);
        if (answers.length === 0) {
          new Notice("至少回答一个问题");
          return;
        }
        submit.disabled = true;
        void this.client.rpc("ouroboros.session.answer", {
          id: session.id,
          answers,
          actor: this.plugin.actorId()
        }, 240_000).then(() => this.renderSpec()).catch((error) => {
          new Notice(error instanceof Error ? error.message : String(error));
        }).finally(() => {
          submit.disabled = false;
        });
      });
    }

    if (session.last_error) {
      const error = card.createDiv({ cls: "goclaw-safety is-error" });
      setIcon(error.createSpan(), "circle-x");
      error.createSpan({ text: safeExcerpt(session.last_error, 160) });
    }
    if (session.status === "awaiting_seed_approval") {
      const note = card.createDiv({ cls: "goclaw-safety" });
      setIcon(note.createSpan(), "shield-check");
      note.createSpan({ text: `Seed ${shortHash(session.pending_seed_hash)} 等待人工审批` });
    }
    if (session.status === "evolution_pending") {
      const note = card.createDiv({ cls: "goclaw-safety" });
      setIcon(note.createSpan(), "git-compare-arrows");
      note.createSpan({
        text: `候选 G${session.pending_evolution?.candidate_generation ?? "?"} · 本体相似度 ${Math.round((session.pending_evolution?.ontology_similarity ?? 0) * 100)}%`
      });
    }

    const actions = card.createDiv({ cls: "goclaw-actions" });
    if (["interviewing", "clarification_required"].includes(session.status)) {
      this.actionButton(actions, "重新评估", "", async () => {
        await this.client.rpc("ouroboros.session.reassess", {
          id: session.id,
          actor: this.plugin.actorId()
        }, 240_000);
        await this.renderSpec();
      });
    }
    if (session.status === "seed_ready") {
      this.actionButton(actions, "生成不可变 Seed", "mod-cta", async () => {
        await this.client.rpc("ouroboros.session.crystallize", {
          id: session.id,
          actor: this.plugin.actorId()
        }, 240_000);
        new Notice("Seed 已生成，请到审批页复核");
        await this.renderSpec();
      });
    }
    if (session.status === "approved") {
      this.actionButton(actions, "编译为开发任务", "mod-cta", async () => {
        await this.client.rpc("ouroboros.session.compile", {
          id: session.id,
          actor: this.plugin.actorId()
        });
        new Notice("已编译为 Orchestrator Lite 任务，仍需四类审批");
        await this.renderSpec();
      });
    }
    if (session.status === "compiled") {
      const task = session.compiled_tasks?.at(-1);
      if (task) {
        this.actionButton(actions, "依据证据评估", "mod-cta", async () => {
          await this.client.rpc("ouroboros.session.evaluate", {
            id: session.id,
            task_id: task.task_id,
            actor: this.plugin.actorId()
          }, 900_000);
          await this.renderSpec();
        });
      }
    }
    if (session.status === "evaluated") {
      this.actionButton(actions, "生成演化候选", "mod-cta", async () => {
        await this.client.rpc("ouroboros.session.evolve", {
          id: session.id,
          actor: this.plugin.actorId()
        }, 240_000);
        await this.renderSpec();
      });
    }
    if (!["converged", "cancelled", "rejected"].includes(session.status)) {
      const cancelBlock = card.createDiv({ cls: "goclaw-question" });
      const fields = this.reviewFields(cancelBlock, "取消 Ouroboros 会话");
      this.actionButton(cancelBlock, "取消会话", "mod-warning", async () => {
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

  private async renderApprovals(): Promise<void> {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "加载审批队列");
    try {
      const [knowledge, memoryCandidates, experiments, tasks, ouroborosSessions] = await Promise.all([
        this.client.rpc<KnowledgeProposal[]>("knowledge.proposals", { status: "pending" }),
        this.client.rpc<CatalogMemoryRecord[]>("memory.catalog.list", {
          project_id: this.plugin.settings.projectId,
          status: "pending",
          limit: 100
        }).catch(() => []),
        this.client.rpc<HarnessExperiment[]>("harness.experiments"),
        this.client
          .rpc<DevTask[]>("dev.tasks", { project_id: this.plugin.settings.projectId })
          .catch(() => []),
        this.client
          .rpc<OuroborosSession[]>("ouroboros.sessions", {
            project_id: this.plugin.settings.projectId
          })
          .catch(() => [])
      ]);
      if (this.activeTab !== "approvals" || !this.bodyEl) return;
      target.empty();
      this.renderApprovalSection(target, "知识提案", knowledge.length, knowledge, (proposal) =>
        this.renderKnowledgeProposal(proposal));
      this.renderApprovalSection(
        target,
        "记忆编目候选",
        memoryCandidates.length,
        memoryCandidates,
        (record) => this.renderCatalogCandidate(record)
      );
      const reviewable = experiments.filter((experiment) =>
        ["validated", "human_approved"].includes(experiment.status));
      this.renderApprovalSection(target, "Harness 实验", reviewable.length, reviewable, (experiment) =>
        this.renderExperiment(experiment));
      const developmentReviewable = tasks.filter((task) =>
        ["review_pending", "ready_to_freeze", "blocked", "awaiting_acceptance"].includes(task.status));
      this.renderApprovalSection(
        target,
        "开发任务",
        developmentReviewable.length,
        developmentReviewable,
        (task) => this.renderDevelopmentApproval(task)
      );
      const seedReviewable = ouroborosSessions.filter((session) =>
        session.status === "awaiting_seed_approval");
      const evolutionReviewable = ouroborosSessions.filter((session) =>
        session.status === "evolution_pending");
      const cognitiveReviewable = ouroborosSessions.filter((session) => {
        const latest = session.rounds?.at(-1)?.assessment;
        return Boolean(
          latest?.human_decision_required ||
          session.decision_conflicts?.some((conflict) => conflict.status === "open") ||
          session.evaluations?.some((evaluation) => evaluation.human_decision_required)
        );
      });
      const seedHashes = Array.from(new Set([
        ...seedReviewable.map((session) => session.pending_seed_hash),
        ...evolutionReviewable.map((session) => session.pending_evolution?.candidate_seed_hash)
      ].filter((hash): hash is string => Boolean(hash))));
      const seedEntries = await Promise.all(seedHashes.map(async (hash) => {
        const seed = await this.client
          .rpc<OuroborosSeed>("ouroboros.seed.get", { hash })
          .catch(() => null);
        return [hash, seed] as const;
      }));
      const seeds = new Map<string, OuroborosSeed | null>(seedEntries);
      if (this.activeTab !== "approvals" || !this.bodyEl) return;
      this.renderApprovalSection(
        target,
        "认知分歧与利益相关方冲突",
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
        "Ouroboros 演化候选",
        evolutionReviewable.length,
        evolutionReviewable,
        (session) => this.renderOuroborosEvolutionApproval(
          session,
          session.pending_evolution?.candidate_seed_hash
            ? seeds.get(session.pending_evolution.candidate_seed_hash) ?? null
            : null
        )
      );
      if (
        knowledge.length +
        memoryCandidates.length +
        reviewable.length +
        developmentReviewable.length +
        cognitiveReviewable.length +
        seedReviewable.length +
        evolutionReviewable.length === 0
      ) {
        this.renderEmpty(target, "inbox", "没有待审批事项", "新提案与通过评测的实验会出现在这里。");
      }
    } catch (error) {
      if (this.activeTab === "approvals") this.renderError(target, error);
    }
  }

  private renderCognitiveEscalation(session: OuroborosSession): HTMLElement {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const latest = session.rounds?.at(-1);
    const disputedEvaluation = [...(session.evaluations ?? [])]
      .reverse()
      .find((evaluation) => evaluation.human_decision_required);
    const title = row.createDiv({ cls: "goclaw-review-title" });
    setIcon(title.createSpan(), "scale");
    const copy = title.createDiv();
    copy.createEl("strong", { text: session.title });
    copy.createSpan({
      text: disputedEvaluation
        ? `证据评估争议 · ${disputedEvaluation.distinct_models ?? 0} 个模型 · 分差 ${Math.round((disputedEvaluation.score_spread ?? 0) * 100)}%`
        : `需求评估分歧 · ${latest?.assessment.distinct_models ?? 0} 个模型 · 分差 ${Math.round((latest?.assessment.score_spread ?? 0) * 100)}%${latest?.assessment.gray_zone ? " · 阈值灰区" : ""}`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: safeExcerpt(
        disputedEvaluation?.consensus.summary ?? latest?.assessment.unresolved?.join("；"),
        240
      ) ||
        "多个评估视角无法给出稳定的自动结论。"
    });
    const fields = this.reviewFields(row, "认知分歧");
    const openConflicts = (session.decision_conflicts ?? [])
      .filter((conflict) => conflict.status === "open");
    openConflicts.forEach((conflict) => {
      const block = row.createDiv({ cls: "goclaw-question" });
      block.createEl("strong", { text: conflict.description });
      const resolution = block.createEl("textarea", {
        attr: {
          rows: "2",
          placeholder: "明确选择、优先级或可验证折中方案…",
          "aria-label": `Resolve conflict ${conflict.id}`
        }
      });
      this.actionButton(block, "解决此冲突", "mod-cta", async () => {
        if (!resolution.value.trim()) throw new Error("请填写明确的冲突解决方案");
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
      block.createEl("strong", { text: `评估 ${disputedEvaluation.id}` });
      block.createDiv({
        text: [
          `机械门：${disputedEvaluation.mechanical.passed ? "通过" : "失败"}`,
          `语义门：${disputedEvaluation.semantic.passed ? "通过" : "失败"}`,
          `共识门：${disputedEvaluation.consensus.passed ? "通过" : "争议"}`
        ].join(" · ")
      });
      block.createDiv({
        cls: "goclaw-review-reason",
        text: "这里仅裁决证据争议，不代表开发任务验收、部署授权或 Harness 晋级。"
      });
      const actions = block.createDiv({ cls: "goclaw-actions" });
      this.actionButton(actions, "驳回争议评估", "mod-warning", async () => {
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
      this.actionButton(actions, "接受证据结论（非验收）", "mod-cta", async () => {
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
      this.actionButton(actions, "判定仍需澄清", "mod-warning", async () => {
        await this.client.rpc("ouroboros.readiness.resolve", {
          id: session.id,
          ready: false,
          ...this.reviewPayload(fields, [`assessment:${session.id}:${latest.number}`], true)
        });
        await this.renderApprovals();
      });
      this.actionButton(actions, "判定可结晶", "mod-cta", async () => {
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

  private renderApprovalSection<T>(
    parent: HTMLElement,
    title: string,
    count: number,
    items: T[],
    render: (item: T) => HTMLElement
  ): void {
    const section = parent.createDiv({ cls: "goclaw-section" });
    const heading = section.createDiv({ cls: "goclaw-section-heading" });
    heading.createEl("h3", { text: title });
    heading.createSpan({ cls: "goclaw-count", text: String(count) });
    items.forEach((item) => section.appendChild(render(item)));
  }

  private renderKnowledgeProposal(proposal: KnowledgeProposal): HTMLElement {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    const icon = title.createSpan();
    setIcon(icon, "file-check-2");
    const titleText = title.createDiv();
    titleText.createEl("strong", { text: proposal.target_path });
    titleText.createSpan({ text: relativeTime(proposal.created_at) });
    row.createDiv({ cls: "goclaw-review-reason", text: proposal.reason });
    const safety = row.createDiv({ cls: "goclaw-safety" });
    const shield = safety.createSpan();
    setIcon(shield, "shield-check");
    safety.createSpan({
      text: `基于创建时版本 · SHA ${shortHash(proposal.base_sha256)}`
    });
    const fields = this.reviewFields(row, "知识提案");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "拒绝", "mod-warning", async () => {
      await this.client.rpc("knowledge.proposal.reject", {
        id: proposal.id,
        ...this.reviewPayload(fields, [`knowledge-proposal:${proposal.id}`], false)
      });
      new Notice("知识提案已拒绝");
      await this.renderApprovals();
    });
    this.actionButton(actions, "批准", "mod-cta", async () => {
      await this.client.rpc("knowledge.proposal.approve", {
        id: proposal.id,
        ...this.reviewPayload(fields, [`knowledge-proposal:${proposal.id}`], true)
      });
      new Notice("知识提案已应用到 Vault");
      await this.renderApprovals();
    });
    return row;
  }

  private renderCatalogCandidate(record: CatalogMemoryRecord): HTMLElement {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    setIcon(title.createSpan(), "library");
    const titleText = title.createDiv();
    titleText.createEl("strong", { text: record.title });
    titleText.createSpan({
      text: `${record.kind} · v${record.version} · ${relativeTime(record.created_at)}`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: record.abstract || safeExcerpt(record.content, 240)
    });
    const safety = row.createDiv({ cls: "goclaw-safety" });
    setIcon(safety.createSpan(), "fingerprint");
    safety.createSpan({
      text: `${record.provenance.source_uri} · SHA ${shortHash(record.checksum)}`
    });
    const fields = this.reviewFields(row, "记忆候选");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "拒绝", "mod-warning", async () => {
      await this.client.rpc("memory.catalog.candidate.reject", {
        id: record.id,
        ...this.reviewPayload(fields, [`catalog:${record.id}@v${record.version}`], false)
      });
      new Notice("记忆候选已拒绝，未进入检索上下文");
      await this.renderApprovals();
    });
    this.actionButton(actions, "批准入藏", "mod-cta", async () => {
      await this.client.rpc("memory.catalog.candidate.approve", {
        id: record.id,
        ...this.reviewPayload(fields, [`catalog:${record.id}@v${record.version}`], true)
      });
      new Notice("记忆已批准入藏");
      await this.renderApprovals();
    });
    return row;
  }

  private async renderMemory(): Promise<void> {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "加载记忆目录");
    try {
      const [stats, active, pending] = await Promise.all([
        this.client.rpc<CatalogMemoryStats>("memory.catalog.status", {
          project_id: this.plugin.settings.projectId
        }),
        this.client.rpc<CatalogMemoryRecord[]>("memory.catalog.list", {
          project_id: this.plugin.settings.projectId,
          status: "active",
          limit: 30
        }),
        this.client.rpc<CatalogMemoryRecord[]>("memory.catalog.list", {
          project_id: this.plugin.settings.projectId,
          status: "pending",
          limit: 30
        })
      ]);
      if (this.activeTab !== "memory" || !this.bodyEl) return;
      target.empty();

      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "在藏", stats.by_status.active ?? 0);
      this.metric(summary, "待编目", stats.by_status.pending ?? 0, (stats.by_status.pending ?? 0) > 0);
      this.metric(summary, "待复核", stats.review_due, stats.review_due > 0);
      this.metric(
        summary,
        "冲突",
        stats.unresolved_contradictions,
        stats.unresolved_contradictions > 0
      );

      const search = target.createDiv({ cls: "goclaw-memory-search" });
      const input = search.createEl("input", {
        type: "search",
        attr: {
          placeholder: "检索项目决策、约束、事实或偏好…",
          "aria-label": "Search catalog memory"
        }
      });
      const button = search.createEl("button", { text: "检索", cls: "mod-cta" });
      const resultsTarget = target.createDiv({ cls: "goclaw-section" });
      const renderResults = (results: CatalogMemorySearchResult[]): void => {
        resultsTarget.empty();
        const heading = resultsTarget.createDiv({ cls: "goclaw-section-heading" });
        heading.createEl("h3", { text: "检索结果" });
        heading.createSpan({ cls: "goclaw-count", text: String(results.length) });
        results.forEach((result) =>
          resultsTarget.appendChild(this.renderCatalogRecord(result.record, result)));
        if (results.length === 0) {
          this.renderEmpty(resultsTarget, "search-x", "没有匹配的已批准记忆", "候选记录不会出现在检索结果中。");
        }
      };
      const runSearch = async (): Promise<void> => {
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
          const results = await this.client.rpc<CatalogMemorySearchResult[]>(
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
      pendingHeading.createEl("h3", { text: "待编目" });
      pendingHeading.createSpan({ cls: "goclaw-count", text: String(pending.length) });
      pending.slice(0, 8).forEach((record) =>
        pendingSection.appendChild(this.renderCatalogCandidate(record)));
      if (pending.length === 0) {
        this.renderEmpty(pendingSection, "archive-restore", "没有待编目记录", "Agent 提案与 Vault 摄取结果会先进入这里。");
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

  private renderCatalogRecord(
    record: CatalogMemoryRecord,
    result: CatalogMemorySearchResult
  ): HTMLElement {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    setIcon(title.createSpan(), result.review_due ? "clock-alert" : "book-check");
    const copy = title.createDiv();
    copy.createEl("strong", { text: record.title });
    copy.createSpan({
      text: `${record.kind} · v${record.version} · ${Math.round(result.score * 100)}%`
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
        text: (result.warnings ?? []).join(" · ")
      });
    }
    if (result.review_due) {
      const fields = this.reviewFields(row, "记忆复核");
      const actions = row.createDiv({ cls: "goclaw-actions" });
      this.actionButton(actions, "确认并延长 90 天", "mod-cta", async () => {
        await this.client.rpc("memory.catalog.review.renew", {
          id: record.id,
          days: 90,
          ...this.reviewPayload(fields, [result.citation], true)
        });
        new Notice("记忆复核周期已更新");
        await this.renderMemory();
      });
      this.actionButton(actions, "退藏", "mod-warning", async () => {
        await this.client.rpc("memory.catalog.withdraw", {
          id: record.id,
          ...this.reviewPayload(fields, [result.citation], false)
        });
        new Notice("记录已退藏，不再进入自动上下文");
        await this.renderMemory();
      });
    }
    return row;
  }

  private renderExperiment(experiment: HarnessExperiment): HTMLElement {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    const icon = title.createSpan();
    setIcon(icon, "flask-conical");
    const text = title.createDiv();
    text.createEl("strong", { text: experiment.candidate_version });
    text.createSpan({ text: experiment.change_manifest.change_summary });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: `根因：${experiment.change_manifest.root_cause}`
    });
    const fields = this.reviewFields(row, "Harness 实验");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "拒绝", "mod-warning", async () => {
      await this.client.rpc("harness.experiment.reject", {
        id: experiment.id,
        ...this.reviewPayload(fields, [`experiment:${experiment.id}`], false)
      });
      await this.renderApprovals();
    });
    if (experiment.status === "validated") {
      this.actionButton(actions, "批准实验", "mod-cta", async () => {
        await this.client.rpc("harness.experiment.approve", {
          id: experiment.id,
          ...this.reviewPayload(fields, [`experiment:${experiment.id}`], true)
        });
        await this.renderApprovals();
      });
    } else {
      this.actionButton(actions, "提升为当前版本", "mod-cta", async () => {
        await this.client.rpc("harness.experiment.promote", {
          id: experiment.id,
          ...this.reviewPayload(fields, [`experiment:${experiment.id}`], true)
        });
        new Notice("Harness 已提升；后续会话将使用新版本");
        await this.renderApprovals();
      });
    }
    return row;
  }

  private renderOuroborosSeedApproval(
    session: OuroborosSession,
    seed: OuroborosSeed | null
  ): HTMLElement {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "is-expanded");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    setIcon(title.createSpan(), "scan-text");
    const info = title.createDiv();
    info.createEl("strong", { text: session.title });
    info.createSpan({
      text: `Seed ${shortHash(session.pending_seed_hash)} · ${relativeTime(session.updated_at)}`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: seed?.goal ?? safeExcerpt(session.raw_request, 180)
    });
    this.renderOuroborosSeedDetails(row, seed);
    const safety = row.createDiv({ cls: "goclaw-safety" });
    setIcon(safety.createSpan(), "shield-check");
    safety.createSpan({
      text: "批准仅授权进入任务编译；不代表实现正确，也不会直接执行代码。"
    });
    const fields = this.reviewFields(row, "Seed");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "拒绝 Seed", "mod-warning", async () => {
      await this.client.rpc("ouroboros.seed.reject", {
        id: session.id,
        ...this.reviewPayload(fields, [`seed:${session.pending_seed_hash ?? ""}`], false)
      });
      await this.renderApprovals();
    });
    if (seed) {
      this.actionButton(actions, "批准 Seed", "mod-cta", async () => {
        await this.client.rpc("ouroboros.seed.approve", {
          id: session.id,
          ...this.reviewPayload(fields, [`seed:${session.pending_seed_hash ?? ""}`], true)
        });
        new Notice("Seed 已批准，可在规格页编译任务");
        await this.renderApprovals();
      });
    }
    return row;
  }

  private renderOuroborosEvolutionApproval(
    session: OuroborosSession,
    seed: OuroborosSeed | null
  ): HTMLElement {
    const proposal = session.pending_evolution;
    const row = document.createElement("article");
    row.addClass("goclaw-review-item");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    setIcon(title.createSpan(), "git-compare-arrows");
    const info = title.createDiv();
    info.createEl("strong", { text: `${session.title} · G${proposal?.candidate_generation ?? "?"}` });
    info.createSpan({
      text: `本体相似度 ${Math.round((proposal?.ontology_similarity ?? 0) * 100)}%`
    });
    row.createDiv({
      cls: "goclaw-review-reason",
      text: safeExcerpt(proposal?.reasons?.join("；"), 200) || "模型提出了下一代候选 Seed。"
    });
    this.renderOuroborosSeedDetails(row, seed);
    const safety = row.createDiv({ cls: "goclaw-safety" });
    setIcon(safety.createSpan(), proposal?.oscillation_detected ? "triangle-alert" : "shield-check");
    safety.createSpan({
      text: proposal?.oscillation_detected
        ? "检测到本体振荡，必须人工决定。"
        : "候选尚未生效；批准后只切换 active Seed，不直接修改代码或知识库。"
    });
    const fields = this.reviewFields(row, "演化候选");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    this.actionButton(actions, "拒绝候选", "mod-warning", async () => {
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
      this.actionButton(actions, "批准候选", "mod-cta", async () => {
        await this.client.rpc("ouroboros.evolution.approve", {
          id: session.id,
          ...this.reviewPayload(
            fields,
            [`seed:${proposal?.candidate_seed_hash ?? ""}`, `evolution:${proposal?.id ?? ""}`],
            true
          )
        });
        new Notice("候选 Seed 已切换为 active；需重新编译新一代任务");
        await this.renderApprovals();
      });
    }
    return row;
  }

  private renderOuroborosSeedDetails(
    parent: HTMLElement,
    seed: OuroborosSeed | null
  ): void {
    if (!seed) {
      parent.createDiv({
        cls: "goclaw-review-reason",
        text: "Seed 详情读取失败；不要在未核对完整规格时批准。"
      });
      return;
    }
    const details = parent.createEl("details", { cls: "goclaw-seed-details" });
    details.createEl("summary", {
      text: `查看完整审批摘要 · G${seed.generation} · ${shortHash(seed.hash)}`
    });
    const body = details.createDiv();
    body.createEl("strong", { text: "约束" });
    const constraints = body.createEl("ul");
    seed.constraints.forEach((constraint) =>
      constraints.createEl("li", { text: constraint }));
    body.createEl("strong", { text: "验收标准与命令" });
    const criteria = body.createEl("ol");
    seed.acceptance_criteria.forEach((criterion) => {
      const item = criteria.createEl("li");
      item.createSpan({ text: criterion.description });
      if (criterion.verify_command?.length) {
        item.createEl("code", { text: criterion.verify_command.join(" ") });
      }
    });
    body.createEl("strong", { text: "备选方案与不行动成本" });
    const alternatives = body.createEl("ul");
    (seed.alternatives ?? []).forEach((alternative) => {
      alternatives.createEl("li", {
        text: `${alternative.selected ? "已选" : "未选"} · ${alternative.title}：${alternative.summary}`
      });
    });
    const inaction = body.createEl("ul");
    (seed.cost_of_inaction ?? []).forEach((cost) => inaction.createEl("li", { text: cost }));
    body.createEl("strong", { text: "反证与停止条件" });
    const falsifiers = body.createEl("ul");
    (seed.falsifiers ?? []).forEach((falsifier) => falsifiers.createEl("li", {
      text: `${falsifier.criterion_id} · ${falsifier.condition} · 证据：${falsifier.evidence_required}`
    }));
    const kills = body.createEl("ul");
    (seed.kill_conditions ?? []).forEach((condition) => kills.createEl("li", {
      text: `${condition.id} · ${condition.metric} > ${condition.threshold} → ${condition.action}`
    }));
    body.createEl("strong", { text: "预注册预测" });
    const predictions = body.createEl("ul");
    (seed.predictions ?? []).forEach((prediction) => predictions.createEl("li", {
      text: `${prediction.horizon} · ${Math.round(prediction.confidence * 100)}% · ${prediction.expected_outcome}`
    }));
    body.createEl("strong", { text: "执行边界" });
    body.createEl("p", {
      text: `${seed.scope.allowed_paths.join(", ")} · 最多 ${seed.scope.max_changed_files} 文件 / ${seed.scope.max_changed_lines} 行 · 风险 ${seed.risk.level} · 修复 ${seed.cost.max_repair_attempts} 次`
    });
    body.createEl("strong", { text: "回滚" });
    body.createEl("p", { text: seed.risk.rollback });
  }

  private renderDevelopmentApproval(task: DevTask): HTMLElement {
    const row = document.createElement("article");
    row.addClass("goclaw-review-item", "goclaw-dev-review");
    const title = row.createDiv({ cls: "goclaw-review-title" });
    const icon = title.createSpan();
    setIcon(icon, "git-pull-request-draft");
    const text = title.createDiv();
    text.createEl("strong", { text: task.title });
    text.createSpan({ text: `${task.status} · rev ${task.compile.revision}` });
    row.createDiv({ cls: "goclaw-review-reason", text: safeExcerpt(task.goal.objective, 160) });

    const reviewLabels: Record<DevReviewKind, string> = {
      scenario: "场景",
      capacity: "容量",
      risk: "风险",
      cost: "成本"
    };
    const reviewGrid = row.createDiv({ cls: "goclaw-dev-review-grid" });
    (Object.keys(reviewLabels) as DevReviewKind[]).forEach((kind) => {
      const record = task.reviews[kind];
      const item = reviewGrid.createDiv({
        cls: `goclaw-dev-review-chip is-${record?.decision ?? "pending"}`
      });
      item.createSpan({ text: reviewLabels[kind] });
      item.createEl("strong", { text: record?.decision ?? "pending" });
    });

    const safety = row.createDiv({ cls: "goclaw-safety" });
    const shield = safety.createSpan();
    setIcon(shield, "shield-check");
    safety.createSpan({
      text: `最多 ${task.scope.max_changed_files} 文件 / ${task.scope.max_changed_lines} 行`
    });

    const fields = this.reviewFields(row, "开发任务");
    const actions = row.createDiv({ cls: "goclaw-actions" });
    if (task.status === "review_pending" || task.status === "blocked") {
      (Object.keys(reviewLabels) as DevReviewKind[])
        .filter((kind) => task.reviews[kind]?.decision !== "approved")
        .forEach((kind) => {
          this.actionButton(actions, `批准${reviewLabels[kind]}`, "", async () => {
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
      this.actionButton(actions, "冻结执行包", "mod-cta", async () => {
        await this.client.rpc("dev.task.freeze", {
          id: task.id,
          actor: this.plugin.actorId()
        });
        new Notice("开发任务已冻结，可进入开发页执行");
        await this.renderApprovals();
      });
    }
    if (task.status === "awaiting_acceptance") {
      this.actionButton(actions, "最终验收", "mod-cta", async () => {
        await this.client.rpc("dev.task.accept", {
          id: task.id,
          ...this.reviewPayload(fields, [`task:${task.id}`, task.last_evidence ?? ""], true)
        });
        new Notice("开发任务已通过最终验收");
        await this.renderApprovals();
      });
    }
    return row;
  }

  private async renderDevelopment(): Promise<void> {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "加载开发任务");
    try {
      const tasks = await this.client.rpc<DevTask[]>("dev.tasks", {
        project_id: this.plugin.settings.projectId
      });
      if (this.activeTab !== "development") return;
      target.empty();
      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "任务", tasks.length);
      this.metric(summary, "执行中", tasks.filter((task) =>
        ["running", "checking"].includes(task.status)).length);
      this.metric(summary, "待验收", tasks.filter((task) =>
        task.status === "awaiting_acceptance").length);

      const list = target.createDiv({ cls: "goclaw-dev-list" });
      tasks.forEach((task) => {
        const card = list.createEl("article", { cls: `goclaw-dev-card is-${task.status}` });
        const heading = card.createDiv({ cls: "goclaw-dev-card-heading" });
        const info = heading.createDiv();
        info.createEl("strong", { text: task.title });
        info.createSpan({ text: `${task.id} · rev ${task.compile.revision}` });
        heading.createSpan({ cls: `goclaw-state is-${task.status}`, text: task.status });
        card.createEl("p", { text: safeExcerpt(task.goal.objective, 150) });
        const meta = card.createDiv({ cls: "goclaw-dev-meta" });
        meta.createSpan({ text: task.branch || task.compile.base_ref });
        meta.createSpan({ text: `${task.scope.max_changed_files} 文件上限` });
        meta.createSpan({ text: relativeTime(task.updated_at) });
        if (task.last_gate) {
          const gate = card.createDiv({
            cls: `goclaw-safety${task.last_gate.passed ? "" : " is-error"}`
          });
          const gateIcon = gate.createSpan();
          setIcon(gateIcon, task.last_gate.passed ? "badge-check" : "circle-x");
          gate.createSpan({
            text: task.last_gate.passed
              ? "DoneGate 已通过"
              : safeExcerpt(task.last_gate.reasons?.join("；"), 120)
          });
        }
        const actions = card.createDiv({ cls: "goclaw-actions" });
        if (task.status === "frozen") {
          this.actionButton(actions, "开始执行", "mod-cta", async () => {
            await this.client.rpc("dev.task.enqueue", {
              task_id: task.id,
              capabilities: ["codex"]
            });
            new Notice("冻结版本已进入工作站队列");
            await this.renderDevelopment();
          });
        }
        if (["repair_pending", "failed"].includes(task.status)) {
          this.actionButton(actions, "创建修复版本", "mod-cta", async () => {
            await this.client.rpc("dev.task.revise", {
              id: task.id,
              expected_revision: task.compile.revision,
              reason: "Workstation DoneGate 未通过，创建新 revision 并重新履行四类评审。"
            });
            new Notice("已创建新 revision；重新完成四类评审、冻结并入队");
            await this.renderDevelopment();
          });
        }
      });
      if (tasks.length === 0) {
        this.renderEmpty(target, "git-branch-plus", "还没有开发任务", "先使用 goclaw dev create 编译任务契约。");
      }
    } catch (error) {
      if (this.activeTab === "development") this.renderError(target, error);
    }
  }

  private async renderTeam(): Promise<void> {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    const project = { project_id: this.plugin.settings.projectId };
    this.renderLoading(target, "加载团队控制面");
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
    ] as const);
    if (this.activeTab !== "team") return;

    target.empty();
    const members = membersResult.status === "fulfilled"
      ? collectionItems(membersResult.value)
      : [];
    const work = workResult.status === "fulfilled"
      ? collectionItems(workResult.value)
      : [];
    const issues = issuesResult.status === "fulfilled"
      ? collectionItems(issuesResult.value)
      : [];
    const runners = runnersResult.status === "fulfilled"
      ? collectionItems(runnersResult.value)
      : [];
    const policy = policyResult.status === "fulfilled" ? policyResult.value : null;
    const docs = docsResult.status === "fulfilled" ? docsResult.value : null;
    const components = componentsResult.status === "fulfilled" ? componentsResult.value : null;

    const intro = target.createDiv({ cls: "goclaw-team-intro" });
    const introIcon = intro.createSpan();
    setIcon(introIcon, "users-round");
    const introText = intro.createDiv();
    introText.createEl("strong", { text: "团队只读控制台" });
    introText.createEl("p", {
      text: "成员、任务、Bug、Runner、策略、文档与组件均由 Gateway 按当前项目授权后返回。"
    });

    const activeWork = work.filter((item) =>
      ["ready", "in_progress", "in_review", "blocked"].includes(item.status)).length;
    const openIssues = issues.filter((issue) =>
      !["resolved", "closed"].includes(issue.status)).length;
    const onlineRunners = runners.filter((runner) =>
      ["online", "busy", "draining"].includes(runner.status)).length;
    const summary = target.createDiv({ cls: "goclaw-team-summary" });
    this.metric(summary, "成员", members.length);
    this.metric(summary, "活动任务", activeWork, work.some((item) => item.status === "blocked"));
    this.metric(summary, "未关闭 Bug", openIssues, issues.some((issue) =>
      ["critical", "high"].includes(issue.severity) &&
      !["resolved", "closed"].includes(issue.status)));
    this.metric(summary, "在线 Runner", onlineRunners, runners.some((runner) => {
      const leaseTone = leaseState(runner.lease?.expires_at).tone;
      return runner.status !== "offline" &&
        (!runner.lease || leaseTone === "warning" || leaseTone === "danger");
    }));

    const failures: Array<[string, PromiseRejectedResult]> = [];
    ([
      ["成员负载", membersResult],
      ["项目任务", workResult],
      ["Bug", issuesResult],
      ["Runner", runnersResult],
      ["策略", policyResult],
      ["文档", docsResult],
      ["组件", componentsResult]
    ] as const).forEach(([label, result]) => {
      if (result.status === "rejected") failures.push([label, result]);
    });
    if (failures.length > 0) {
      const warning = target.createDiv({ cls: "goclaw-team-warning" });
      const warningIcon = warning.createSpan();
      setIcon(warningIcon, "triangle-alert");
      const warningText = warning.createDiv();
      warningText.createEl("strong", { text: `${failures.length} 个模块暂不可用` });
      warningText.createEl("p", {
        text: failures.map(([label, result]) =>
          `${label}：${result.reason instanceof Error ? result.reason.message : String(result.reason)}`
        ).join("；")
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

  private renderTeamMembers(
    parent: HTMLElement,
    members: TeamMember[],
    failed: boolean
  ): void {
    const list = this.teamSection(parent, "成员负载", members.length);
    if (failed) return this.renderTeamModuleError(list, "成员负载");
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
        ].filter(Boolean).join(" · ") || member.id
      });
      const state = memberState(member.status ?? "offline");
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
      const utilization = clampPercent(member.capacity?.utilization_percent);
      const load = body.createDiv({ cls: "goclaw-team-load" });
      const loadLabel = load.createDiv();
      loadLabel.createSpan({
        text: `${member.capacity?.active_work ?? 0} 进行中 · ${member.capacity?.queued_work ?? 0} 排队`
      });
      loadLabel.createEl("strong", { text: `${utilization}%` });
      const track = load.createDiv({ cls: "goclaw-team-load-track" });
      track.createDiv({ cls: "goclaw-team-load-fill" })
        .style.setProperty("width", `${utilization}%`);
      if ((member.capacity?.blocked_work ?? 0) > 0) {
        body.createDiv({
          cls: "goclaw-team-note is-danger",
          text: `${member.capacity?.blocked_work} 个任务受阻`
        });
      } else if (member.last_seen_at) {
        body.createDiv({
          cls: "goclaw-team-note",
          text: `最后活动 ${relativeTime(member.last_seen_at)}`
        });
      }
    });
    if (members.length === 0) {
      this.renderTeamEmpty(list, "尚未登记项目成员");
    }
  }

  private renderTeamWork(
    parent: HTMLElement,
    work: TeamWorkItem[],
    memberNames: Map<string, string>,
    failed: boolean
  ): void {
    const active = work.filter((item) => item.status !== "done" && item.status !== "cancelled");
    const list = this.teamSection(parent, "项目任务", active.length);
    if (failed) return this.renderTeamModuleError(list, "项目任务");
    active.slice(0, 10).forEach((item) => {
      const card = list.createEl("article", { cls: "goclaw-team-row" });
      const heading = card.createDiv({ cls: "goclaw-team-card-heading" });
      const text = heading.createDiv();
      text.createEl("strong", { text: item.title });
      text.createSpan({
        text: [
          item.kind,
          item.business_domain,
          item.assignee_id ? memberNames.get(item.assignee_id) ?? item.assignee_id : "未分配"
        ].filter(Boolean).join(" · ")
      });
      const state = workState(item.status);
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
      const links = [
        item.id,
        item.task_id,
        item.issue_id,
        ...(item.source_refs ?? []).slice(0, 2)
      ].filter(Boolean);
      card.createDiv({ cls: "goclaw-team-links", text: links.join(" · ") });
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
    if (active.length === 0) this.renderTeamEmpty(list, "当前没有活动任务");
  }

  private renderTeamIssues(
    parent: HTMLElement,
    issues: TeamIssue[],
    memberNames: Map<string, string>,
    failed: boolean
  ): void {
    const open = issues.filter((issue) => issue.status !== "resolved" && issue.status !== "closed");
    const list = this.teamSection(parent, "Bug 状态", open.length);
    if (failed) return this.renderTeamModuleError(list, "Bug");
    open.slice(0, 8).forEach((issue) => {
      const card = list.createEl("article", { cls: "goclaw-team-row" });
      const heading = card.createDiv({ cls: "goclaw-team-card-heading" });
      const text = heading.createDiv();
      text.createEl("strong", { text: issue.title });
      text.createSpan({
        text: `${issue.id} · ${issue.owner_id
          ? memberNames.get(issue.owner_id) ?? issue.owner_id
          : "未分配"}`
      });
      const status = issueState(issue.status);
      heading.createSpan({ cls: teamStateClass(status), text: status.label });
      const metadata = card.createDiv({ cls: "goclaw-team-meta-line" });
      const severity = severityState(issue.severity);
      metadata.createSpan({ cls: teamStateClass(severity), text: `严重度 ${severity.label}` });
      if (issue.work_item_id) metadata.createSpan({ text: issue.work_item_id });
      if (issue.regression_case_id) metadata.createSpan({ text: `回归 ${issue.regression_case_id}` });
      if (issue.updated_at) {
        card.createDiv({ cls: "goclaw-team-note", text: relativeTime(issue.updated_at) });
      }
    });
    if (open.length > 8) this.renderTeamMore(list, open.length - 8);
    if (open.length === 0) this.renderTeamEmpty(list, "没有未关闭 Bug");
  }

  private renderTeamRunners(
    parent: HTMLElement,
    runners: TeamRunner[],
    memberNames: Map<string, string>,
    failed: boolean
  ): void {
    const list = this.teamSection(parent, "Runner 在线与租约", runners.length);
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
        ].filter(Boolean).join(" · ") || "未绑定任务"
      });
      const badges = card.createDiv({ cls: "goclaw-team-runner-states" });
      badges.createSpan({ cls: teamStateClass(status), text: status.label });
      const lease = leaseState(runner.lease?.expires_at);
      badges.createSpan({ cls: teamStateClass(lease), text: lease.label });
      if (runner.lease?.expires_at) {
        badges.setAttribute("title", `租约到期：${runner.lease.expires_at}`);
      }
    });
    if (runners.length === 0) this.renderTeamEmpty(list, "当前没有登记的 Runner");
  }

  private renderTeamPolicy(
    parent: HTMLElement,
    policy: TeamPolicyStatus | null,
    failed: boolean
  ): void {
    const list = this.teamSection(parent, "生效策略", policy?.layers?.length ?? 0);
    if (failed) return this.renderTeamModuleError(list, "策略");
    if (!policy) return this.renderTeamEmpty(list, "当前项目没有策略状态");
    const facts = list.createDiv({ cls: "goclaw-team-facts" });
    this.fact(facts, "有效版本", policy.effective_version || "未锁定");
    this.fact(facts, "合规", policy.compliant ? "通过" : "未通过");
    this.fact(facts, "漂移", String(policy.drift_count ?? 0));
    this.fact(facts, "检查", policy.checked_at ? relativeTime(policy.checked_at) : "未知");
    (policy.layers ?? []).forEach((layer) => {
      const row = list.createDiv({ cls: "goclaw-team-policy-layer" });
      row.createSpan({ text: layer.scope });
      row.createEl("strong", { text: `${layer.id}@${layer.version}` });
      const state = layer.compliant === false
        ? { label: "漂移", tone: "danger" as const }
        : { label: "一致", tone: "success" as const };
      row.createSpan({ cls: teamStateClass(state), text: state.label });
    });
    (policy.violations ?? []).slice(0, 5).forEach((violation) => {
      list.createDiv({
        cls: `goclaw-team-note is-${violation.severity === "warning" ? "warning" : "danger"}`,
        text: `${violation.code} · ${violation.message}`
      });
    });
  }

  private renderTeamDocs(
    parent: HTMLElement,
    docs: TeamDocsSummary | null,
    failed: boolean
  ): void {
    const list = this.teamSection(parent, "方案文档", docs?.total ?? 0);
    if (failed) return this.renderTeamModuleError(list, "文档");
    if (!docs) return this.renderTeamEmpty(list, "当前项目没有文档概览");
    const facts = list.createDiv({ cls: "goclaw-team-facts" });
    this.fact(facts, "已批准", String(docs.approved ?? 0));
    this.fact(facts, "待复核", String(docs.review_due ?? 0));
    this.fact(facts, "陈旧", String(docs.stale ?? 0));
    this.fact(facts, "未关联任务", String(docs.unlinked ?? 0));
    (docs.items ?? []).slice(0, 6).forEach((document) =>
      this.renderTeamDocument(list, document));
    if ((docs.items?.length ?? 0) === 0) {
      this.renderTeamEmpty(list, "没有返回最近文档");
    }
  }

  private renderTeamDocument(parent: HTMLElement, document: TeamDocument): void {
    const row = parent.createEl("article", { cls: "goclaw-team-row" });
    const heading = row.createDiv({ cls: "goclaw-team-card-heading" });
    const text = heading.createDiv();
    text.createEl("strong", { text: document.title || document.path });
    text.createSpan({
      text: [
        document.kind,
        document.owner_id,
        `${document.linked_work_ids?.length ?? 0} 个任务关联`
      ].filter(Boolean).join(" · ")
    });
    if (document.status) {
      const state = document.status === "approved"
        ? { label: "已批准", tone: "success" as const }
        : document.status === "stale"
          ? { label: "陈旧", tone: "danger" as const }
          : { label: document.status === "review" ? "评审中" : "草稿", tone: "warning" as const };
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
    }
    row.createDiv({ cls: "goclaw-team-links", text: document.path });
  }

  private renderTeamComponents(
    parent: HTMLElement,
    components: TeamComponentsSummary | null,
    failed: boolean
  ): void {
    const list = this.teamSection(parent, "共享组件", components?.total ?? 0);
    if (failed) return this.renderTeamModuleError(list, "组件");
    if (!components) return this.renderTeamEmpty(list, "当前项目没有组件概览");
    const facts = list.createDiv({ cls: "goclaw-team-facts" });
    this.fact(facts, "可复用", String(components.reusable ?? 0));
    this.fact(facts, "待评审", String(components.pending_review ?? 0));
    this.fact(facts, "已弃用", String(components.deprecated ?? 0));
    this.fact(facts, "总计", String(components.total ?? 0));
    (components.items ?? []).slice(0, 6).forEach((component) =>
      this.renderTeamComponent(list, component));
    if ((components.items?.length ?? 0) === 0) {
      this.renderTeamEmpty(list, "没有返回推荐组件");
    }
  }

  private renderTeamComponent(parent: HTMLElement, component: TeamComponent): void {
    const row = parent.createEl("article", { cls: "goclaw-team-row" });
    const heading = row.createDiv({ cls: "goclaw-team-card-heading" });
    const text = heading.createDiv();
    text.createEl("strong", { text: component.name });
    text.createSpan({
      text: [
        component.kind,
        component.version,
        component.owner_id
      ].filter(Boolean).join(" · ") || component.id
    });
    if (component.status) {
      const state = component.status === "active"
        ? { label: "可用", tone: "success" as const }
        : component.status === "deprecated"
          ? { label: "弃用", tone: "danger" as const }
          : { label: "实验", tone: "warning" as const };
      heading.createSpan({ cls: teamStateClass(state), text: state.label });
    }
    row.createDiv({
      cls: "goclaw-team-links",
      text: `${component.id} · 已复用 ${component.reuse_count ?? 0} 次`
    });
  }

  private teamSection(parent: HTMLElement, title: string, count: number): HTMLElement {
    const section = parent.createDiv({ cls: "goclaw-section goclaw-team-section" });
    const heading = section.createDiv({ cls: "goclaw-section-heading" });
    heading.createEl("h3", { text: title });
    heading.createSpan({ cls: "goclaw-count", text: String(count) });
    return section.createDiv({ cls: "goclaw-team-list" });
  }

  private renderTeamModuleError(parent: HTMLElement, label: string): void {
    parent.createDiv({
      cls: "goclaw-team-empty is-error",
      text: `${label}接口暂不可用；其他团队模块仍可继续查看。`
    });
  }

  private renderTeamEmpty(parent: HTMLElement, text: string): void {
    parent.createDiv({ cls: "goclaw-team-empty", text });
  }

  private renderTeamMore(parent: HTMLElement, count: number): void {
    parent.createDiv({ cls: "goclaw-team-more", text: `还有 ${count} 项，请在对应系统查看完整列表` });
  }

  private async renderProgress(): Promise<void> {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "加载运行轨迹");
    try {
      const traces = await this.client.rpc<HarnessTrace[]>("harness.traces", {
        project_id: this.plugin.settings.projectId,
        limit: 40
      });
      if (this.activeTab !== "progress") return;
      target.empty();
      const complete = traces.filter((trace) => trace.status === "completed").length;
      const failed = traces.filter((trace) => trace.status !== "completed").length;
      const summary = target.createDiv({ cls: "goclaw-progress-summary" });
      this.metric(summary, "运行", traces.length);
      this.metric(summary, "完成", complete);
      this.metric(summary, "异常", failed, failed > 0);
      const list = target.createDiv({ cls: "goclaw-run-list" });
      traces.forEach((trace) => {
        const row = list.createDiv({ cls: `goclaw-run is-${trace.status}` });
        const dot = row.createSpan({ cls: "goclaw-run-dot" });
        dot.setAttribute("aria-label", trace.status);
        const content = row.createDiv({ cls: "goclaw-run-content" });
        content.createEl("strong", {
          text: safeExcerpt(trace.output || trace.input || trace.error, 90) || "无摘要"
        });
        content.createSpan({
          text: `${trace.harness_version || "unversioned"} · ${trace.duration_ms} ms · ${relativeTime(trace.started_at)}`
        });
      });
      if (traces.length === 0) {
        this.renderEmpty(target, "activity", "还没有运行轨迹", "从 Obsidian 或飞书发起一次项目会话即可生成。");
      }
    } catch (error) {
      if (this.activeTab === "progress") this.renderError(target, error);
    }
  }

  private async renderHarness(): Promise<void> {
    if (!this.bodyEl) return;
    const target = this.bodyEl;
    this.renderLoading(target, "加载 Harness 状态");
    try {
      const [status, experiments] = await Promise.all([
        this.client.rpc<HarnessStatus>("harness.status"),
        this.client.rpc<HarnessExperiment[]>("harness.experiments")
      ]);
      if (this.activeTab !== "harness") return;
      target.empty();
      const hero = target.createDiv({ cls: "goclaw-harness-hero" });
      hero.createSpan({ text: "当前版本" });
      hero.createEl("strong", { text: status.active.version });
      hero.createEl("p", { text: status.manifest.description || status.manifest.name });
      const facts = target.createDiv({ cls: "goclaw-facts" });
      this.fact(facts, "模型", status.manifest.model_profile || "配置默认值");
      this.fact(facts, "Golden 门槛", `${Math.round(status.manifest.minimum_golden * 100)}%`);
      this.fact(facts, "Holdout 门槛", `${Math.round(status.manifest.minimum_holdout * 100)}%`);
      this.fact(facts, "组件", String(Object.keys(status.manifest.components).length));

      const section = target.createDiv({ cls: "goclaw-section" });
      section.createEl("h3", { text: "最近实验" });
      experiments.slice(0, 8).forEach((experiment) => {
        const row = section.createDiv({ cls: "goclaw-experiment-row" });
        const info = row.createDiv();
        info.createEl("strong", { text: experiment.candidate_version });
        info.createSpan({ text: safeExcerpt(experiment.change_manifest.change_summary, 80) });
        row.createSpan({ cls: `goclaw-state is-${experiment.status}`, text: experiment.status });
      });
      if (status.active.previous_version) {
        const fields = this.reviewFields(target, "Harness 回滚");
        this.actionButton(target, `回滚到 ${status.active.previous_version}`, "", async () => {
          await this.client.rpc("harness.rollback", {
            ...this.reviewPayload(
              fields,
              [`harness:${status.active.version}`, `rollback:${status.active.previous_version}`],
              true
            )
          });
          new Notice("Harness 已回滚");
          await this.renderHarness();
        });
      }
    } catch (error) {
      if (this.activeTab === "harness") this.renderError(target, error);
    }
  }

  private actionButton(parent: HTMLElement, label: string, cls: string, action: () => Promise<void>): void {
    const button = parent.createEl("button", { text: label, cls });
    button.addEventListener("click", () => {
      button.disabled = true;
      void action().catch((error) => {
        new Notice(error instanceof Error ? error.message : String(error));
      }).finally(() => {
        button.disabled = false;
      });
    });
  }

  private reviewFields(parent: HTMLElement, subject: string): ReviewFields {
    const rationale = parent.createEl("textarea", {
      cls: "goclaw-review-comment",
      attr: {
        rows: "2",
        placeholder: `必填：${subject}决策依据、已核对证据与边界…`,
        "aria-label": `${subject} review rationale`
      }
    });
    const counterargument = parent.createEl("textarea", {
      cls: "goclaw-review-comment",
      attr: {
        rows: "2",
        placeholder: "批准时必填：这个决定最可能错在哪里？",
        "aria-label": `${subject} strongest counterargument`
      }
    });
    return { rationale, counterargument };
  }

  private reviewPayload(
    fields: ReviewFields,
    evidenceRefs: string[],
    approval: boolean
  ): Record<string, unknown> {
    const rationale = fields.rationale.value.trim();
    const counterargument = fields.counterargument.value.trim();
    if (!rationale) throw new Error("请填写可审计的决策依据");
    if (approval && !counterargument) {
      throw new Error("批准前请填写最强反对理由，避免确认偏差");
    }
    return this.plugin.governanceParams(
      rationale,
      counterargument,
      evidenceRefs.filter(Boolean)
    );
  }

  private metric(parent: HTMLElement, label: string, value: number, warning = false): void {
    const item = parent.createDiv({ cls: `goclaw-metric${warning ? " is-warning" : ""}` });
    item.createEl("strong", { text: String(value) });
    item.createSpan({ text: label });
  }

  private fact(parent: HTMLElement, label: string, value: string): void {
    const item = parent.createDiv({ cls: "goclaw-fact" });
    item.createSpan({ text: label });
    item.createEl("strong", { text: value });
  }

  private renderLoading(parent: HTMLElement, label: string): void {
    parent.empty();
    const loading = parent.createDiv({ cls: "goclaw-loading" });
    const icon = loading.createSpan();
    setIcon(icon, "loader-circle");
    loading.createSpan({ text: label });
  }

  private renderError(parent: HTMLElement, error: unknown): void {
    parent.empty();
    this.renderEmpty(
      parent,
      "circle-alert",
      "加载失败",
      error instanceof Error ? error.message : String(error)
    );
  }

  private renderEmpty(parent: HTMLElement, iconName: string, title: string, description: string): void {
    const empty = parent.createDiv({ cls: "goclaw-empty" });
    const icon = empty.createSpan();
    setIcon(icon, iconName);
    empty.createEl("strong", { text: title });
    empty.createEl("p", { text: description });
  }

  private updateStatus(state: ConnectionState, detail?: string): void {
    if (!this.statusEl) return;
    const status = this.statusEl.parentElement;
    status?.removeClass("is-connected", "is-error", "is-connecting");
    if (state === "connected") {
      status?.addClass("is-connected");
      this.statusEl.setText("已连接");
    } else if (state === "connecting") {
      status?.addClass("is-connecting");
      this.statusEl.setText("连接中");
    } else if (state === "error") {
      status?.addClass("is-error");
      this.statusEl.setText("连接错误");
    } else {
      this.statusEl.setText("未连接");
    }
    this.statusEl.setAttribute("title", detail ?? "");
  }
}

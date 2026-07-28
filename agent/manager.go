package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/goclaw/agent/tools"
	"github.com/smallnest/goclaw/bus"
	"github.com/smallnest/goclaw/channels"
	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/harness"
	"github.com/smallnest/goclaw/internal/logger"
	"github.com/smallnest/goclaw/memory/catalog"
	"github.com/smallnest/goclaw/providers"
	"github.com/smallnest/goclaw/session"
	"go.uber.org/zap"
)

var cronJobIDPattern = regexp.MustCompile(`\bjob-[a-zA-Z0-9]+\b`)
var cronListLinePattern = regexp.MustCompile(`^(job-[a-zA-Z0-9]+)\s+\((enabled|disabled)\)$`)

// AgentManager 管理多个 Agent 实例
type AgentManager struct {
	agents         map[string]*Agent        // agentID -> Agent
	bindings       map[string]*BindingEntry // channel:accountID -> BindingEntry
	defaultAgent   *Agent                   // 默认 Agent
	bus            *bus.MessageBus
	sessionMgr     *session.Manager
	provider       providers.Provider
	tools          *ToolRegistry
	mu             sync.RWMutex
	cfg            *config.Config
	contextBuilder *ContextBuilder
	skillsLoader   *SkillsLoader
	helper         *AgentHelper
	channelMgr     *channels.Manager
	manualCronMu   sync.Mutex
	manualCronLast map[string]time.Time
	harnessSvc     *harness.Service
	// 分身支持
	subagentRegistry  *SubagentRegistry
	subagentAnnouncer *SubagentAnnouncer
	dataDir           string
}

// BindingEntry Agent 绑定条目
type BindingEntry struct {
	AgentID   string
	Channel   string
	AccountID string
	Agent     *Agent
}

// NewAgentManagerConfig AgentManager 配置
type NewAgentManagerConfig struct {
	Bus            *bus.MessageBus
	Provider       providers.Provider
	SessionMgr     *session.Manager
	Tools          *ToolRegistry
	DataDir        string          // 数据目录，用于存储分身注册表
	ContextBuilder *ContextBuilder // 上下文构建器
	SkillsLoader   *SkillsLoader   // 技能加载器
	ChannelMgr     *channels.Manager
	Harness        *harness.Service
}

// NewAgentManager 创建 Agent 管理器
func NewAgentManager(cfg *NewAgentManagerConfig) *AgentManager {
	// 创建分身注册表
	subagentRegistry := NewSubagentRegistry(cfg.DataDir)

	// 创建分身宣告器
	subagentAnnouncer := NewSubagentAnnouncer(nil) // 回调在 Start 中设置

	return &AgentManager{
		agents:            make(map[string]*Agent),
		bindings:          make(map[string]*BindingEntry),
		bus:               cfg.Bus,
		sessionMgr:        cfg.SessionMgr,
		provider:          cfg.Provider,
		tools:             cfg.Tools,
		subagentRegistry:  subagentRegistry,
		subagentAnnouncer: subagentAnnouncer,
		dataDir:           cfg.DataDir,
		contextBuilder:    cfg.ContextBuilder,
		skillsLoader:      cfg.SkillsLoader,
		helper:            NewAgentHelper(cfg.SessionMgr),
		channelMgr:        cfg.ChannelMgr,
		manualCronLast:    make(map[string]time.Time),
		harnessSvc:        cfg.Harness,
	}
}

// handleSubagentCompletion 处理分身完成事件
func (m *AgentManager) handleSubagentCompletion(runID string, record *SubagentRunRecord) {

	// 启动宣告流程
	if record.Outcome != nil {
		announceParams := &SubagentAnnounceParams{
			ChildSessionKey:     record.ChildSessionKey,
			ChildRunID:          record.RunID,
			RequesterSessionKey: record.RequesterSessionKey,
			RequesterOrigin:     record.RequesterOrigin,
			RequesterDisplayKey: record.RequesterDisplayKey,
			Task:                record.Task,
			Label:               record.Label,
			StartedAt:           record.StartedAt,
			EndedAt:             record.EndedAt,
			Outcome:             record.Outcome,
			Cleanup:             record.Cleanup,
			AnnounceType:        SubagentAnnounceTypeTask,
		}

		if err := m.subagentAnnouncer.RunAnnounceFlow(announceParams); err != nil {
			logger.Error("Failed to announce subagent result",
				zap.String("run_id", runID),
				zap.Error(err))
		}

		// 标记清理完成
		m.subagentRegistry.Cleanup(runID, record.Cleanup, true)
	}
}

// SetupFromConfig 从配置设置 Agent 和绑定
func (m *AgentManager) SetupFromConfig(cfg *config.Config, contextBuilder *ContextBuilder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cfg = cfg
	m.contextBuilder = contextBuilder

	logger.Info("Setting up agents from config")

	// 1. 创建 Agent 实例
	for _, agentCfg := range cfg.Agents.List {
		if err := m.createAgent(agentCfg, contextBuilder, cfg); err != nil {
			logger.Error("Failed to create agent",
				zap.String("agent_id", agentCfg.ID),
				zap.Error(err))
			continue
		}
	}

	// 2. 如果没有配置 Agent，创建默认 Agent
	if len(m.agents) == 0 {
		logger.Info("No agents configured, creating default agent")
		defaultAgentCfg := config.AgentConfig{
			ID:        "default",
			Name:      "Default Agent",
			Default:   true,
			Model:     cfg.Agents.Defaults.Model.Effective(),
			Workspace: cfg.Workspace.Path,
		}
		if err := m.createAgent(defaultAgentCfg, contextBuilder, cfg); err != nil {
			return fmt.Errorf("failed to create default agent: %w", err)
		}
	}

	// 3. 设置绑定
	for _, binding := range cfg.Bindings {
		if err := m.setupBinding(binding); err != nil {
			logger.Error("Failed to setup binding",
				zap.String("agent_id", binding.AgentID),
				zap.String("channel", binding.Match.Channel),
				zap.String("account_id", binding.Match.AccountID),
				zap.Error(err))
		}
	}

	// 4. 设置分身支持
	m.setupSubagentSupport(cfg, contextBuilder)

	logger.Info("Agent manager setup complete",
		zap.Int("agents", len(m.agents)),
		zap.Int("bindings", len(m.bindings)))

	return nil
}

// setupSubagentSupport 设置分身支持
func (m *AgentManager) setupSubagentSupport(cfg *config.Config, contextBuilder *ContextBuilder) {
	// 加载分身注册表
	if err := m.subagentRegistry.LoadFromDisk(); err != nil {
		logger.Warn("Failed to load subagent registry", zap.Error(err))
	}

	// 设置分身运行完成回调
	m.subagentRegistry.SetOnRunComplete(func(runID string, record *SubagentRunRecord) {
		m.handleSubagentCompletion(runID, record)
	})

	// 更新宣告器回调
	m.subagentAnnouncer = NewSubagentAnnouncer(func(sessionKey, message string) error {
		// 发送宣告消息到指定会话
		return m.sendToSession(sessionKey, message)
	})

	// 创建分身注册表适配器
	registryAdapter := &subagentRegistryAdapter{registry: m.subagentRegistry}

	// 注册 sessions_spawn 工具
	spawnTool := tools.NewSubagentSpawnTool(registryAdapter)
	spawnTool.SetAgentConfigGetter(func(agentID string) *config.AgentConfig {
		for _, agentCfg := range cfg.Agents.List {
			if agentCfg.ID == agentID {
				return &agentCfg
			}
		}
		return nil
	})
	spawnTool.SetDefaultConfigGetter(func() *config.AgentDefaults {
		return &cfg.Agents.Defaults
	})
	spawnTool.SetAgentIDGetter(func(sessionKey string) string {
		// 从会话密钥中解析 agent ID
		agentID, _, _ := ParseAgentSessionKey(sessionKey)
		if agentID == "" {
			// 尝试从绑定中查找
			for _, entry := range m.bindings {
				if entry.Agent != nil {
					return entry.AgentID
				}
			}
		}
		return agentID
	})
	spawnTool.SetOnSpawn(func(result *tools.SubagentSpawnResult) error {
		return m.handleSubagentSpawn(result)
	})

	// 注册工具
	if err := m.tools.RegisterExisting(spawnTool); err != nil {
		logger.Error("Failed to register sessions_spawn tool", zap.Error(err))
	}

	logger.Info("Subagent support configured")
}

// subagentRegistryAdapter 分身注册表适配器
type subagentRegistryAdapter struct {
	registry *SubagentRegistry
}

// RegisterRun 注册分身运行
func (a *subagentRegistryAdapter) RegisterRun(params *tools.SubagentRunParams) error {
	// 转换 RequesterOrigin
	var requesterOrigin *DeliveryContext
	if params.RequesterOrigin != nil {
		requesterOrigin = &DeliveryContext{
			Channel:   params.RequesterOrigin.Channel,
			AccountID: params.RequesterOrigin.AccountID,
			To:        params.RequesterOrigin.To,
			ThreadID:  params.RequesterOrigin.ThreadID,
		}
	}

	return a.registry.RegisterRun(&SubagentRunParams{
		RunID:               params.RunID,
		ChildSessionKey:     params.ChildSessionKey,
		RequesterSessionKey: params.RequesterSessionKey,
		RequesterOrigin:     requesterOrigin,
		RequesterDisplayKey: params.RequesterDisplayKey,
		Task:                params.Task,
		Cleanup:             params.Cleanup,
		Label:               params.Label,
		ArchiveAfterMinutes: params.ArchiveAfterMinutes,
	})
}

// handleSubagentSpawn 处理分身生成
func (m *AgentManager) handleSubagentSpawn(result *tools.SubagentSpawnResult) error {
	// 解析子会话密钥
	agentID, subagentID, isSubagent := ParseAgentSessionKey(result.ChildSessionKey)
	if !isSubagent {
		return fmt.Errorf("invalid subagent session key: %s", result.ChildSessionKey)
	}

	// Get the agent to use for this subagent
	var agent *Agent
	if agentID != "" {
		var ok bool
		agent, ok = m.GetAgent(agentID)
		if !ok {
			agent = m.GetDefaultAgent()
		}
	} else {
		agent = m.GetDefaultAgent()
	}

	if agent == nil {
		return fmt.Errorf("no agent available for subagent: %s", result.ChildSessionKey)
	}

	// Set the system prompt if provided
	if result.ChildSystemPrompt != "" {
		// For subagent, we need to pass this through context
		// This will be used when the subagent processes messages
		logger.Debug("Subagent system prompt set",
			zap.String("run_id", result.RunID),
			zap.String("subagent_id", subagentID),
			zap.Int("prompt_length", len(result.ChildSystemPrompt)))
	}

	logger.Debug("Subagent spawn handled",
		zap.String("run_id", result.RunID),
		zap.String("subagent_id", subagentID),
		zap.String("child_session_key", result.ChildSessionKey))

	return nil
}

// sendToSession 发送消息到指定会话
func (m *AgentManager) sendToSession(sessionKey, message string) error {
	// Parse session key to get delivery context
	// Format: agent:<agentId>:subagent:<uuid> or agent:<agentId>:<sessionKey>
	parts := strings.Split(sessionKey, ":")
	if len(parts) < 3 {
		return fmt.Errorf("invalid session key format: %s", sessionKey)
	}

	// Extract channel and chat_id from session key
	// For now, we publish to CLI as default
	// In a real implementation, this should route to the appropriate channel
	logger.Debug("Message sent to session",
		zap.String("session_key", sessionKey),
		zap.Int("message_length", len(message)))

	// Publish the message as an outbound message
	// The message will be delivered to the user via the configured channel
	outbound := &bus.OutboundMessage{
		Channel:   "cli", // Default to CLI, could be extracted from session key
		ChatID:    sessionKey,
		Content:   message,
		Timestamp: time.Now(),
	}

	ctx := context.Background()
	if err := m.bus.PublishOutbound(ctx, outbound); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// createAgent 创建 Agent 实例
func (m *AgentManager) createAgent(cfg config.AgentConfig, contextBuilder *ContextBuilder, globalCfg *config.Config) error {
	// 获取 workspace 路径
	workspace := cfg.Workspace
	if workspace == "" {
		workspace = globalCfg.Workspace.Path
	}

	// 获取模型
	model := cfg.Model
	if model == "" {
		model = globalCfg.Agents.Defaults.Model.Effective()
	}

	// 获取最大迭代次数
	maxIterations := globalCfg.Agents.Defaults.MaxIterations
	if maxIterations == 0 {
		maxIterations = 15
	}

	// 获取最大历史消息数
	maxHistoryMessages := globalCfg.Agents.Defaults.MaxHistoryMessages
	if maxHistoryMessages == 0 {
		maxHistoryMessages = 100
	}

	// 获取重试配置
	var retryConfig *RetryConfig
	if globalCfg.Agents.Defaults.Retry != nil {
		retryConfig = convertRetryConfig(globalCfg.Agents.Defaults.Retry)
	}

	// 创建 Agent
	agent, err := NewAgent(&NewAgentConfig{
		Bus:                m.bus,
		Provider:           m.provider,
		SessionMgr:         m.sessionMgr,
		Tools:              m.tools,
		Context:            contextBuilder,
		Workspace:          workspace,
		MaxIteration:       maxIterations,
		MaxHistoryMessages: maxHistoryMessages,
		SkillsLoader:       m.skillsLoader,
		Retry:              retryConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create agent %s: %w", cfg.ID, err)
	}

	// 设置系统提示词
	if cfg.SystemPrompt != "" {
		agent.SetSystemPrompt(cfg.SystemPrompt)
	}

	// 存储到管理器
	m.agents[cfg.ID] = agent

	// 如果是默认 Agent，设置默认
	if cfg.Default {
		m.defaultAgent = agent
	}

	logger.Info("Agent created",
		zap.String("agent_id", cfg.ID),
		zap.String("name", cfg.Name),
		zap.String("workspace", workspace),
		zap.String("model", model),
		zap.Bool("is_default", cfg.Default))

	return nil
}

// setupBinding 设置 Agent 绑定
func (m *AgentManager) setupBinding(binding config.BindingConfig) error {
	// 获取 Agent
	agent, ok := m.agents[binding.AgentID]
	if !ok {
		return fmt.Errorf("agent not found: %s", binding.AgentID)
	}

	// 构建绑定键
	bindingKey := fmt.Sprintf("%s:%s", binding.Match.Channel, binding.Match.AccountID)

	// 存储绑定
	m.bindings[bindingKey] = &BindingEntry{
		AgentID:   binding.AgentID,
		Channel:   binding.Match.Channel,
		AccountID: binding.Match.AccountID,
		Agent:     agent,
	}

	logger.Info("Binding setup",
		zap.String("binding_key", bindingKey),
		zap.String("agent_id", binding.AgentID))

	return nil
}

// RouteInbound 路由入站消息到对应的 Agent
func (m *AgentManager) RouteInbound(ctx context.Context, msg *bus.InboundMessage) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 构建绑定键
	bindingKey := fmt.Sprintf("%s:%s", msg.Channel, msg.AccountID)

	// 查找绑定的 Agent
	entry, ok := m.bindings[bindingKey]
	var agent *Agent
	if ok {
		agent = entry.Agent
		logger.Debug("Message routed by binding",
			zap.String("binding_key", bindingKey),
			zap.String("agent_id", entry.AgentID))
	} else if m.defaultAgent != nil {
		// 使用默认 Agent
		agent = m.defaultAgent
		logger.Debug("Message routed to default agent",
			zap.String("channel", msg.Channel),
			zap.String("account_id", msg.AccountID))
	} else {
		return fmt.Errorf("no agent found for message: %s", bindingKey)
	}

	// 处理消息
	return m.handleInboundMessage(ctx, msg, agent)
}

// handleInboundMessage 处理入站消息
func (m *AgentManager) handleInboundMessage(ctx context.Context, msg *bus.InboundMessage, agent *Agent) error {
	logger.Info("[Manager] Processing inbound message",
		zap.String("message_id", msg.ID),
		zap.String("channel", msg.Channel),
		zap.String("account_id", msg.AccountID),
		zap.String("chat_id", msg.ChatID),
		zap.String("content", msg.Content),
	)

	if handled, err := m.handleDirectCronOneShot(ctx, msg); handled {
		logger.Info("[Manager] Message handled by cron oneshot", zap.String("message_id", msg.ID))
		return err
	}

	runStarted := time.Now().UTC()
	harnessVersion, projectID, harnessFiles := m.contextBuilder.HarnessIdentity()
	projectScopedConversation := m.harnessSvc != nil
	if projectID == "" && m.harnessSvc != nil {
		projectID = m.harnessSvc.ProjectID()
	}
	if m.harnessSvc != nil {
		if routed := m.harnessSvc.ResolveProject(msg.Channel, msg.AccountID, msg.ChatID); routed != "" {
			projectID = routed
		}
	}
	if projectID == "" {
		projectID = "default"
	}
	topicID := "inbox"
	if msg.Metadata != nil {
		if value, ok := msg.Metadata["project_id"].(string); ok && strings.TrimSpace(value) != "" {
			projectID = value
			projectScopedConversation = true
		}
		if value, ok := msg.Metadata["topic_id"].(string); ok && strings.TrimSpace(value) != "" {
			topicID = value
		}
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata["project_id"] = projectID
	msg.Metadata["topic_id"] = topicID

	// 调用 Agent 处理消息（内部逻辑和 agent.go 中的 handleInboundMessage 类似）
	logger.Debug("[Manager] Routing to agent",
		zap.String("channel", msg.Channel),
		zap.String("account_id", msg.AccountID),
		zap.String("chat_id", msg.ChatID))

	// Project conversations use the versioned session authority. Legacy
	// channel sessions retain their existing account/chat boundary.
	sessionKey := ""
	var sess *session.Session
	var err error
	if projectScopedConversation {
		sess, topicID, err = m.sessionMgr.GetOrCreateProjectConversation(
			projectID,
			topicID,
		)
		if err == nil {
			sessionKey = sess.Key
			msg.Metadata["topic_id"] = topicID
		}
	} else {
		sessionKey = conversationSessionKey(
			msg,
			projectID,
			topicID,
			false,
		)
		sess, err = m.sessionMgr.GetOrCreate(sessionKey)
	}
	if err != nil {
		logger.Error("Failed to get project conversation", zap.Error(err))
		return err
	}
	if !projectScopedConversation && (msg.ChatID == "default" || msg.ChatID == "") {
		logger.Debug("[Manager] Creating fresh session", zap.String("session_key", sessionKey))
	}

	sessionMetadata := map[string]interface{}{
		"project_id":      projectID,
		"topic_id":        topicID,
		"conversation_id": fmt.Sprintf("%s:%s:%s", msg.Channel, msg.AccountID, msg.ChatID),
		"channel":         msg.Channel,
		"harness_version": harnessVersion,
	}
	sess.MergeMetadata(sessionMetadata)

	// 转换为 Agent 消息
	agentMsg := AgentMessage{
		Role:      RoleUser,
		Content:   []ContentBlock{TextContent{Text: msg.Content}},
		Timestamp: msg.Timestamp.UnixMilli(),
		Metadata: map[string]any{
			"project_id": projectID,
			"topic_id":   topicID,
		},
	}

	// 添加媒体内容
	for _, media := range msg.Media {
		if media.Type == "image" {
			agentMsg.Content = append(agentMsg.Content, ImageContent{
				URL:      media.URL,
				Data:     media.Base64,
				MimeType: media.MimeType,
			})
		}
	}

	// 获取 Agent 的 orchestrator
	orchestrator := agent.GetOrchestrator()
	agent.RefreshSystemPrompt()

	// 加载历史消息并添加当前消息
	// 使用配置的最大历史消息数限制，避免 token 超限
	// 使用 GetHistorySafe 确保不会在工具调用中间截断消息
	maxHistory := m.cfg.Agents.Defaults.MaxHistoryMessages
	if maxHistory <= 0 {
		maxHistory = 100 // 默认值
	}
	history := sess.GetHistorySafe(maxHistory)
	historyAgentMsgs := sessionMessagesToAgentMessages(history)
	allMessages := append(historyAgentMsgs, agentMsg)

	// 执行 Agent
	logger.Info("[Manager] Starting agent execution",
		zap.String("message_id", msg.ID),
		zap.Int("history_count", len(history)),
		zap.Int("total_messages", len(allMessages)),
	)

	// 启动事件监听 goroutine 来转发流式事件
	runID := uuid.New().String()
	eventSeq := 0
	go func() {
		for event := range orchestrator.Subscribe() {
			m.handleOrchestratorEvent(ctx, msg, event, runID, &eventSeq)
		}
	}()

	runContext := catalog.WithProjectScope(ctx, projectID)
	finalMessages, err := orchestrator.Run(runContext, allMessages)
	logger.Info("[Manager] Agent execution completed",
		zap.String("message_id", msg.ID),
		zap.Int("final_messages", len(finalMessages)),
		zap.Error(err),
	)
	if err != nil {
		// Check if error is related to tool_call_id mismatch (old session format)
		errStr := err.Error()
		if strings.Contains(errStr, "tool_call_id") && strings.Contains(errStr, "mismatch") {
			logger.Warn("Detected old session format, clearing session",
				zap.String("session_key", sessionKey),
				zap.Error(err))
			// Clear old session and retry
			if delErr := m.sessionMgr.Delete(sessionKey); delErr != nil {
				logger.Error("Failed to clear old session", zap.Error(delErr))
			} else {
				logger.Debug("Cleared old session, retrying with fresh session")
				// Get fresh session through the same scoped authority.
				var fresh *session.Session
				var getErr error
				if projectScopedConversation {
					fresh, topicID, getErr = m.sessionMgr.GetOrCreateProjectConversation(
						projectID,
						topicID,
					)
				} else {
					fresh, getErr = m.sessionMgr.GetOrCreate(sessionKey)
				}
				if getErr != nil {
					logger.Error("Failed to create fresh session", zap.Error(getErr))
					return getErr
				}
				fresh.MergeMetadata(sessionMetadata)
				// Retry with fresh session (no history)
				finalMessages, retryErr := orchestrator.Run(runContext, []AgentMessage{agentMsg})
				if retryErr != nil {
					m.recordHarnessTrace(msg, sessionKey, projectID, topicID, harnessVersion, harnessFiles, runStarted, nil, retryErr, len(history))
					logger.Error("Agent execution failed on retry", zap.Error(retryErr))
					return retryErr
				}
				// Update session with new messages
				m.updateSession(fresh, finalMessages, 0)
				// Publish response
				if len(finalMessages) > 0 {
					lastMsg := finalMessages[len(finalMessages)-1]
					if lastMsg.Role == RoleAssistant {
						m.publishToBus(ctx, msg.Channel, msg.ChatID, lastMsg, msg.ID, msg.Metadata)
					}
				}
				m.recordHarnessTrace(msg, sessionKey, projectID, topicID, harnessVersion, harnessFiles, runStarted, finalMessages, nil, 0)
				return nil
			}
		}
		m.recordHarnessTrace(msg, sessionKey, projectID, topicID, harnessVersion, harnessFiles, runStarted, finalMessages, err, len(history))
		logger.Error("Agent execution failed", zap.Error(err))
		return err
	}

	// 更新会话（只保存新产生的消息）
	m.updateSession(sess, finalMessages, len(history))

	// 发布响应
	if len(finalMessages) > 0 {
		lastMsg := finalMessages[len(finalMessages)-1]
		if lastMsg.Role == RoleAssistant {
			// 发送 final 事件
			content := extractTextContent(lastMsg)
			m.publishChatEvent(ctx, msg.Channel, msg.ChatID, bus.ChatEventStateFinal, content, runID, eventSeq+1, msg.Metadata)
			// 发布到总线
			m.publishToBus(ctx, msg.Channel, msg.ChatID, lastMsg, msg.ID, msg.Metadata)
		}
	}

	m.recordHarnessTrace(msg, sessionKey, projectID, topicID, harnessVersion, harnessFiles, runStarted, finalMessages, nil, len(history))

	return nil
}

func (m *AgentManager) recordHarnessTrace(
	msg *bus.InboundMessage,
	sessionKey, projectID, topicID, harnessVersion string,
	harnessFiles []string,
	started time.Time,
	finalMessages []AgentMessage,
	runErr error,
	historyCount int,
) {
	if m.harnessSvc == nil {
		return
	}
	status := "completed"
	errText := ""
	if runErr != nil {
		status = "failed"
		errText = runErr.Error()
	}
	output := ""
	toolCalls := make([]harness.ToolCallTrace, 0)
	toolCallIndexes := make(map[string]int)
	tokenUsage := harness.TokenUsage{}
	newMessages := finalMessages
	if historyCount > 0 && historyCount < len(finalMessages) {
		newMessages = finalMessages[historyCount:]
	}
	for _, message := range newMessages {
		if message.Role == RoleAssistant {
			if usage, ok := message.Metadata["usage"].(map[string]any); ok {
				tokenUsage.Input += numericInt(usage["input"])
				tokenUsage.Output += numericInt(usage["output"])
				tokenUsage.Total += numericInt(usage["total"])
			}
			if text := extractTextContent(message); text != "" {
				output = text
			}
			for _, block := range message.Content {
				if tool, ok := block.(ToolCallContent); ok {
					toolCallIndexes[tool.ID] = len(toolCalls)
					toolCalls = append(toolCalls, harness.ToolCallTrace{
						ID:     tool.ID,
						Name:   tool.Name,
						Params: tool.Arguments,
						Status: "started",
					})
				}
			}
		}
		if message.Role == RoleToolResult {
			callID, _ := message.Metadata["tool_call_id"].(string)
			toolName, _ := message.Metadata["tool_name"].(string)
			index, found := toolCallIndexes[callID]
			if !found {
				index = len(toolCalls)
				toolCallIndexes[callID] = index
				toolCalls = append(toolCalls, harness.ToolCallTrace{ID: callID, Name: toolName})
			}
			trace := &toolCalls[index]
			trace.Result = extractTextContent(message)
			trace.Status = "completed"
			if errorText, ok := message.Metadata["error"].(string); ok && errorText != "" {
				trace.Error = errorText
				trace.Status = "failed"
			}
			if len(message.Metadata) > 0 {
				trace.Details = make(map[string]any, len(message.Metadata))
				for key, value := range message.Metadata {
					trace.Details[key] = value
				}
			}
		}
	}
	finished := time.Now().UTC()
	metadata := map[string]any{}
	for key, value := range msg.Metadata {
		metadata[key] = value
	}
	trace := harness.Trace{
		ID:               "trace-" + uuid.NewString(),
		HarnessVersion:   harnessVersion,
		ProjectID:        projectID,
		RepositoryID:     traceMetadataString(msg.Metadata, "repository_id"),
		TaskID:           traceMetadataString(msg.Metadata, "task_id"),
		WorkItemID:       traceMetadataString(msg.Metadata, "work_item_id"),
		IssueID:          traceMetadataString(msg.Metadata, "issue_id"),
		RunID:            traceMetadataString(msg.Metadata, "development_run_id"),
		CorrelationID:    traceMetadataString(msg.Metadata, "correlation_id"),
		CommitSHA:        traceMetadataString(msg.Metadata, "commit_sha"),
		PullRequestURL:   traceMetadataString(msg.Metadata, "pull_request_url"),
		PolicyBundleHash: traceMetadataString(msg.Metadata, "policy_bundle_hash"),
		DocumentRefs:     traceMetadataStrings(msg.Metadata, "document_refs"),
		TopicID:          topicID,
		ConversationID:   fmt.Sprintf("%s:%s:%s", msg.Channel, msg.AccountID, msg.ChatID),
		SessionID:        sessionKey,
		MessageID:        msg.ID,
		Channel:          msg.Channel,
		AccountID:        msg.AccountID,
		ActorID:          msg.SenderID,
		Model:            m.cfg.Agents.Defaults.Model.Effective(),
		Status:           status,
		Input:            msg.Content,
		Output:           output,
		Error:            errText,
		ToolCalls:        toolCalls,
		Context: harness.ContextManifest{
			LoadedFiles:    append([]string(nil), harnessFiles...),
			MemoryIDs:      catalogMemoryIDs(finalMessages),
			RecentMessages: historyCount,
		},
		Metadata:   metadata,
		StartedAt:  started,
		FinishedAt: finished,
		DurationMS: finished.Sub(started).Milliseconds(),
		TokenUsage: tokenUsage,
	}
	if err := m.harnessSvc.RecordTrace(trace); err != nil {
		logger.Warn("Failed to persist harness trace", zap.Error(err))
	}
}

func traceMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func traceMetadataStrings(metadata map[string]interface{}, key string) []string {
	if metadata == nil {
		return nil
	}
	var result []string
	switch values := metadata[key].(type) {
	case []string:
		result = append(result, values...)
	case []interface{}:
		for _, item := range values {
			if value, ok := item.(string); ok {
				result = append(result, value)
			}
		}
	}
	seen := make(map[string]struct{}, len(result))
	filtered := result[:0]
	for _, value := range result {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		filtered = append(filtered, value)
	}
	return filtered
}

func catalogMemoryIDs(messages []AgentMessage) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, message := range messages {
		if message.Metadata == nil {
			continue
		}
		switch values := message.Metadata["catalog_memory_ids"].(type) {
		case []string:
			for _, value := range values {
				if _, exists := seen[value]; value != "" && !exists {
					seen[value] = struct{}{}
					result = append(result, value)
				}
			}
		case []interface{}:
			for _, item := range values {
				value, _ := item.(string)
				if _, exists := seen[value]; value != "" && !exists {
					seen[value] = struct{}{}
					result = append(result, value)
				}
			}
		}
	}
	return result
}

func numericInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		parsed, _ := typed.Int64()
		return int(parsed)
	default:
		return 0
	}
}

func (m *AgentManager) handleDirectCronOneShot(ctx context.Context, msg *bus.InboundMessage) (bool, error) {
	if msg == nil || m.tools == nil {
		return false, nil
	}

	content := strings.TrimSpace(msg.Content)
	if !isCronOneShotRequest(content) {
		return false, nil
	}

	jobID, err := m.resolveCronJobIDForOneShot(ctx, content)
	if err != nil {
		errMsg := AgentMessage{
			Role:      RoleAssistant,
			Content:   []ContentBlock{TextContent{Text: "已识别为一次性测试请求，但未找到可执行任务：" + err.Error()}},
			Timestamp: time.Now().UnixMilli(),
		}
		m.publishToBus(ctx, msg.Channel, msg.ChatID, errMsg, msg.ID)
		return true, nil
	}
	if ok, wait := m.allowManualCronRun(jobID, time.Now()); !ok {
		errMsg := AgentMessage{
			Role:      RoleAssistant,
			Content:   []ContentBlock{TextContent{Text: fmt.Sprintf("任务 `%s` 刚刚手工触发过，请 %d 秒后再试。", jobID, wait)}},
			Timestamp: time.Now().UnixMilli(),
		}
		m.publishToBus(ctx, msg.Channel, msg.ChatID, errMsg, msg.ID)
		return true, nil
	}

	ack := AgentMessage{
		Role:      RoleAssistant,
		Content:   []ContentBlock{TextContent{Text: fmt.Sprintf("收到，开始手工执行一次任务 `%s`。", jobID)}},
		Timestamp: time.Now().UnixMilli(),
	}
	m.publishToBus(ctx, msg.Channel, msg.ChatID, ack, msg.ID)

	go func(channel, chatID, replyTo, id string) {
		runCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_, runErr := m.tools.Execute(runCtx, "cron", map[string]interface{}{
			"command": fmt.Sprintf("run %s", id),
		})

		text := fmt.Sprintf("已手工执行一次任务 `%s`。", id)
		if runErr != nil {
			text = fmt.Sprintf("手工执行任务 `%s` 失败：%v", id, runErr)
		}

		done := AgentMessage{
			Role:      RoleAssistant,
			Content:   []ContentBlock{TextContent{Text: text}},
			Timestamp: time.Now().UnixMilli(),
		}
		m.publishToBus(context.Background(), channel, chatID, done, replyTo)
	}(msg.Channel, msg.ChatID, msg.ID, jobID)

	return true, nil
}

func (m *AgentManager) allowManualCronRun(jobID string, now time.Time) (bool, int) {
	const cooldown = 60 * time.Second
	m.manualCronMu.Lock()
	defer m.manualCronMu.Unlock()

	if last, ok := m.manualCronLast[jobID]; ok {
		if delta := now.Sub(last); delta < cooldown {
			wait := int((cooldown - delta).Round(time.Second).Seconds())
			if wait < 1 {
				wait = 1
			}
			return false, wait
		}
	}
	m.manualCronLast[jobID] = now
	return true, 0
}

func isCronOneShotRequest(text string) bool {
	if text == "" {
		return false
	}
	normalized := strings.ToLower(strings.TrimSpace(text))
	if strings.Contains(normalized, "cron run") {
		return true
	}
	keywords := []string{
		"执行一次定时任务",
		"只测试一次定时任务",
		"手工执行一次定时任务",
		"临时执行一次定时任务",
		"测试一次定时任务",
	}
	for _, kw := range keywords {
		if strings.Contains(normalized, kw) {
			return true
		}
	}
	return false
}

func (m *AgentManager) resolveCronJobIDForOneShot(ctx context.Context, text string) (string, error) {
	if id := cronJobIDPattern.FindString(text); id != "" {
		return id, nil
	}

	listOut, err := m.tools.Execute(ctx, "cron", map[string]interface{}{"command": "list"})
	if err != nil {
		return "", fmt.Errorf("获取任务列表失败: %w", err)
	}

	enabledIDs := extractEnabledCronJobIDs(listOut)
	switch len(enabledIDs) {
	case 0:
		return "", fmt.Errorf("没有启用中的任务")
	case 1:
		return enabledIDs[0], nil
	default:
		return "", fmt.Errorf("存在多个启用任务，请在消息中指定 job-id")
	}
}

func extractEnabledCronJobIDs(listOutput string) []string {
	lines := strings.Split(listOutput, "\n")
	ids := make([]string, 0)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		matches := cronListLinePattern.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue
		}
		if matches[2] == "enabled" {
			ids = append(ids, matches[1])
		}
	}
	return ids
}

// updateSession 更新会话
func (m *AgentManager) updateSession(sess *session.Session, messages []AgentMessage, historyLen int) {
	// 只保存新产生的消息（不包括历史消息）
	newMessages := messages
	if historyLen >= 0 && len(messages) > historyLen {
		newMessages = messages[historyLen:]
	}

	_ = m.helper.UpdateSession(sess, newMessages, &UpdateSessionOptions{SaveImmediately: true})
}

func conversationSessionKey(
	msg *bus.InboundMessage,
	projectID, topicID string,
	projectScoped bool,
) string {
	if projectScoped {
		// Project and topic are the canonical conversation boundary. This makes
		// Obsidian, Feishu, and future channels share one project history.
		key, _, err := session.ProjectConversationKey(projectID, topicID)
		if err != nil {
			return ""
		}
		return key
	}
	if msg.ChatID == "default" || msg.ChatID == "" {
		return fmt.Sprintf(
			"%s:%s:%d",
			msg.Channel,
			msg.AccountID,
			msg.Timestamp.Unix(),
		)
	}
	return fmt.Sprintf("%s:%s:%s", msg.Channel, msg.AccountID, msg.ChatID)
}

func cloneMessageMetadata(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]interface{}, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

// publishToBus 发布消息到总线
func (m *AgentManager) publishToBus(
	ctx context.Context,
	channel, chatID string,
	msg AgentMessage,
	replyTo string,
	metadata ...map[string]interface{},
) {
	content := extractTextContent(msg)

	outbound := &bus.OutboundMessage{
		Channel:   channel,
		ChatID:    chatID,
		Content:   content,
		ReplyTo:   replyTo,
		Timestamp: time.Unix(msg.Timestamp/1000, 0),
	}
	if len(metadata) > 0 {
		outbound.Metadata = cloneMessageMetadata(metadata[0])
	}

	if err := m.bus.PublishOutbound(ctx, outbound); err != nil {
		logger.Error("Failed to publish outbound", zap.Error(err))
	}
}

// publishChatEvent 发布聊天事件到总线
func (m *AgentManager) publishChatEvent(
	ctx context.Context,
	channel, chatID, state, content, runID string,
	seq int,
	metadata ...map[string]interface{},
) {
	event := &bus.ChatEvent{
		Channel:   channel,
		ChatID:    chatID,
		State:     state,
		Content:   content,
		RunID:     runID,
		Seq:       seq,
		Timestamp: time.Now(),
	}
	if len(metadata) > 0 {
		event.Metadata = cloneMessageMetadata(metadata[0])
	}

	if err := m.bus.PublishChatEvent(ctx, event); err != nil {
		logger.Error("Failed to publish chat event", zap.Error(err))
	}
}

// handleOrchestratorEvent 处理 orchestrator 事件
func (m *AgentManager) handleOrchestratorEvent(ctx context.Context, msg *bus.InboundMessage, event *Event, runID string, seq *int) {
	if event == nil {
		return
	}

	*seq++

	switch event.Type {
	case EventStreamContent:
		// 流式内容事件
		m.publishChatEvent(ctx, msg.Channel, msg.ChatID, bus.ChatEventStateDelta, event.StreamContent, runID, *seq, msg.Metadata)
	case EventStreamThinking:
		// 思考过程事件
		m.publishChatEvent(ctx, msg.Channel, msg.ChatID, bus.ChatEventStateThinking, event.StreamContent, runID, *seq, msg.Metadata)
	case EventStreamFinal:
		// 最终内容事件
		m.publishChatEvent(ctx, msg.Channel, msg.ChatID, bus.ChatEventStateDelta, event.StreamContent, runID, *seq, msg.Metadata)
	case EventStreamDone:
		// 流结束
		logger.Debug("Stream done event", zap.String("run_id", runID))
	case EventToolExecutionStart:
		// 工具开始事件
		toolInfo := map[string]interface{}{
			"tool_name": event.ToolName,
			"tool_id":   event.ToolID,
		}
		toolJSON, _ := json.Marshal(toolInfo)
		m.publishChatEvent(ctx, msg.Channel, msg.ChatID, bus.ChatEventStateTool, string(toolJSON), runID, *seq, msg.Metadata)
	case EventToolExecutionEnd:
		// 工具结束事件
		toolInfo := map[string]interface{}{
			"tool_name":   event.ToolName,
			"tool_id":     event.ToolID,
			"tool_result": event.ToolResult,
		}
		toolJSON, _ := json.Marshal(toolInfo)
		m.publishChatEvent(ctx, msg.Channel, msg.ChatID, bus.ChatEventStateTool, string(toolJSON), runID, *seq, msg.Metadata)
	case EventTurnEnd, EventAgentEnd:
		// Turn/Agent 结束事件
		logger.Debug("Agent turn end", zap.String("run_id", runID), zap.String("event_type", string(event.Type)))
	}
}

// sessionMessagesToAgentMessages 将 session 消息转换为 Agent 消息
func sessionMessagesToAgentMessages(sessMsgs []session.Message) []AgentMessage {
	result := make([]AgentMessage, 0, len(sessMsgs))
	for _, sessMsg := range sessMsgs {
		agentMsg := AgentMessage{
			Role:      MessageRole(sessMsg.Role),
			Content:   []ContentBlock{TextContent{Text: sessMsg.Content}},
			Timestamp: sessMsg.Timestamp.UnixMilli(),
		}

		// Handle tool calls in assistant messages
		if sessMsg.Role == "assistant" && len(sessMsg.ToolCalls) > 0 {
			// Clear the text content if there are tool calls
			agentMsg.Content = []ContentBlock{}
			for _, tc := range sessMsg.ToolCalls {
				agentMsg.Content = append(agentMsg.Content, ToolCallContent{
					ID:        tc.ID,
					Name:      tc.Name,
					Arguments: tc.Params,
				})
			}
		}

		// Handle tool result messages
		if sessMsg.Role == "tool" {
			agentMsg.Role = RoleToolResult
			// Set tool_call_id in metadata
			if sessMsg.ToolCallID != "" {
				if agentMsg.Metadata == nil {
					agentMsg.Metadata = make(map[string]any)
				}
				agentMsg.Metadata["tool_call_id"] = sessMsg.ToolCallID
			}
			// Restore tool_name from metadata if exists
			if toolName, ok := sessMsg.Metadata["tool_name"].(string); ok {
				if agentMsg.Metadata == nil {
					agentMsg.Metadata = make(map[string]any)
				}
				agentMsg.Metadata["tool_name"] = toolName
			}
		}

		result = append(result, agentMsg)
	}
	return result
}

// GetAgent 获取 Agent
func (m *AgentManager) GetAgent(agentID string) (*Agent, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, ok := m.agents[agentID]
	return agent, ok
}

// ListAgents 列出所有 Agent ID
func (m *AgentManager) ListAgents() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.agents))
	for id := range m.agents {
		ids = append(ids, id)
	}
	return ids
}

// Start 启动所有 Agent
func (m *AgentManager) Start(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id := range m.agents {
		logger.Info("Agent registered under manager (inbound loop handled by AgentManager)",
			zap.String("agent_id", id))
	}

	// 启动消息处理器
	go m.processMessages(ctx)

	return nil
}

// Stop 停止所有 Agent
func (m *AgentManager) Stop() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for id, agent := range m.agents {
		if err := agent.Stop(); err != nil {
			logger.Error("Failed to stop agent",
				zap.String("agent_id", id),
				zap.Error(err))
		}
	}

	return nil
}

// processMessages 处理入站消息
func (m *AgentManager) processMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			logger.Info("Agent manager message processor stopped")
			return
		default:
			msg, err := m.bus.ConsumeInbound(ctx)
			if err != nil {
				if err == context.DeadlineExceeded || err == context.Canceled {
					continue
				}
				logger.Error("Failed to consume inbound", zap.Error(err))
				continue
			}

			logger.Debug("[Manager] Consumed inbound message from bus",
				zap.String("message_id", msg.ID),
				zap.String("channel", msg.Channel),
				zap.String("chat_id", msg.ChatID),
			)
			if err := m.RouteInbound(ctx, msg); err != nil {
				logger.Error("Failed to route message",
					zap.String("channel", msg.Channel),
					zap.String("account_id", msg.AccountID),
					zap.Error(err))
			}
		}
	}
}

// GetDefaultAgent 获取默认 Agent
func (m *AgentManager) GetDefaultAgent() *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultAgent
}

// GetToolsInfo 获取工具信息
func (m *AgentManager) GetToolsInfo() (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 从 tool registry 获取工具列表
	existingTools := m.tools.ListExisting()
	result := make(map[string]interface{})

	for _, tool := range existingTools {
		result[tool.Name()] = map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
		}
	}

	return result, nil
}

// convertRetryConfig converts config.RetryConfig to agent.RetryConfig
func convertRetryConfig(cfg *config.RetryConfig) *RetryConfig {
	if cfg == nil {
		return nil
	}
	return &RetryConfig{
		Enabled:               cfg.Enabled,
		MaxRetries:            cfg.MaxRetries,
		InitialDelay:          cfg.InitialDelay,
		MaxDelay:              cfg.MaxDelay,
		BackoffFactor:         cfg.BackoffFactor,
		RetryableErrors:       cfg.RetryableErrors,
		ContextOverflowAction: cfg.ContextOverflowAction,
	}
}

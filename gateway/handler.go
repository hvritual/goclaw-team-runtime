package gateway

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/smallnest/goclaw/bus"
	"github.com/smallnest/goclaw/channels"
	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/cron"
	"github.com/smallnest/goclaw/governance"
	"github.com/smallnest/goclaw/harness"
	"github.com/smallnest/goclaw/internal/logger"
	"github.com/smallnest/goclaw/memory/catalog"
	"github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/ouroboros"
	"github.com/smallnest/goclaw/session"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
	"go.uber.org/zap"
)

// Handler WebSocket 消息处理器
type Handler struct {
	registry   *MethodRegistry
	bus        *bus.MessageBus
	sessionMgr *session.Manager
	channelMgr *channels.Manager
	cronSvc    *cron.Service
	cfg        *config.Config
	harnessSvc *harness.Service
	catalogSvc *catalog.Service
	devSvc     *orchestratorlite.Service
	ouroSvc    *ouroboros.Service
	teamSvc    *teamcontrol.Service
	runnerSvc  *workstation.Service
	devLocks   sync.Map
}

func (h *Handler) lockDevelopmentTask(id string) func() {
	value, _ := h.devLocks.LoadOrStore(id, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

// SetDevelopmentService enables the approval-gated Orchestrator Lite control
// surface. Long-running execution remains opt-in through
// development.gateway_allow_execution.
func (h *Handler) SetDevelopmentService(service *orchestratorlite.Service) {
	h.devSvc = service
	if service != nil {
		h.registerDevelopmentMethods()
	}
}

// SetOuroborosService enables the specification-first JSON-RPC surface.
// Model operations remain read-only; Seed and evolution approvals are explicit
// methods and task execution is still owned by Orchestrator Lite.
func (h *Handler) SetOuroborosService(service *ouroboros.Service) {
	h.ouroSvc = service
	if service != nil {
		h.registerOuroborosMethods()
	}
}

// SetHarnessService enables the Better-Harness JSON-RPC surface. It is kept as
// an optional extension so existing gateway construction and tests remain
// backward compatible.
func (h *Handler) SetHarnessService(service *harness.Service) {
	h.harnessSvc = service
	if service != nil {
		h.registerHarnessMethods()
	}
}

func (h *Handler) SetMemoryCatalog(service *catalog.Service) {
	h.catalogSvc = service
	if service != nil {
		h.registerMemoryCatalogMethods()
	}
}

// SetTeamControlService enables authenticated, project-scoped team methods.
// The Server is responsible for authenticating the personal access token and
// passing a user-bound session to HandleRequest.
func (h *Handler) SetTeamControlService(service *teamcontrol.Service) {
	h.teamSvc = service
	if service != nil {
		h.registerTeamControlMethods()
	}
}

// SetWorkstationService enables the durable workstation runner queue methods.
func (h *Handler) SetWorkstationService(service *workstation.Service) {
	h.runnerSvc = service
	if service != nil {
		h.registerWorkstationMethods()
	}
}

// NewHandler 创建处理器
func NewHandler(messageBus *bus.MessageBus, sessionMgr *session.Manager, channelMgr *channels.Manager, cronSvc *cron.Service, cfg *config.Config) *Handler {
	h := &Handler{
		registry:   NewMethodRegistry(),
		bus:        messageBus,
		sessionMgr: sessionMgr,
		channelMgr: channelMgr,
		cronSvc:    cronSvc,
		cfg:        cfg,
	}

	// 注册系统方法
	h.registerSystemMethods()

	// 注册 Agent 方法
	h.registerAgentMethods()

	// 注册 Channel 方法
	h.registerChannelMethods()

	// 注册 Browser 方法
	h.registerBrowserMethods()

	// 注册 Cron 方法
	h.registerCronMethods()

	// Cross-store checks are registered even before optional services are
	// attached; the method fails closed until all three pilot services exist.
	h.registerControlConsistencyMethods()

	return h
}

// HandleRequest 处理请求
func (h *Handler) HandleRequest(sessionID string, req *JSONRPCRequest) *JSONRPCResponse {
	if err := h.authorizeTeamMethod(sessionID, req.Method, req.Params); err != nil {
		logger.Warn("Team method authorization failed",
			zap.String("method", req.Method),
			zap.String("session_id", sessionID),
			zap.Error(err))
		return NewErrorResponse(req.ID, ErrorInvalidRequest, err.Error())
	}
	result, err := h.registry.Call(req.Method, sessionID, req.Params)
	if err != nil {
		logger.Error("Method execution failed",
			zap.String("method", req.Method),
			zap.String("session_id", sessionID),
			zap.Error(err))
		return NewErrorResponse(req.ID, ErrorInternalError, err.Error())
	}

	return NewSuccessResponse(req.ID, result)
}

// registerSystemMethods 注册系统方法
func (h *Handler) registerSystemMethods() {
	// config.get - 获取配置
	h.registry.Register("config.get", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		key, ok := params["key"].(string)
		if !ok {
			return nil, fmt.Errorf("key parameter is required")
		}
		// 这里应该从配置中读取
		// 简化实现：返回模拟数据
		return map[string]interface{}{
			"key":   key,
			"value": "config_value",
		}, nil
	})

	// config.set - 设置配置
	h.registry.Register("config.set", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		key, _ := params["key"].(string)
		value := params["value"]
		// 这里应该更新配置
		return map[string]interface{}{
			"key":   key,
			"value": value,
		}, nil
	})

	// health - 健康检查
	h.registry.Register("health", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
			"version":   ProtocolVersion,
		}, nil
	})

	// logs - 获取日志
	h.registry.Register("logs.get", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		lines := 100
		if l, ok := params["lines"].(float64); ok {
			lines = int(l)
		}
		// 这里应该从日志中读取
		return map[string]interface{}{
			"lines": lines,
			"logs":  []string{}, // 实际应该返回日志
		}, nil
	})
}

// registerAgentMethods 注册 Agent 方法
func (h *Handler) registerAgentMethods() {
	h.registry.Register("chat.history", h.rpcChatHistory)

	// chat - 直接与 gateway 对话
	h.registry.Register("chat", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("content parameter is required")
		}
		senderID, chatID, metadata, err := h.chatIdentity(sessionID, params)
		if err != nil {
			return nil, err
		}

		// 构造入站消息
		msg := &bus.InboundMessage{
			Channel:   "gateway",
			SenderID:  senderID,
			ChatID:    chatID,
			Content:   content,
			Metadata:  metadata,
			Timestamp: time.Now(),
		}

		// 发布到消息总线
		if err := h.bus.PublishInbound(context.Background(), msg); err != nil {
			return nil, fmt.Errorf("failed to publish message: %w", err)
		}

		return map[string]interface{}{
			"status":  "queued",
			"msg_id":  msg.ID,
			"session": chatID,
		}, nil
	})

	// agent - 发送消息给 Agent
	h.registry.Register("agent", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("content parameter is required")
		}
		senderID, chatID, metadata, err := h.chatIdentity(sessionID, params)
		if err != nil {
			return nil, err
		}

		// 构造入站消息
		msg := &bus.InboundMessage{
			Channel:   "gateway",
			SenderID:  senderID,
			ChatID:    chatID,
			Content:   content,
			Metadata:  metadata,
			Timestamp: time.Now(),
		}

		// 发布到消息总线
		if err := h.bus.PublishInbound(context.Background(), msg); err != nil {
			return nil, fmt.Errorf("failed to publish message: %w", err)
		}

		return map[string]interface{}{
			"status": "queued",
			"msg_id": msg.ID,
		}, nil
	})

	// agent.wait - 发送消息并等待响应
	h.registry.Register("agent.wait", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("content parameter is required")
		}
		senderID, chatID, metadata, err := h.chatIdentity(sessionID, params)
		if err != nil {
			return nil, err
		}

		timeout := 30 * time.Second
		if t, ok := params["timeout"].(float64); ok {
			timeout = time.Duration(t) * time.Second
		}

		// 构造入站消息
		msg := &bus.InboundMessage{
			Channel:   "gateway",
			SenderID:  senderID,
			ChatID:    chatID,
			Content:   content,
			Metadata:  metadata,
			Timestamp: time.Now(),
		}

		// 发布到消息总线
		if err := h.bus.PublishInbound(context.Background(), msg); err != nil {
			return nil, fmt.Errorf("failed to publish message: %w", err)
		}

		// 等待响应（简化实现）
		// 实际应该通过监听出站消息来获取响应
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for response")
		default:
			// 返回初始响应
			return map[string]interface{}{
				"status":  "waiting",
				"msg_id":  msg.ID,
				"timeout": timeout.String(),
			}, nil
		}
	})

	// sessions.list - 列出所有会话
	h.registry.Register("sessions.list", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		sessions, err := h.sessionMgr.List()
		if err != nil {
			return nil, fmt.Errorf("failed to list sessions: %w", err)
		}

		var updateTimes []int64
		var sessionMap = map[int64][]*session.Session{}
		for _, key := range sessions {
			sess, err := h.sessionMgr.GetOrCreate(key)
			if err != nil {
				continue
			}
			upt := sess.UpdatedAt.Unix()
			if v, ok := sessionMap[upt]; ok {
				sessionMap[upt] = append(v, sess)
			} else {
				updateTimes = append(updateTimes, upt)
				sessionMap[upt] = []*session.Session{sess}
			}
		}
		slices.SortFunc(updateTimes, func(e int64, e2 int64) int {
			return cmp.Compare(e2, e)
		})
		result := make([]map[string]interface{}, 0, len(sessions))
		for _, updateTime := range updateTimes {
			ss := sessionMap[updateTime]
			for _, sess := range ss {
				result = append(result, map[string]interface{}{
					"key":           sess.Key,
					"message_count": len(sess.Messages),
					"created_at":    sess.CreatedAt,
					"updated_at":    sess.UpdatedAt,
				})
			}
		}
		return result, nil
	})

	// sessions.get - 获取会话详情
	h.registry.Register("sessions.get", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		key, ok := params["key"].(string)
		if !ok {
			return nil, fmt.Errorf("key parameter is required")
		}

		sess, err := h.sessionMgr.GetOrCreate(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}

		return map[string]interface{}{
			"key":        sess.Key,
			"messages":   sess.Messages,
			"created_at": sess.CreatedAt,
			"updated_at": sess.UpdatedAt,
			"metadata":   sess.Metadata,
		}, nil
	})

	// sessions.clear - 清空会话
	h.registry.Register("sessions.clear", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		key, ok := params["key"].(string)
		if !ok {
			return nil, fmt.Errorf("key parameter is required")
		}

		sess, err := h.sessionMgr.GetOrCreate(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}

		sess.Clear()

		return map[string]interface{}{
			"status": "cleared",
			"key":    key,
		}, nil
	})
}

// registerChannelMethods 注册 Channel 方法
func (h *Handler) registerChannelMethods() {
	// channels.status - 获取通道状态
	h.registry.Register("channels.status", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		name, ok := params["channel"].(string)
		if !ok {
			return nil, fmt.Errorf("channel parameter is required")
		}

		status, err := h.channelMgr.Status(name)
		if err != nil {
			return nil, fmt.Errorf("failed to get channel status: %w", err)
		}

		return status, nil
	})

	// channels.list - 列出所有通道
	h.registry.Register("channels.list", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		channels := h.channelMgr.List()
		return map[string]interface{}{
			"channels": channels,
		}, nil
	})

	// send - 发送消息到通道
	h.registry.Register("send", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		channel, ok := params["channel"].(string)
		if !ok {
			return nil, fmt.Errorf("channel parameter is required")
		}

		chatID, ok := params["chat_id"].(string)
		if !ok {
			return nil, fmt.Errorf("chat_id parameter is required")
		}

		content, ok := params["content"].(string)
		if !ok {
			return nil, fmt.Errorf("content parameter is required")
		}

		msg := &bus.OutboundMessage{
			Channel:   channel,
			ChatID:    chatID,
			Content:   content,
			Timestamp: time.Now(),
		}

		if err := h.bus.PublishOutbound(context.Background(), msg); err != nil {
			return nil, fmt.Errorf("failed to send message: %w", err)
		}

		return map[string]interface{}{
			"status":  "sent",
			"msg_id":  msg.ID,
			"channel": channel,
			"chat_id": chatID,
		}, nil
	})

	// chat - 发送聊天消息（简化版）
	h.registry.Register("chat.send", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		// 与 send 相同，但可以添加更多聊天相关功能
		return h.registry.Call("send", sessionID, params)
	})
}

func (h *Handler) registerHarnessMethods() {
	h.registry.Register("harness.status", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		active, err := h.harnessSvc.ActiveState()
		if err != nil {
			return nil, err
		}
		manifest, err := h.harnessSvc.ActiveManifest()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"active":   active,
			"manifest": manifest,
			"project":  h.harnessSvc.ProjectID(),
		}, nil
	})

	h.registry.Register("harness.versions", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		return h.harnessSvc.ListVersions()
	})

	h.registry.Register("harness.evals", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		return h.harnessSvc.ListEvalCases()
	})

	h.registry.Register("harness.traces", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		projectID, _ := params["project_id"].(string)
		limit := 100
		if value, ok := params["limit"].(float64); ok {
			limit = int(value)
		}
		return h.harnessSvc.ListTraces(projectID, limit)
	})

	h.registry.Register("harness.feedback", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		traceID, _ := params["trace_id"].(string)
		rating, _ := params["rating"].(string)
		comment, _ := params["comment"].(string)
		if traceID == "" || rating == "" {
			return nil, fmt.Errorf("trace_id and rating are required")
		}
		err := h.harnessSvc.AddHumanFeedback(traceID, harness.HumanFeedback{
			Rating:    rating,
			Comment:   comment,
			CreatedAt: time.Now().UTC(),
		})
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"status": "recorded", "trace_id": traceID}, nil
	})

	h.registry.Register("harness.experiments", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		return h.harnessSvc.ListExperiments()
	})

	h.registry.Register("harness.experiment.get", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id, _ := params["id"].(string)
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		return h.harnessSvc.GetExperiment(id)
	})

	h.registry.Register("harness.experiment.create", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		baseVersion, _ := params["base_version"].(string)
		candidateVersion, _ := params["candidate_version"].(string)
		change := harness.ChangeManifest{
			TargetComponents:    stringSliceParam(params["target_components"]),
			EvidenceTraceIDs:    stringSliceParam(params["evidence_trace_ids"]),
			RootCause:           stringParam(params["root_cause"]),
			ChangeSummary:       stringParam(params["change_summary"]),
			ExpectedFixTags:     stringSliceParam(params["expected_fix_tags"]),
			PossibleRegressions: stringSliceParam(params["possible_regressions"]),
		}
		if change.RootCause == "" || change.ChangeSummary == "" {
			return nil, fmt.Errorf("root_cause and change_summary are required")
		}
		return h.harnessSvc.CreateExperiment(baseVersion, candidateVersion, change)
	})

	h.registry.Register("harness.experiment.validate", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id, _ := params["id"].(string)
		execute, _ := params["execute"].(bool)
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		if execute {
			return nil, fmt.Errorf("command-backed Harness evals are disabled through Gateway; run them from the local CLI")
		}
		return h.harnessSvc.ValidateExperiment(context.Background(), id, execute)
	})

	h.registry.Register("harness.experiment.approve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id, _ := params["id"].(string)
		review, err := h.humanReview(sessionID, params, governance.RoleHarnessApprove)
		if err != nil {
			return nil, err
		}
		return h.harnessSvc.ApproveExperimentWithReview(id, review)
	})

	h.registry.Register("harness.experiment.reject", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id, _ := params["id"].(string)
		review, err := h.humanReview(sessionID, params, governance.RoleHarnessApprove)
		if err != nil {
			return nil, err
		}
		return h.harnessSvc.RejectExperimentWithReview(id, review)
	})

	h.registry.Register("harness.experiment.promote", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id, _ := params["id"].(string)
		review, err := h.humanReview(sessionID, params, governance.RoleHarnessPromote)
		if err != nil {
			return nil, err
		}
		return h.harnessSvc.PromoteExperimentWithReview(id, review)
	})

	h.registry.Register("harness.rollback", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		review, err := h.humanReview(sessionID, params, governance.RoleHarnessRollback)
		if err != nil {
			return nil, err
		}
		return h.harnessSvc.RollbackWithReview(review)
	})

	h.registry.Register("knowledge.proposals", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		status := harness.KnowledgeProposalStatus(stringParam(params["status"]))
		return h.harnessSvc.ListKnowledgeProposals(status)
	})

	h.registry.Register("knowledge.proposal.get", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		if id == "" {
			return nil, fmt.Errorf("id is required")
		}
		return h.harnessSvc.GetKnowledgeProposal(id)
	})

	h.registry.Register("knowledge.proposal.create", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		target := stringParam(params["target_path"])
		content := stringParam(params["proposed_content"])
		reason := stringParam(params["reason"])
		if target == "" || content == "" || reason == "" {
			return nil, fmt.Errorf("target_path, proposed_content, and reason are required")
		}
		actor, err := h.actorForSession(sessionID, stringParam(params["actor"]))
		if err != nil {
			return nil, err
		}
		return h.harnessSvc.CreateKnowledgeProposal(
			target,
			content,
			reason,
			stringParam(params["evidence_trace_id"]),
			actor,
		)
	})

	h.registry.Register("knowledge.proposal.approve", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		review, err := h.humanReview(sessionID, params, governance.RoleKnowledgeApprove)
		if err != nil {
			return nil, err
		}
		return h.harnessSvc.ApproveKnowledgeProposalWithReview(id, review)
	})

	h.registry.Register("knowledge.proposal.reject", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		id := stringParam(params["id"])
		review, err := h.humanReview(sessionID, params, governance.RoleKnowledgeApprove)
		if err != nil {
			return nil, err
		}
		return h.harnessSvc.RejectKnowledgeProposalWithReview(id, review)
	})
}

func stringParam(value interface{}) string {
	result, _ := value.(string)
	return result
}

func stringSliceParam(value interface{}) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				result = append(result, value)
			}
		}
		return result
	default:
		return nil
	}
}

func projectMetadata(params map[string]interface{}) map[string]interface{} {
	metadata := make(map[string]interface{})
	if projectID := stringParam(params["project_id"]); projectID != "" {
		metadata["project_id"] = projectID
	}
	if topicID := stringParam(params["topic_id"]); topicID != "" {
		metadata["topic_id"] = topicID
	}
	return metadata
}

// registerBrowserMethods 注册 Browser 方法
func (h *Handler) registerBrowserMethods() {
	// browser.request - 浏览器请求
	h.registry.Register("browser.request", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		action, ok := params["action"].(string)
		if !ok {
			return nil, fmt.Errorf("action parameter is required")
		}

		// 这里应该调用浏览器工具
		// 简化实现：返回模拟响应
		return map[string]interface{}{
			"status": "executed",
			"action": action,
			"result": "browser action executed",
		}, nil
	})
}

// BroadcastNotification 广播通知
func (h *Handler) BroadcastNotification(method string, data interface{}) ([]byte, error) {
	notif := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params: map[string]interface{}{
			"data": data,
		},
	}

	return json.Marshal(notif)
}

// registerCronMethods 注册 Cron 方法
func (h *Handler) registerCronMethods() {
	// cron.status - 获取 cron 服务状态
	h.registry.Register("cron.status", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.cronSvc == nil {
			return nil, fmt.Errorf("cron service is not available")
		}

		return h.cronSvc.GetStatus(), nil
	})

	// cron.list - 列出所有 cron 任务
	h.registry.Register("cron.list", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.cronSvc == nil {
			return nil, fmt.Errorf("cron service is not available")
		}

		includeDisabled := false
		if v, ok := params["include_disabled"].(bool); ok {
			includeDisabled = v
		}

		jobs := h.cronSvc.ListJobs()

		// Filter by enabled status if needed
		var filteredJobs []*cron.Job
		if includeDisabled {
			filteredJobs = jobs
		} else {
			filteredJobs = make([]*cron.Job, 0)
			for _, job := range jobs {
				if job.State.Enabled {
					filteredJobs = append(filteredJobs, job)
				}
			}
		}

		return map[string]interface{}{
			"jobs":  filteredJobs,
			"count": len(filteredJobs),
		}, nil
	})

	// cron.add - 添加 cron 任务
	h.registry.Register("cron.add", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.cronSvc == nil {
			return nil, fmt.Errorf("cron service is not available")
		}

		// Parse job parameters
		name, ok := params["name"].(string)
		if !ok {
			return nil, fmt.Errorf("name parameter is required")
		}

		job := &cron.Job{
			Name:      name,
			State:     cron.JobState{Enabled: true},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Set default session target to main
		job.SessionTarget = cron.SessionTargetMain

		// Parse session target
		if st, ok := params["session_target"].(string); ok {
			job.SessionTarget = cron.SessionTarget(st)
		}

		// Parse schedule
		if s, ok := params["schedule"].(map[string]interface{}); ok {
			if typ, ok := s["type"].(string); ok {
				job.Schedule.Type = cron.ScheduleType(typ)
				if at, ok := s["at"].(string); ok {
					t, err := time.Parse(time.RFC3339, at)
					if err != nil {
						return nil, fmt.Errorf("invalid at time: %w", err)
					}
					job.Schedule.At = t
				}
				if every, ok := s["every"].(string); ok {
					dur, err := cron.ParseHumanDuration(every)
					if err != nil {
						return nil, fmt.Errorf("invalid every duration: %w", err)
					}
					job.Schedule.EveryDuration = dur
				}
				if expr, ok := s["cron"].(string); ok {
					job.Schedule.CronExpression = expr
				}
			}
		}

		// Parse payload
		if p, ok := params["payload"].(map[string]interface{}); ok {
			if typ, ok := p["type"].(string); ok {
				job.Payload.Type = cron.PayloadType(typ)
				if msg, ok := p["message"].(string); ok {
					job.Payload.Message = msg
				}
				if evt, ok := p["system_event_type"].(string); ok {
					job.Payload.SystemEventType = evt
				}
			}
		}

		// Parse wake mode
		if wm, ok := params["wake_mode"].(string); ok {
			job.WakeMode = cron.WakeMode(wm)
		}

		// Parse delivery
		if d, ok := params["delivery"].(map[string]interface{}); ok {
			job.Delivery = &cron.Delivery{}
			if mode, ok := d["mode"].(string); ok {
				job.Delivery.Mode = cron.DeliveryMode(mode)
			}
			if url, ok := d["webhook_url"].(string); ok {
				job.Delivery.WebhookURL = url
			}
			if token, ok := d["webhook_token"].(string); ok {
				job.Delivery.WebhookToken = token
			}
			if bestEffort, ok := d["best_effort"].(bool); ok {
				job.Delivery.BestEffort = bestEffort
			}
		}

		// Add job (ID will be auto-generated if empty)
		if err := h.cronSvc.AddJob(job); err != nil {
			return nil, fmt.Errorf("failed to add job: %w", err)
		}

		return job, nil
	})

	// cron.update - 更新 cron 任务
	h.registry.Register("cron.update", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.cronSvc == nil {
			return nil, fmt.Errorf("cron service is not available")
		}

		id, ok := params["id"].(string)
		if !ok {
			return nil, fmt.Errorf("id parameter is required")
		}

		// Apply patch
		if err := h.cronSvc.UpdateJob(id, func(job *cron.Job) error {
			if patch, ok := params["patch"].(map[string]interface{}); ok {
				if name, ok := patch["name"].(string); ok {
					job.Name = name
				}
				if enabled, ok := patch["enabled"].(bool); ok {
					job.State.Enabled = enabled
				}
				if wm, ok := patch["wake_mode"].(string); ok {
					job.WakeMode = cron.WakeMode(wm)
				}
			}
			job.UpdatedAt = time.Now()
			return nil
		}); err != nil {
			return nil, fmt.Errorf("failed to update job: %w", err)
		}

		// Get updated job
		job, err := h.cronSvc.GetJob(id)
		if err != nil {
			return nil, fmt.Errorf("failed to get updated job: %w", err)
		}

		return job, nil
	})

	// cron.remove - 删除 cron 任务
	h.registry.Register("cron.remove", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.cronSvc == nil {
			return nil, fmt.Errorf("cron service is not available")
		}

		id, ok := params["id"].(string)
		if !ok {
			return nil, fmt.Errorf("id parameter is required")
		}

		if err := h.cronSvc.RemoveJob(id); err != nil {
			return nil, fmt.Errorf("failed to remove job: %w", err)
		}

		return map[string]interface{}{
			"status": "removed",
			"id":     id,
		}, nil
	})

	// cron.run - 运行 cron 任务
	h.registry.Register("cron.run", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.cronSvc == nil {
			return nil, fmt.Errorf("cron service is not available")
		}

		id, ok := params["id"].(string)
		if !ok {
			return nil, fmt.Errorf("id parameter is required")
		}

		modeStr := "normal"
		if m, ok := params["mode"].(string); ok {
			modeStr = m
		}
		if modeStr != "normal" && modeStr != "force" {
			return nil, fmt.Errorf("invalid mode: %s (expected normal or force)", modeStr)
		}

		var force bool
		if modeStr == "force" {
			force = true
		}

		if err := h.cronSvc.RunJob(context.Background(), id, force); err != nil {
			return nil, fmt.Errorf("failed to run job: %w", err)
		}

		return map[string]interface{}{
			"status": "run_requested",
			"id":     id,
			"mode":   modeStr,
		}, nil
	})

	// cron.runs - 获取 cron 任务运行历史
	h.registry.Register("cron.runs", func(sessionID string, params map[string]interface{}) (interface{}, error) {
		if h.cronSvc == nil {
			return nil, fmt.Errorf("cron service is not available")
		}

		id, ok := params["id"].(string)
		if !ok {
			return nil, fmt.Errorf("id parameter is required")
		}

		limit := 50
		if l, ok := params["limit"].(float64); ok {
			limit = int(l)
			if limit <= 0 {
				limit = 50
			}
		}

		// Create filter
		filter := cron.RunLogFilter{Limit: limit}

		runs, err := h.cronSvc.GetRunLogs(id, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to get run logs: %w", err)
		}

		return map[string]interface{}{
			"job_id": id,
			"runs":   runs,
			"count":  len(runs),
		}, nil
	})
}

package gateway

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/smallnest/goclaw/bus"
	"github.com/smallnest/goclaw/channels"
	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/cron"
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

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     checkWebSocketOrigin,
	Subprotocols:    []string{"goclaw.v1"},
}

// SetHarnessService attaches the optional Better-Harness control plane before
// or after the gateway starts.
func (s *Server) SetHarnessService(service *harness.Service) {
	if s.handler != nil {
		s.handler.SetHarnessService(service)
	}
}

func (s *Server) SetMemoryCatalog(service *catalog.Service) {
	if s.handler != nil {
		s.handler.SetMemoryCatalog(service)
	}
}

func (s *Server) SetDevelopmentService(service *orchestratorlite.Service) {
	if s.handler != nil {
		s.handler.SetDevelopmentService(service)
	}
}

func (s *Server) SetOuroborosService(service *ouroboros.Service) {
	if s.handler != nil {
		s.handler.SetOuroborosService(service)
	}
}

// SetTeamControlService enables the second authentication layer and project
// authorization. The configured Gateway bearer remains the outer perimeter.
func (s *Server) SetTeamControlService(service *teamcontrol.Service) {
	s.mu.Lock()
	s.teamSvc = service
	s.mu.Unlock()
	if s.handler != nil {
		s.handler.SetTeamControlService(service)
	}
}

func (s *Server) SetWorkstationService(service *workstation.Service) {
	s.mu.Lock()
	s.runnerSvc = service
	s.mu.Unlock()
	if s.handler != nil {
		s.handler.SetWorkstationService(service)
	}
}

// Server HTTP 网关服务器
type Server struct {
	config        *config.Config
	wsConfig      *WebSocketConfig
	bus           *bus.MessageBus
	channelMgr    *channels.Manager
	sessionMgr    *session.Manager
	server        *http.Server
	wsServer      *http.Server
	handler       *Handler
	mu            sync.RWMutex
	running       bool
	connections   map[string]*Connection
	connectionsMu sync.RWMutex
	enableAuth    bool
	authToken     string
	teamSvc       *teamcontrol.Service
	runnerSvc     *workstation.Service
	webSessions   *webSessionStore
}

// WebSocketConfig WebSocket 配置
type WebSocketConfig struct {
	Host           string
	Port           int
	Path           string
	EnableAuth     bool
	AuthToken      string
	PingInterval   time.Duration
	PongTimeout    time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxMessageSize int64
	// TLS 配置
	EnableTLS bool
	CertFile  string
	KeyFile   string
}

// NewServer 创建网关服务器
func NewServer(cfg *config.Config, messageBus *bus.MessageBus, channelMgr *channels.Manager, sessionMgr *session.Manager, cronSvc *cron.Service) *Server {
	// 从配置文件获取 WebSocket 设置，如果未配置则使用默认值
	wsPort := cfg.Gateway.WebSocket.Port
	if wsPort == 0 {
		wsPort = 28789 // 默认端口
	}
	wsHost := cfg.Gateway.WebSocket.Host
	if wsHost == "" {
		wsHost = "0.0.0.0" // 默认监听地址
	}
	wsPath := cfg.Gateway.WebSocket.Path
	if wsPath == "" {
		wsPath = "/ws" // 默认路径
	}
	pingInterval := cfg.Gateway.WebSocket.PingInterval
	if pingInterval == 0 {
		pingInterval = 30 * time.Second
	}
	pongTimeout := cfg.Gateway.WebSocket.PongTimeout
	if pongTimeout == 0 {
		pongTimeout = 60 * time.Second
	}
	readTimeout := cfg.Gateway.WebSocket.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 60 * time.Second
	}
	writeTimeout := cfg.Gateway.WebSocket.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 10 * time.Second
	}

	return &Server{
		config: cfg,
		wsConfig: &WebSocketConfig{
			Host:           wsHost,
			Port:           wsPort,
			Path:           wsPath,
			EnableAuth:     cfg.Gateway.WebSocket.EnableAuth,
			AuthToken:      cfg.Gateway.WebSocket.AuthToken,
			PingInterval:   pingInterval,
			PongTimeout:    pongTimeout,
			ReadTimeout:    readTimeout,
			WriteTimeout:   writeTimeout,
			MaxMessageSize: 10 * 1024 * 1024, // 10MB
		},
		bus:         messageBus,
		channelMgr:  channelMgr,
		sessionMgr:  sessionMgr,
		handler:     NewHandler(messageBus, sessionMgr, channelMgr, cronSvc, cfg),
		connections: make(map[string]*Connection),
		enableAuth:  cfg.Gateway.WebSocket.EnableAuth,
		authToken:   cfg.Gateway.WebSocket.AuthToken,
		webSessions: newWebSessionStore(12 * time.Hour),
	}
}

// SetWebSocketConfig 设置 WebSocket 配置
func (s *Server) SetWebSocketConfig(cfg *WebSocketConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.wsConfig = cfg
	s.enableAuth = cfg.EnableAuth
	s.authToken = cfg.AuthToken
}

// Start 启动服务器
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	// 启动 HTTP 服务器
	if err := s.startHTTPServer(ctx); err != nil {
		return err
	}

	// 启动 WebSocket 服务器
	if err := s.startWebSocketServer(ctx); err != nil {
		return err
	}

	// 启动出站消息广播（使用新的订阅机制）
	go s.broadcastOutbound(ctx)

	// 启动聊天事件广播
	go s.broadcastChatEvents(ctx)

	// 监听上下文取消
	go func() {
		<-ctx.Done()
		_ = s.Stop()
	}()

	return nil
}

// startHTTPServer 启动 HTTP 服务器
func (s *Server) startHTTPServer(ctx context.Context) error {
	// 创建 HTTP 路由
	mux := http.NewServeMux()

	// JSON-RPC 端点
	mux.HandleFunc("/rpc", s.handleJSONRPC)
	mux.HandleFunc("/auth/session", s.handleWebSession)

	// 健康检查端点
	mux.HandleFunc("/health", s.handleHealth)

	// Channels API 端点
	mux.HandleFunc("/api/channels", s.handleChannelsAPI)

	// 飞书 webhook 端点
	mux.HandleFunc("/webhook/feishu", s.handleFeishuWebhook)

	// 通用 webhook 端点
	mux.HandleFunc("/webhook/", s.handleGenericWebhook)

	// UI 静态文件服务
	mux.Handle("/ui/", UIStaticHandler())
	mux.Handle("/assets/", UIStaticHandler())

	// 创建 HTTP 服务器
	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Gateway.Host, s.config.Gateway.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(s.config.Gateway.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.config.Gateway.WriteTimeout) * time.Second,
	}

	// 启动服务器
	go func() {
		logger.Info("HTTP gateway server started",
			zap.String("addr", s.server.Addr),
		)

		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP gateway server error", zap.Error(err))
		}
	}()

	return nil
}

// startWebSocketServer 启动 WebSocket 服务器
func (s *Server) startWebSocketServer(ctx context.Context) error {
	// 创建 WebSocket 路由
	mux := http.NewServeMux()

	// WebSocket 端点
	mux.HandleFunc(s.wsConfig.Path, s.handleWebSocket)

	// JSON-RPC 端点
	// Dashboard UI is served from the WebSocket server port, so it must also expose /rpc.
	mux.HandleFunc("/rpc", s.handleJSONRPC)
	mux.HandleFunc("/auth/session", s.handleWebSession)

	// 健康检查端点
	mux.HandleFunc("/health", s.handleHealth)

	// Channels API 端点
	mux.HandleFunc("/api/channels", s.handleChannelsAPI)

	// The Team Console shell is public static content. Authentication is
	// established through /auth/session before any project data is returned.
	teamConsole := TeamConsoleHandler()
	mux.Handle("/dashboard/", teamConsole)
	mux.Handle("/dashboard", teamConsole)
	mux.Handle("/assets/", teamConsole)

	// 创建 WebSocket 服务器
	// 注意：不设置 ReadTimeout 和 WriteTimeout，因为 WebSocket 连接需要长连接
	// 心跳机制由 WebSocket 自己管理
	s.wsServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.wsConfig.Host, s.wsConfig.Port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		logger.Info("WebSocket gateway server started",
			zap.String("addr", s.wsServer.Addr),
			zap.String("path", s.wsConfig.Path),
			zap.Bool("gateway_auth", s.wsConfig.EnableAuth),
		)

		if err := s.wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("WebSocket gateway server error", zap.Error(err))
		}
	}()

	return nil
}

// getDashboardAuthConfig 获取 Dashboard 认证配置
// 如果启用了 WebSocket 认证且配置了 token，则需要认证（远程访问）
// 本地访问（127.0.0.1/localhost）会自动跳过认证
func (s *Server) getDashboardAuthConfig() *DashboardAuthConfig {
	// 如果启用了 WebSocket 认证且有 token，则需要认证
	if s.wsConfig.EnableAuth && s.wsConfig.AuthToken != "" {
		return &DashboardAuthConfig{
			RequireAuth: true,
			AuthToken:   s.wsConfig.AuthToken,
		}
	}

	// 否则不需要认证
	return &DashboardAuthConfig{
		RequireAuth: false,
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil
	}
	s.running = false
	s.mu.Unlock()

	// 关闭所有 WebSocket 连接
	s.closeAllConnections()

	// 停止 HTTP 服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown HTTP gateway server", zap.Error(err))
		}
	}

	// 停止 WebSocket 服务器
	if s.wsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.wsServer.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown WebSocket gateway server", zap.Error(err))
		}
	}

	logger.Info("Gateway server stopped")
	return nil
}

// closeAllConnections 关闭所有 WebSocket 连接
func (s *Server) closeAllConnections() {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()

	for id, conn := range s.connections {
		conn.Close()
		delete(s.connections, id)
	}
}

// addConnection 添加连接
func (s *Server) addConnection(conn *Connection) {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()
	s.connections[conn.ID] = conn
}

// removeConnection 移除连接
func (s *Server) removeConnection(id string) {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()
	delete(s.connections, id)
}

// _getConnection 获取连接 (未使用，保留供将来使用)
// nolint:unused
func (s *Server) _getConnection(id string) (*Connection, bool) {
	s.connectionsMu.RLock()
	defer s.connectionsMu.RUnlock()
	conn, ok := s.connections[id]
	return conn, ok
}

// IsRunning 检查是否运行中
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// handleHealth 健康检查处理器
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Unix(),
	})
}

// handleFeishuWebhook 飞书 webhook 处理器
func (s *Server) handleFeishuWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取飞书通道
	_, ok := s.channelMgr.Get("feishu")
	if !ok {
		http.Error(w, "Feishu channel not found", http.StatusServiceUnavailable)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read webhook body", zap.Error(err))
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// 验证签名（由通道处理）
	// 这里我们需要将请求传递给飞书通道处理
	// 由于接口限制，我们暂时记录日志

	logger.Debug("Received Feishu webhook",
		zap.Int("content_length", len(body)),
	)

	// 将事件发布到消息总线（由飞书通道解析）
	// 这里简化处理，实际应该由飞书通道解析并发布

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleGenericWebhook 通用 webhook 处理器
func (s *Server) handleGenericWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 从 URL 路径提取通道名称
	// /webhook/{channel}
	channelName := r.URL.Path[len("/webhook/"):]
	if channelName == "" {
		http.Error(w, "Channel not specified", http.StatusBadRequest)
		return
	}

	// 获取通道
	_, ok := s.channelMgr.Get(channelName)
	if !ok {
		http.Error(w, fmt.Sprintf("Channel %s not found", channelName), http.StatusServiceUnavailable)
		return
	}

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("Failed to read webhook body",
			zap.String("channel", channelName),
			zap.Error(err),
		)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	logger.Debug("Received webhook",
		zap.String("channel", channelName),
		zap.Int("content_length", len(body)),
	)

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// handleWebSocket WebSocket 连接处理器
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// The shared Gateway credential remains the outer authentication layer.
	// A browser that already exchanged that credential plus a personal Team
	// token may use its short-lived, revocable web session instead.
	if s.wsConfig.EnableAuth && !s.authenticateWebSocketPerimeter(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	principalID, err := s.authenticateTeamWebSocket(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 升级到 WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("Failed to upgrade to WebSocket", zap.Error(err))
		return
	}

	// 创建连接对象
	connection := NewConnection(conn, s.wsConfig)
	connection.PrincipalID = principalID
	if principalID != "" {
		connection.SessionID = teamSessionID(principalID)
	}
	sessionID := connection.SessionID

	// 添加到连接管理
	s.addConnection(connection)

	logger.Info("WebSocket connection established",
		zap.String("session_id", sessionID),
		zap.String("remote_addr", r.RemoteAddr),
	)

	// 发送欢迎消息
	welcome := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "connected",
		Params: map[string]interface{}{
			"session_id":    sessionID,
			"connection_id": connection.ID,
			"principal_id":  principalID,
			"version":       ProtocolVersion,
		},
	}
	if err := connection.SendJSON(welcome); err != nil {
		logger.Error("Failed to send welcome message", zap.Error(err))
	}

	logger.Debug("Welcome message sent, starting heartbeat and message handler",
		zap.String("session_id", sessionID))

	// 启动心跳
	go connection.heartbeat()

	// 处理消息 (阻塞直到连接关闭)
	s.handleWebSocketMessages(connection)

	logger.Debug("handleWebSocketMessages returned, connection handling complete",
		zap.String("session_id", sessionID))
}

func (s *Server) authenticateWebSocketPerimeter(r *http.Request) bool {
	if s.authenticateWebSocket(r) {
		return true
	}
	_, _, ok := s.authenticateWebSession(r)
	return ok
}

// authenticateWebSocket 验证 WebSocket 连接
func (s *Server) authenticateWebSocket(r *http.Request) bool {
	// 从查询参数获取 token
	token := r.URL.Query().Get("token")
	if token == "" {
		// 从 Authorization header 获取
		auth := r.Header.Get("Authorization")
		if auth != "" {
			// 支持 "Bearer <token>" 格式
			if len(auth) > 7 && auth[:7] == "Bearer " {
				token = auth[7:]
			}
		}
	}
	if token == "" {
		if cookie, err := r.Cookie("dashboard_token"); err == nil {
			token = cookie.Value
		}
	}
	if token == "" {
		for _, protocol := range websocket.Subprotocols(r) {
			const prefix = "goclaw.bearer."
			if len(protocol) <= len(prefix) || protocol[:len(prefix)] != prefix {
				continue
			}
			decoded, err := base64.RawURLEncoding.DecodeString(protocol[len(prefix):])
			if err == nil {
				token = string(decoded)
				break
			}
		}
	}

	if token == "" {
		return false
	}

	// 使用恒定时间比较防止时序攻击
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.authToken)) == 1
}

func (s *Server) authenticateTeamWebSocket(r *http.Request) (string, error) {
	s.mu.RLock()
	service := s.teamSvc
	s.mu.RUnlock()
	if service == nil {
		return "", nil
	}

	if principalID, _, ok := s.authenticateWebSession(r); ok {
		return principalID, nil
	}

	// Browser WebSocket APIs cannot add arbitrary headers, so the subprotocol
	// is the primary transport. A header remains useful for native clients.
	token := tokenFromSubprotocol(r, "goclaw.user.")
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-GoClaw-User-Token"))
	}
	if token == "" {
		return "", errUnauthenticatedPrincipal
	}
	user, err := service.AuthenticateAccessToken(token)
	if err != nil {
		return "", err
	}
	return user.ID, nil
}

func (s *Server) authenticateHTTP(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	token := ""
	if len(auth) > len(prefix) && auth[:len(prefix)] == prefix {
		token = auth[len(prefix):]
	}
	if token == "" {
		if cookie, err := r.Cookie("dashboard_token"); err == nil {
			token = cookie.Value
		}
	}
	if token == "" || s.authToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.authToken)) == 1
}

func (s *Server) authenticateHTTPPerimeter(r *http.Request) bool {
	if s.authenticateHTTP(r) {
		return true
	}
	_, _, ok := s.authenticateWebSession(r)
	return ok
}

func (s *Server) authenticateTeamHTTP(r *http.Request) (string, *webSession, error) {
	s.mu.RLock()
	service := s.teamSvc
	s.mu.RUnlock()
	if service == nil {
		return "", nil, nil
	}
	if principalID, session, ok := s.authenticateWebSession(r); ok {
		return principalID, session, nil
	}
	token := strings.TrimSpace(r.Header.Get("X-GoClaw-User-Token"))
	if token == "" {
		return "", nil, errUnauthenticatedPrincipal
	}
	user, err := service.AuthenticateAccessToken(token)
	if err != nil {
		return "", nil, err
	}
	return user.ID, nil, nil
}

func tokenFromSubprotocol(r *http.Request, prefix string) string {
	for _, protocol := range websocket.Subprotocols(r) {
		if !strings.HasPrefix(protocol, prefix) || len(protocol) <= len(prefix) {
			continue
		}
		decoded, err := base64.RawURLEncoding.DecodeString(protocol[len(prefix):])
		if err == nil {
			return string(decoded)
		}
	}
	return ""
}

// handleWebSocketMessages 处理 WebSocket 消息
func (s *Server) handleWebSocketMessages(conn *Connection) {
	defer func() {
		conn.Close()
		s.removeConnection(conn.ID)
		logger.Info("WebSocket connection closed",
			zap.String("session_id", conn.ID),
		)
	}()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			// 检查是否是正常关闭
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				logger.Info("WebSocket closed by client",
					zap.String("session_id", conn.ID),
					zap.Error(err))
			} else {
				logger.Info("WebSocket read error, closing connection",
					zap.String("session_id", conn.ID),
					zap.Error(err))
			}
			break
		}

		logger.Debug("WebSocket message received",
			zap.String("session_id", conn.ID),
			zap.Int("message_type", messageType),
			zap.Int("data_len", len(data)),
		)

		// 只处理文本消息
		if messageType != websocket.TextMessage {
			continue
		}

		// 解析请求
		req, err := ParseRequest(data)
		if err != nil {
			logger.Error("Failed to parse WebSocket message",
				zap.String("session_id", conn.ID),
				zap.Error(err))
			errorResp := NewErrorResponse("", ErrorParseError, "Parse error")
			_ = conn.SendJSON(errorResp)
			continue
		}

		logger.Debug("WebSocket request",
			zap.String("session_id", conn.ID),
			zap.String("method", req.Method),
		)

		// 处理请求
		resp := s.handler.HandleRequest(conn.SessionID, req)

		// 发送响应
		if err := conn.SendJSON(resp); err != nil {
			logger.Error("Failed to send WebSocket response",
				zap.String("session_id", conn.ID),
				zap.Error(err))
		}
	}
}

// broadcastOutbound 广播出站消息到所有 WebSocket 连接
func (s *Server) broadcastOutbound(ctx context.Context) {
	logger.Debug("Starting WebSocket outbound broadcaster")

	// 订阅出站消息
	subscription := s.bus.SubscribeOutbound()
	defer subscription.Unsubscribe()

	logger.Debug("WebSocket broadcaster subscribed",
		zap.String("subscription_id", subscription.ID))

	busChan := subscription.Channel

	for {
		select {
		case <-ctx.Done():
			logger.Debug("WebSocket outbound broadcaster stopped")
			return
		case msg, ok := <-busChan:
			if !ok {
				logger.Debug("Outbound channel closed, exiting broadcaster")
				return
			}
			if msg == nil {
				continue
			}

			logger.Debug("Broadcasting to WebSocket connections",
				zap.String("channel", msg.Channel),
				zap.String("chat_id", msg.ChatID),
				zap.Int("connections", len(s.connections)))

			// 如果是 gateway channel 的消息，定向发送给特定 session
			if msg.Channel == "gateway" {
				service := s.teamControlService()
				scope, scopeOK := resolveProjectBroadcastScope(
					msg.Channel,
					msg.ChatID,
					msg.Metadata,
				)
				if service != nil && !scopeOK {
					logger.Warn("Dropping team outbound message with invalid project scope",
						zap.String("channel", msg.Channel),
						zap.String("chat_id", msg.ChatID))
					continue
				}
				projectID := gatewayProjectID(msg.ChatID, msg.Metadata)
				topicID := gatewayTopicID(msg.ChatID, msg.Metadata)
				if scopeOK {
					projectID = scope.ProjectID
					topicID = scope.TopicID
				}
				s.connectionsMu.RLock()
				found := false
				for _, conn := range s.connections {
					deliver := projectBroadcastAllowed(
						service,
						conn,
						conn.SessionID == msg.ChatID,
						scope,
						scopeOK,
					)
					if deliver {
						found = true
						// 创建通知
						notif, err := s.handler.BroadcastNotification("chat.response", map[string]interface{}{
							"content":    msg.Content,
							"project_id": projectID,
							"topic_id":   topicID,
							"timestamp":  msg.Timestamp,
						})
						if err != nil {
							logger.Error("Failed to create notification", zap.Error(err))
							continue
						}

						// 发送通知
						if err := conn.SendMessage(websocket.TextMessage, notif); err != nil {
							logger.Error("Failed to send notification",
								zap.String("session_id", conn.ID),
								zap.Error(err))
						} else {
							logger.Info("Chat response sent to WebSocket client",
								zap.String("session_id", conn.ID),
								zap.String("content", msg.Content))
						}
					}
				}
				if !found {
					logger.Warn("No WebSocket connection found for gateway message",
						zap.String("chat_id", msg.ChatID))
				}
				s.connectionsMu.RUnlock()
				continue
			}

			// 其他 channel 的消息，广播到所有连接
			service := s.teamControlService()
			scope, scopeOK := resolveProjectBroadcastScope(
				msg.Channel,
				msg.ChatID,
				msg.Metadata,
			)
			if service != nil && !scopeOK {
				logger.Warn("Dropping team outbound message with invalid project scope",
					zap.String("channel", msg.Channel),
					zap.String("chat_id", msg.ChatID))
				continue
			}
			s.connectionsMu.RLock()
			for _, conn := range s.connections {
				if !projectBroadcastAllowed(
					service,
					conn,
					true,
					scope,
					scopeOK,
				) {
					continue
				}
				// 创建通知
				notif, err := s.handler.BroadcastNotification("message.outbound", map[string]interface{}{
					"channel":    msg.Channel,
					"chat_id":    msg.ChatID,
					"content":    msg.Content,
					"metadata":   msg.Metadata,
					"project_id": scope.ProjectID,
					"topic_id":   scope.TopicID,
					"timestamp":  msg.Timestamp,
				})
				if err != nil {
					logger.Error("Failed to create notification", zap.Error(err))
					continue
				}

				// 发送通知
				if err := conn.SendMessage(websocket.TextMessage, notif); err != nil {
					logger.Error("Failed to broadcast notification",
						zap.String("session_id", conn.ID),
						zap.Error(err))
				}
			}
			s.connectionsMu.RUnlock()
		}
	}
}

// broadcastChatEvents 广播聊天事件到 WebSocket 连接
func (s *Server) broadcastChatEvents(ctx context.Context) {
	logger.Debug("Starting WebSocket chat event broadcaster")

	// 订阅聊天事件
	subscription := s.bus.SubscribeChatEvent()
	defer subscription.Unsubscribe()

	logger.Debug("WebSocket chat event broadcaster subscribed",
		zap.String("subscription_id", subscription.ID))

	for {
		select {
		case <-ctx.Done():
			logger.Debug("WebSocket chat event broadcaster stopped")
			return
		case event, ok := <-subscription.Channel:
			if !ok {
				logger.Debug("Chat event channel closed, exiting broadcaster")
				return
			}
			if event == nil {
				continue
			}

			logger.Debug("Broadcasting chat event",
				zap.String("channel", event.Channel),
				zap.String("chat_id", event.ChatID),
				zap.String("state", event.State),
				zap.Int("seq", event.Seq))

			// 如果是 gateway channel 的消息，定向发送给特定 session
			if event.Channel == "gateway" {
				service := s.teamControlService()
				scope, scopeOK := resolveProjectBroadcastScope(
					event.Channel,
					event.ChatID,
					event.Metadata,
				)
				if service != nil && !scopeOK {
					logger.Warn("Dropping team chat event with invalid project scope",
						zap.String("channel", event.Channel),
						zap.String("chat_id", event.ChatID),
						zap.String("run_id", event.RunID))
					continue
				}
				projectID := gatewayProjectID(event.ChatID, event.Metadata)
				topicID := gatewayTopicID(event.ChatID, event.Metadata)
				if scopeOK {
					projectID = scope.ProjectID
					topicID = scope.TopicID
				}
				s.connectionsMu.RLock()
				for _, conn := range s.connections {
					deliver := projectBroadcastAllowed(
						service,
						conn,
						conn.SessionID == event.ChatID,
						scope,
						scopeOK,
					)
					if deliver {
						// 创建事件通知
						notif, err := s.handler.BroadcastNotification("chat.event", map[string]interface{}{
							"run_id":     event.RunID,
							"seq":        event.Seq,
							"state":      event.State,
							"content":    event.Content,
							"project_id": projectID,
							"topic_id":   topicID,
							"timestamp":  event.Timestamp,
						})
						if err != nil {
							logger.Error("Failed to create chat event notification", zap.Error(err))
							continue
						}

						// 发送通知
						if err := conn.SendMessage(websocket.TextMessage, notif); err != nil {
							logger.Error("Failed to send chat event notification",
								zap.String("session_id", conn.ID),
								zap.Error(err))
						}
					}
				}
				s.connectionsMu.RUnlock()
				continue
			}

			// 其他 channel 的事件，广播到所有连接
			service := s.teamControlService()
			scope, scopeOK := resolveProjectBroadcastScope(
				event.Channel,
				event.ChatID,
				event.Metadata,
			)
			if service != nil && !scopeOK {
				logger.Warn("Dropping team chat event with invalid project scope",
					zap.String("channel", event.Channel),
					zap.String("chat_id", event.ChatID),
					zap.String("run_id", event.RunID))
				continue
			}
			s.connectionsMu.RLock()
			for _, conn := range s.connections {
				if !projectBroadcastAllowed(
					service,
					conn,
					true,
					scope,
					scopeOK,
				) {
					continue
				}
				// 创建事件通知
				notif, err := s.handler.BroadcastNotification("chat.event", map[string]interface{}{
					"channel":    event.Channel,
					"chat_id":    event.ChatID,
					"run_id":     event.RunID,
					"seq":        event.Seq,
					"state":      event.State,
					"content":    event.Content,
					"metadata":   event.Metadata,
					"project_id": scope.ProjectID,
					"topic_id":   scope.TopicID,
					"timestamp":  event.Timestamp,
				})
				if err != nil {
					logger.Error("Failed to create chat event notification", zap.Error(err))
					continue
				}

				// 发送通知
				if err := conn.SendMessage(websocket.TextMessage, notif); err != nil {
					logger.Error("Failed to broadcast chat event notification",
						zap.String("session_id", conn.ID),
						zap.Error(err))
				}
			}
			s.connectionsMu.RUnlock()
		}
	}
}

func (s *Server) connectionCanReadProject(conn *Connection, projectID string) bool {
	service := s.teamControlService()
	if service == nil {
		return true
	}
	if conn == nil || conn.PrincipalID == "" || projectID == "" {
		return false
	}
	return service.Authorize(
		conn.PrincipalID,
		projectID,
		teamcontrol.ActionProjectRead,
	) == nil
}

func (s *Server) teamControlService() *teamcontrol.Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.teamSvc
}

func (s *Server) teamControlEnabled() bool {
	return s.teamControlService() != nil
}

func metadataProjectID(metadata interface{}) string {
	switch value := metadata.(type) {
	case map[string]interface{}:
		projectID, _ := value["project_id"].(string)
		return strings.TrimSpace(projectID)
	case map[string]string:
		return strings.TrimSpace(value["project_id"])
	default:
		return ""
	}
}

func metadataTopicID(metadata interface{}) string {
	switch value := metadata.(type) {
	case map[string]interface{}:
		topicID, _ := value["topic_id"].(string)
		return strings.TrimSpace(topicID)
	case map[string]string:
		return strings.TrimSpace(value["topic_id"])
	default:
		return ""
	}
}

func gatewayProjectID(chatID string, metadata interface{}) string {
	if projectID := metadataProjectID(metadata); projectID != "" {
		return projectID
	}
	const prefix = "project:"
	const separator = ":topic:"
	if !strings.HasPrefix(chatID, prefix) {
		return ""
	}
	remainder := strings.TrimPrefix(chatID, prefix)
	index := strings.Index(remainder, separator)
	if index <= 0 {
		return ""
	}
	return strings.TrimSpace(remainder[:index])
}

func gatewayTopicID(chatID string, metadata interface{}) string {
	if topicID := metadataTopicID(metadata); topicID != "" {
		return topicID
	}
	const separator = ":topic:"
	index := strings.Index(chatID, separator)
	if index < 0 {
		return ""
	}
	return strings.TrimSpace(chatID[index+len(separator):])
}

// Connection WebSocket 连接
type Connection struct {
	*websocket.Conn
	ID          string
	SessionID   string
	PrincipalID string
	// nolint:unused
	_sessionID   string // 保留供将来使用
	pingInterval time.Duration
	pongTimeout  time.Duration
	mu           sync.Mutex
}

// NewConnection 创建连接
func NewConnection(ws *websocket.Conn, cfg *WebSocketConfig) *Connection {
	id := uuid.New().String()
	return &Connection{
		Conn:         ws,
		ID:           id,
		SessionID:    id,
		pingInterval: cfg.PingInterval,
		pongTimeout:  cfg.PongTimeout,
	}
}

// SendJSON 发送 JSON 消息
func (c *Connection) SendJSON(v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.WriteJSON(v)
}

// SendMessage 发送消息
func (c *Connection) SendMessage(messageType int, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.WriteMessage(messageType, data)
}

// heartbeat 心跳
func (c *Connection) heartbeat() {
	// 设置 pong 处理器，当收到 pong 响应时重置读取截止时间
	c.SetPongHandler(func(string) error {
		return nil
	})

	ticker := time.NewTicker(c.pingInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		if err := c.WriteMessage(websocket.PingMessage, nil); err != nil {
			c.mu.Unlock()
			return
		}
		c.mu.Unlock()
	}
}

// handleChannelsAPI 处理 channels API 请求
func (s *Server) handleChannelsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取查询参数
	channelName := r.URL.Query().Get("channel")

	w.Header().Set("Content-Type", "application/json")

	if channelName != "" {
		// 获取特定 channel 的状态
		status, err := s.channelMgr.Status(channelName)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": err.Error(),
			})
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(status)
	} else {
		// 获取所有 channels 列表和状态
		channelNames := s.channelMgr.List()
		channels := make([]map[string]interface{}, 0, len(channelNames))

		for _, name := range channelNames {
			status, _ := s.channelMgr.Status(name)
			channels = append(channels, status)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"channels": channels,
			"count":    len(channels),
		})
	}
}

// handleJSONRPC 处理 JSON-RPC 请求
func (s *Server) handleJSONRPC(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.enableAuth && !s.authenticateHTTPPerimeter(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	principalID, browserSession, err := s.authenticateTeamHTTP(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if browserSession != nil && !validCSRFHeader(r, browserSession.CSRFToken) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// 读取请求体
	maxMessageSize := int64(10 * 1024 * 1024)
	if s.wsConfig != nil && s.wsConfig.MaxMessageSize > 0 {
		maxMessageSize = s.wsConfig.MaxMessageSize
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxMessageSize))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 解析请求
	req, err := ParseRequest(body)
	if err != nil {
		logger.Error("Failed to parse JSON-RPC request", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logger.Debug("HTTP JSON-RPC request",
		zap.String("method", req.Method),
		zap.String("id", req.ID))

	// 处理请求
	sessionID := ""
	if principalID != "" {
		sessionID = teamSessionID(principalID)
	}
	resp := s.handler.HandleRequest(sessionID, req)

	// 发送响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Close 关闭连接
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 发送关闭帧
	message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	_ = c.WriteMessage(websocket.CloseMessage, message)

	// 关闭连接
	return c.Conn.Close()
}

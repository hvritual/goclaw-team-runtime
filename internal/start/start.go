package start

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smallnest/goclaw/agent"
	"github.com/smallnest/goclaw/agent/tools"
	"github.com/smallnest/goclaw/bus"
	"github.com/smallnest/goclaw/channels"
	"github.com/smallnest/goclaw/config"
	"github.com/smallnest/goclaw/cron"
	"github.com/smallnest/goclaw/gateway"
	"github.com/smallnest/goclaw/harness"
	"github.com/smallnest/goclaw/integration/ouroborosprovider"
	"github.com/smallnest/goclaw/internal"
	"github.com/smallnest/goclaw/internal/logger"
	"github.com/smallnest/goclaw/internal/workspace"
	"github.com/smallnest/goclaw/memory/catalog"
	"github.com/smallnest/goclaw/orchestratorlite"
	"github.com/smallnest/goclaw/ouroboros"
	"github.com/smallnest/goclaw/providers"
	"github.com/smallnest/goclaw/session"
	"github.com/smallnest/goclaw/teamcontrol"
	"github.com/smallnest/goclaw/workstation"
	"go.uber.org/zap"
)

// Config holds configuration for starting the agent
type Config struct {
	LogLevel string
}

// StartAgent starts the goclaw agent with all its components
// This is shared between `goclaw start` and `goclaw gateway run`
func StartAgent(cfg *Config) error {
	// 确保内置技能被复制到用户目录
	if err := internal.EnsureBuiltinSkills(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to ensure builtin skills: %v\n", err)
	}

	// 确保配置文件存在
	configCreated, err := internal.EnsureConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to ensure config: %v\n", err)
	}
	if configCreated {
		fmt.Println("Config file created at: " + internal.GetConfigPath())
		fmt.Println("Please edit the config file to set your API keys and other settings.")
		fmt.Println()
	}

	// 加载配置
	configFile, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		return err
	}
	if err := rejectPilotMaintenanceMode(); err != nil {
		return err
	}

	// 初始化日志
	logLevel := cfg.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}
	if err := logger.Init(logLevel, false); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		return err
	}
	defer func() { _ = logger.Sync() }()

	logger.Info("Starting goclaw agent")

	// 验证配置
	if err := config.Validate(configFile); err != nil {
		logger.Fatal("Invalid configuration", zap.Error(err))
	}

	// 获取 workspace 目录
	workspaceDir, err := config.GetWorkspacePath(configFile)
	if err != nil {
		logger.Fatal("Failed to get workspace path", zap.Error(err))
	}

	// 创建 workspace 管理器并确保文件存在
	workspaceMgr := workspace.NewManager(workspaceDir)
	if err := workspaceMgr.Ensure(); err != nil {
		logger.Warn("Failed to ensure workspace files", zap.Error(err))
	} else {
		logger.Info("Workspace ready", zap.String("path", workspaceDir))
	}

	// 创建消息总线
	messageBus := bus.NewMessageBus(100)
	defer messageBus.Close()

	// 创建会话管理器
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Fatal("Failed to get home directory", zap.Error(err))
	}
	sessionDir := homeDir + "/.goclaw/sessions"
	sessionMgr, err := session.NewManager(sessionDir)
	if err != nil {
		logger.Fatal("Failed to create session manager", zap.Error(err))
	}

	// 创建记忆存储
	memoryStore := agent.NewMemoryStore(workspaceDir)

	// 创建上下文构建器
	contextBuilder := agent.NewContextBuilder(memoryStore, workspaceDir)

	// Initialize the governed catalog independently from embedding/QMD
	// backends. It requires no API key and is the only memory source that may
	// be injected automatically into model context.
	var catalogService *catalog.Service
	if configFile.Memory.Catalog.Enabled {
		catalogService, err = catalog.NewService(configFile.Memory.Catalog)
		if err != nil {
			return fmt.Errorf("failed to initialize memory catalog: %w", err)
		}
		defer func() { _ = catalogService.Close() }()
		catalogService.SetGovernancePolicy(configFile.Governance)
		contextBuilder.SetCatalogSource(catalogService)
		if configFile.Memory.Catalog.AutoIngest {
			for _, sourcePath := range configFile.Memory.Catalog.SourcePaths {
				report, ingestErr := catalogService.IngestPath(sourcePath, catalog.IngestOptions{
					ProjectID:      configFile.Memory.Catalog.DefaultProject,
					SourceRoot:     configFile.Memory.Catalog.SourceRoot,
					SourceScheme:   configFile.Memory.Catalog.SourceScheme,
					SourceKind:     configFile.Memory.Catalog.SourceKind,
					SourceRevision: configFile.Memory.Catalog.SourceRevision,
					Actor:          "startup-importer",
				})
				if ingestErr != nil {
					logger.Warn("Memory catalog auto-ingest failed",
						zap.String("path", sourcePath),
						zap.Error(ingestErr))
					continue
				}
				logger.Info("Memory catalog source scanned",
					zap.String("path", sourcePath),
					zap.Int("created", report.Created),
					zap.Int("existing", report.Existing),
					zap.Int("failed", report.Failed))
			}
		}
		logger.Info("Memory catalog ready",
			zap.String("database", catalogService.Config().DatabasePath),
			zap.String("default_project", catalogService.Config().DefaultProject))
	}

	// Initialize Better-Harness before agents and gateway so every new run can
	// resolve the active version and persist a trace envelope.
	var harnessService *harness.Service
	if configFile.Harness.Enabled {
		harnessService, err = harness.NewService(configFile.Harness)
		if err != nil {
			return fmt.Errorf("failed to initialize better harness: %w", err)
		}
		harnessService.SetGovernancePolicy(configFile.Governance)
		contextBuilder.SetHarnessSource(harnessService)
		active, activeErr := harnessService.ActiveState()
		if activeErr != nil {
			return fmt.Errorf("failed to load active harness: %w", activeErr)
		}
		logger.Info("Better-Harness ready",
			zap.String("root", harnessService.Config().Root),
			zap.String("project_id", harnessService.ProjectID()),
			zap.String("active_version", active.Version))
	}

	var developmentService *orchestratorlite.Service
	if configFile.Development.Enabled {
		developmentService, err = orchestratorlite.NewService(configFile.Development)
		if err != nil {
			return fmt.Errorf("failed to initialize orchestrator lite: %w", err)
		}
		developmentService.SetGovernancePolicy(configFile.Governance)
		logger.Info("Orchestrator Lite ready",
			zap.String("root", developmentService.Config().Root),
			zap.String("repo_path", developmentService.Config().RepoPath),
			zap.String("worktree_root", developmentService.Config().WorktreeRoot),
			zap.Bool("gateway_execution", developmentService.Config().GatewayAllowExecution))
	}

	var teamControlService *teamcontrol.Service
	if configFile.TeamControl.Enabled {
		teamControlService, err = teamcontrol.NewService(configFile.TeamControl)
		if err != nil {
			return fmt.Errorf("failed to initialize team control: %w", err)
		}
		logger.Info("Team control ready",
			zap.String("root", teamControlService.Config().Root))
	}

	var workstationService *workstation.Service
	if configFile.Workstation.Enabled {
		workstationService, err = workstation.NewService(configFile.Workstation)
		if err != nil {
			return fmt.Errorf("failed to initialize workstation scheduler: %w", err)
		}
		logger.Info("Workstation scheduler ready",
			zap.String("root", workstationService.Config().Root),
			zap.Int("lease_seconds", workstationService.Config().LeaseDurationSeconds))
	}

	var ouroborosService *ouroboros.Service
	if configFile.Ouroboros.Enabled {
		ouroborosService, err = ouroboros.NewService(configFile.Ouroboros)
		if err != nil {
			return fmt.Errorf("failed to initialize Go-native Ouroboros: %w", err)
		}
		ouroborosService.SetGovernancePolicy(configFile.Governance)
		logger.Info("Go-native Ouroboros store ready",
			zap.String("root", ouroborosService.Config().Root),
			zap.Float64("ambiguity_threshold", ouroborosService.Config().AmbiguityThreshold),
			zap.Float64("convergence_threshold", ouroborosService.Config().ConvergenceThreshold))
	}

	// 创建工具注册表
	toolRegistry := agent.NewToolRegistry()

	// 创建技能加载器
	// 加载顺序（后加载的同名技能会覆盖前面的）：
	// 1. ./skills/ (当前目录，最高优先级)
	// 2. ${WORKSPACE}/skills/ (工作区目录)
	// 3. ~/.goclaw/skills/ (用户全局目录)
	goclawDir := homeDir + "/.goclaw"
	globalSkillsDir := goclawDir + "/skills"
	workspaceSkillsDir := workspaceDir + "/skills"
	currentSkillsDir := "./skills"

	skillsLoader := agent.NewSkillsLoader(goclawDir, []string{
		globalSkillsDir,    // 最先加载（最低优先级）
		workspaceSkillsDir, // 其次加载
		currentSkillsDir,   // 最后加载（最高优先级）
	})
	if err := skillsLoader.Discover(); err != nil {
		logger.Warn("Failed to discover skills", zap.Error(err))
	} else {
		skills := skillsLoader.List()
		if len(skills) > 0 {
			logger.Info("Skills loaded", zap.Int("count", len(skills)))
		}
	}

	// 注册文件系统工具
	fsTool := tools.NewFileSystemTool(configFile.Tools.FileSystem.AllowedPaths, configFile.Tools.FileSystem.DeniedPaths, workspaceDir)
	for _, tool := range fsTool.GetTools() {
		if err := toolRegistry.RegisterExisting(tool); err != nil {
			logger.Warn("Failed to register tool", zap.String("tool", tool.Name()))
		}
	}
	if harnessService != nil &&
		(configFile.Harness.KnowledgeRoot != "" || configFile.Harness.VaultPath != "") {
		for _, tool := range []tools.Tool{
			tools.NewSearchKnowledgeTool(harnessService),
			tools.NewReadKnowledgeTool(harnessService),
			tools.NewKnowledgeProposalTool(harnessService),
		} {
			if err := toolRegistry.RegisterExisting(tool); err != nil {
				logger.Warn("Failed to register governed knowledge tool",
					zap.String("tool", tool.Name()),
					zap.Error(err))
			}
		}
	}
	if catalogService != nil {
		for _, tool := range []tools.Tool{
			tools.NewCatalogMemorySearchTool(catalogService),
			tools.NewCatalogMemoryProposalTool(catalogService),
		} {
			if err := toolRegistry.RegisterExisting(tool); err != nil {
				logger.Warn("Failed to register catalog memory tool",
					zap.String("tool", tool.Name()),
					zap.Error(err))
			}
		}
	}

	// 注册 use_skill 工具（用于两阶段技能加载）
	if err := toolRegistry.RegisterExisting(tools.NewUseSkillTool()); err != nil {
		logger.Warn("Failed to register use_skill tool", zap.Error(err))
	}

	// 注册 Shell 工具
	shellTool := tools.NewShellTool(
		configFile.Tools.Shell.Enabled,
		configFile.Tools.Shell.AllowedCmds,
		configFile.Tools.Shell.DeniedCmds,
		configFile.Tools.Shell.Timeout,
		configFile.Tools.Shell.WorkingDir,
		configFile.Tools.Shell.Sandbox,
	)
	for _, tool := range shellTool.GetTools() {
		if err := toolRegistry.RegisterExisting(tool); err != nil {
			logger.Warn("Failed to register tool", zap.String("tool", tool.Name()))
		}
	}

	// 注册 Web 工具
	webTool := tools.NewWebTool(
		configFile.Tools.Web.SearchAPIKey,
		configFile.Tools.Web.SearchEngine,
		configFile.Tools.Web.Timeout,
	)
	for _, tool := range webTool.GetTools() {
		if err := toolRegistry.RegisterExisting(tool); err != nil {
			logger.Warn("Failed to register tool", zap.String("tool", tool.Name()))
		}
	}

	// 注册浏览器工具（如果启用）
	if configFile.Tools.Browser.Enabled {
		browserTool := tools.NewBrowserTool(
			configFile.Tools.Browser.Headless,
			configFile.Tools.Browser.Timeout,
		)
		for _, tool := range browserTool.GetTools() {
			if err := toolRegistry.RegisterExisting(tool); err != nil {
				logger.Warn("Failed to register tool", zap.String("tool", tool.Name()))
			}
		}
		logger.Info("Browser tools registered")
	}

	// 注册 Cron 工具
	// 注意：cronTool 将在创建 cronService 后注册

	// 创建 LLM 提供商
	provider, err := providers.NewProvider(configFile)
	if err != nil {
		logger.Fatal("Failed to create LLM provider", zap.Error(err))
	}
	defer provider.Close()
	if ouroborosService != nil {
		adapter, adapterErr := ouroborosprovider.New(provider)
		if adapterErr != nil {
			return fmt.Errorf("failed to initialize Ouroboros provider adapter: %w", adapterErr)
		}
		ouroborosService.SetModel(adapter)
		for _, tool := range tools.NewOuroborosTools(
			ouroborosService,
			configFile.Development.RepoPath,
		) {
			if registerErr := toolRegistry.RegisterExisting(tool); registerErr != nil {
				return fmt.Errorf("register Ouroboros tool %s: %w", tool.Name(), registerErr)
			}
		}
		logger.Info("Go-native Ouroboros model and channel tools ready")
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if workstationService != nil {
		go recoverWorkstationLeases(ctx, workstationService)
	}

	// 创建通道管理器
	channelMgr := channels.NewManager(messageBus)
	if err := channelMgr.SetupFromConfig(configFile); err != nil {
		logger.Warn("Failed to setup channels from config", zap.Error(err))
	}

	// 创建 Cron 服务（需要在 Gateway 之前创建，因为 Handler 需要 cronService）
	cronService, err := cron.NewService(cron.DefaultCronConfig(), messageBus)
	if err != nil {
		logger.Warn("Failed to create cron service", zap.Error(err))
	}
	if cronService != nil {
		if err := cronService.Start(ctx); err != nil {
			logger.Warn("Failed to start cron service", zap.Error(err))
		}
		defer func() { _ = cronService.Stop() }()
	}

	// 注册 Cron 工具（使用已创建并启动的 cronService）
	if configFile.Tools.Cron.Enabled {
		logger.Info("Registering cron tools",
			zap.Bool("cron_service_nil", cronService == nil))
		cronTool := tools.NewCronTool(cronService)
		tools := cronTool.GetTools()
		logger.Info("CronTool.GetTools returned",
			zap.Int("count", len(tools)))
		for _, tool := range tools {
			if err := toolRegistry.RegisterExisting(tool); err != nil {
				logger.Warn("Failed to register tool", zap.String("tool", tool.Name()), zap.Error(err))
			} else {
				logger.Info("Tool registered successfully", zap.String("tool", tool.Name()))
			}
		}
		logger.Info("Cron tools registration completed")
	}

	// 创建网关服务器
	gatewayServer := gateway.NewServer(configFile, messageBus, channelMgr, sessionMgr, cronService)
	gatewayServer.SetHarnessService(harnessService)
	gatewayServer.SetMemoryCatalog(catalogService)
	gatewayServer.SetDevelopmentService(developmentService)
	gatewayServer.SetOuroborosService(ouroborosService)
	gatewayServer.SetTeamControlService(teamControlService)
	gatewayServer.SetWorkstationService(workstationService)
	if err := gatewayServer.Start(ctx); err != nil {
		logger.Warn("Failed to start gateway server", zap.Error(err))
	}
	defer func() { _ = gatewayServer.Stop() }()

	// 创建 AgentManager
	agentManager := agent.NewAgentManager(&agent.NewAgentManagerConfig{
		Bus:            messageBus,
		Provider:       provider,
		SessionMgr:     sessionMgr,
		Tools:          toolRegistry,
		DataDir:        workspaceDir, // 使用 workspace 作为数据目录
		ContextBuilder: contextBuilder,
		SkillsLoader:   skillsLoader,
		ChannelMgr:     channelMgr,
		Harness:        harnessService,
	})

	// 从配置设置 Agent 和绑定
	if err := agentManager.SetupFromConfig(configFile, contextBuilder); err != nil {
		logger.Fatal("Failed to setup agent manager", zap.Error(err))
	}

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动通道
	if err := channelMgr.Start(ctx); err != nil {
		logger.Error("Failed to start channels", zap.Error(err))
	}
	defer func() { _ = channelMgr.Stop() }()

	// 启动出站消息分发
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Outbound message dispatcher panicked",
					zap.Any("panic", r))
			}
		}()
		if err := channelMgr.DispatchOutbound(ctx); err != nil {
			logger.Error("Outbound message dispatcher exited with error", zap.Error(err))
		} else {
			logger.Debug("Outbound message dispatcher exited normally")
		}
	}()

	// 启动 AgentManager
	go func() {
		if err := agentManager.Start(ctx); err != nil {
			logger.Error("AgentManager error", zap.Error(err))
		}
	}()

	// 等待信号
	<-sigChan
	logger.Info("Received shutdown signal")

	// 停止 AgentManager
	if err := agentManager.Stop(); err != nil {
		logger.Error("Failed to stop agent manager", zap.Error(err))
	}

	logger.Info("goclaw agent stopped")
	return nil
}

// rejectPilotMaintenanceMode closes the startup/backup race. The backup
// command creates this lock before probing the local Gateway; a newly starting
// Gateway must therefore refuse to open any runtime store until the cold
// snapshot has finished.
func rejectPilotMaintenanceMode() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := home + "/.goclaw/pilot-maintenance.lock"
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return fmt.Errorf("inspect pilot maintenance lock: %w", err)
	case !info.Mode().IsRegular():
		return errors.New("pilot maintenance lock exists but is not a regular file")
	default:
		return fmt.Errorf(
			"pilot maintenance mode is active (%s); Gateway startup is blocked",
			path,
		)
	}
}

func recoverWorkstationLeases(ctx context.Context, service *workstation.Service) {
	recover := func() {
		report, err := service.RecoverExpiredLeases()
		if err != nil {
			logger.Error("Workstation lease recovery failed", zap.Error(err))
			return
		}
		if len(report.RequeuedTaskIDs) > 0 || len(report.FailedTaskIDs) > 0 {
			logger.Warn("Recovered expired workstation leases",
				zap.Int("requeued", len(report.RequeuedTaskIDs)),
				zap.Int("failed", len(report.FailedTaskIDs)))
		}
	}
	recover()
	interval := time.Duration(service.Config().LeaseDurationSeconds) * time.Second / 2
	if interval < 15*time.Second {
		interval = 15 * time.Second
	}
	if interval > 60*time.Second {
		interval = 60 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			recover()
		}
	}
}

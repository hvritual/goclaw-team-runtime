package workspace

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	workspacehttp "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/http"
	"google.golang.org/grpc"
)

type GovernanceOutboxDependencies struct {
	Sink             contract.OutboxSink
	Authorizer       contract.WorkspaceAccessAuthorizer
	EventPolicies    application.GovernanceEventPolicyProvider
	Memberships      contract.WorkspaceMembershipReader
	HTTPIdentity     contract.WorkspaceHTTPIdentityResolver
	HTTPUserIdentity func(*http.Request) (string, error)
	Now              func() time.Time
	NewClaimToken    func() string
	Jitter           func(int) time.Duration
	BatchSize        int
	PollInterval     time.Duration
}

type GovernanceOutbox struct {
	service   *application.OutboxService
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	interval  time.Duration
	started   atomic.Bool
	running   atomic.Bool
	closed    atomic.Bool
	closeOnce sync.Once
}

func NewSQLiteGovernanceOutbox(module *Module, config SqlitePersistenceConfig, dependencies GovernanceOutboxDependencies) (*GovernanceOutbox, error) {
	if module == nil || dependencies.Sink == nil || dependencies.Authorizer == nil || dependencies.EventPolicies == nil || dependencies.Memberships == nil ||
		dependencies.HTTPIdentity == nil || dependencies.HTTPUserIdentity == nil {
		return nil, contract.ErrGovernanceUnavailable
	}
	repository, err := persistence.NewGovernanceRepository(config, persistence.WithGovernanceEventPolicies(dependencies.EventPolicies))
	if err != nil {
		return nil, err
	}
	if dependencies.NewClaimToken == nil {
		dependencies.NewClaimToken = uuid.NewString
	}
	service, err := application.NewOutboxService(application.OutboxServiceConfig{
		Repository: repository,
		Sink:       dependencies.Sink, Authorizer: dependencies.Authorizer, EventPolicies: dependencies.EventPolicies,
		Now: dependencies.Now, NewClaimToken: dependencies.NewClaimToken,
		Jitter: dependencies.Jitter, BatchSize: dependencies.BatchSize,
	})
	if err != nil {
		return nil, err
	}
	interval := dependencies.PollInterval
	if interval <= 0 {
		interval = time.Second
	}
	workerContext, cancel := context.WithCancel(context.Background())
	outbox := &GovernanceOutbox{service: service, ctx: workerContext, cancel: cancel, done: make(chan struct{}), interval: interval}
	service.SetDispatcherRunningProbe(outbox.Running)
	module.extensions = append(module.extensions, &governanceExtension{handler: workspacehttp.NewGovernanceHandler(
		service, dependencies.HTTPIdentity, dependencies.HTTPUserIdentity, dependencies.Memberships,
	)})
	return outbox, nil
}

func (g *GovernanceOutbox) Start() {
	if g == nil || g.closed.Load() || !g.started.CompareAndSwap(false, true) {
		return
	}
	g.running.Store(true)
	go g.run()
}

func (g *GovernanceOutbox) run() {
	defer close(g.done)
	defer g.running.Store(false)
	_, _ = g.service.DispatchOnce(g.ctx)
	ticker := time.NewTicker(g.interval)
	defer ticker.Stop()
	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			_, _ = g.service.DispatchOnce(g.ctx)
		}
	}
}

func (g *GovernanceOutbox) Running() bool {
	return g != nil && g.running.Load()
}

func (g *GovernanceOutbox) ReadGovernanceDiagnostics(ctx context.Context, workspaceID string) (contract.OutboxDiagnostics, error) {
	if g == nil || g.service == nil {
		return contract.OutboxDiagnostics{}, contract.ErrGovernanceUnavailable
	}
	return g.service.ReadGovernanceDiagnostics(ctx, workspaceID)
}

func (g *GovernanceOutbox) Replay(ctx context.Context, identity contract.OutboxRowIdentity) error {
	if g == nil || g.service == nil {
		return contract.ErrGovernanceUnavailable
	}
	return g.service.Replay(ctx, identity)
}

func (g *GovernanceOutbox) Ready(ctx context.Context) error {
	if !g.Running() {
		return contract.ErrGovernanceUnavailable
	}
	_, err := g.ReadGovernanceDiagnostics(ctx, "__governance_readiness__")
	return err
}

func (g *GovernanceOutbox) Close() error {
	if g == nil {
		return nil
	}
	g.closeOnce.Do(func() {
		g.closed.Store(true)
		g.cancel()
		if g.started.Load() {
			<-g.done
		}
	})
	return nil
}

type governanceExtension struct {
	handler *workspacehttp.GovernanceHandler
}

func (e *governanceExtension) RegisterHTTP(server *kratoshttp.Server) { e.handler.Register(server) }
func (*governanceExtension) RegisterGRPC(grpc.ServiceRegistrar)       {}

var _ contract.GovernanceDiagnosticsReader = (*GovernanceOutbox)(nil)

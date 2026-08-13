package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v3"
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/workspace"
)

// Config defines the standalone backend process identity and listen addresses.
type Config struct {
	Name                  string
	Version               string
	HTTPAddress           string
	GRPCAddress           string
	SQLitePath            string
	WorkspaceDependencies workspace.WorkspaceServiceDependencies
}

// Validate rejects incomplete process identity and malformed TCP addresses.
func (c Config) Validate() error {
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("service name is required")
	}
	if strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("service version is required")
	}
	if err := validateTCPAddress("HTTP", c.HTTPAddress); err != nil {
		return err
	}
	if err := validateTCPAddress("gRPC", c.GRPCAddress); err != nil {
		return err
	}
	if strings.TrimSpace(c.SQLitePath) == "" {
		return fmt.Errorf("SQLite path is required")
	}
	return nil
}

func validateTCPAddress(label, address string) error {
	trimmed := strings.TrimSpace(address)
	if trimmed == "" {
		return fmt.Errorf("%s address is required", label)
	}
	_, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		return fmt.Errorf("parse %s address %q: %w", label, address, err)
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 0 || portNumber > 65535 {
		return fmt.Errorf("parse %s address %q: port must be between 0 and 65535", label, address)
	}
	return nil
}

// Runtime owns the Kratos application and its two transports.
type Runtime struct {
	application *Application
	app         *kratos.App
	httpServer  *kratoshttp.Server
	grpcServer  *kratosgrpc.Server
	db          *sql.DB
	closeOnce   sync.Once
	closeErr    error
}

// NewRuntime creates the shared HTTP and gRPC servers and registers all modules.
func NewRuntime(config Config, logger *slog.Logger) (*Runtime, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}

	db, application, err := newSQLiteApplication(context.Background(), config)
	if err != nil {
		return nil, err
	}
	httpServer := kratoshttp.NewServer(kratoshttp.Address(config.HTTPAddress))
	grpcServer := kratosgrpc.NewServer(kratosgrpc.Address(config.GRPCAddress))
	application.RegisterHTTP(httpServer)
	application.RegisterGRPC(grpcServer)
	registerHealthRoutes(httpServer, db)

	app := kratos.New(
		kratos.Name(config.Name),
		kratos.Version(config.Version),
		kratos.Logger(logger),
		kratos.StopTimeout(5*time.Second),
		kratos.Server(httpServer, grpcServer),
	)
	return &Runtime{
		application: application,
		app:         app,
		httpServer:  httpServer,
		grpcServer:  grpcServer,
		db:          db,
	}, nil
}

func registerHealthRoutes(server *kratoshttp.Server, db *sql.DB) {
	router := server.Route("/")
	router.GET("/healthz", func(ctx kratoshttp.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	router.GET("/readyz", func(ctx kratoshttp.Context) error {
		if err := db.PingContext(ctx.Request().Context()); err != nil {
			return ctx.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
		}
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
}

// Run starts both transports and blocks until shutdown or an error.
func (r *Runtime) Run() error {
	defer r.Close()
	return r.app.Run()
}

// Stop requests graceful shutdown of both transports.
func (r *Runtime) Stop() error {
	stopErr := r.app.Stop()
	closeErr := r.Close()
	if stopErr != nil {
		return stopErr
	}
	return closeErr
}

// Close releases the product database. It is safe to call more than once.
func (r *Runtime) Close() error {
	r.closeOnce.Do(func() {
		if r.db != nil {
			r.closeErr = r.db.Close()
		}
	})
	return r.closeErr
}

// Database exposes the owned product database for composition and lifecycle tests.
func (r *Runtime) Database() *sql.DB { return r.db }

// Application returns the assembled bounded contexts.
func (r *Runtime) Application() *Application {
	return r.application
}

// HTTPServer exposes the registered server for transport tests and embedding.
func (r *Runtime) HTTPServer() *kratoshttp.Server {
	return r.httpServer
}

// GRPCServer exposes the registered server for transport tests and embedding.
func (r *Runtime) GRPCServer() *kratosgrpc.Server {
	return r.grpcServer
}

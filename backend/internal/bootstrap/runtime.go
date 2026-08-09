package bootstrap

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3"
	kratosgrpc "github.com/go-kratos/kratos/v3/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

// Config defines the standalone backend process identity and listen addresses.
type Config struct {
	Name        string
	Version     string
	HTTPAddress string
	GRPCAddress string
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
}

// NewRuntime creates the shared HTTP and gRPC servers and registers all modules.
func NewRuntime(config Config, logger *slog.Logger) (*Runtime, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}

	httpServer := kratoshttp.NewServer(kratoshttp.Address(config.HTTPAddress))
	grpcServer := kratosgrpc.NewServer(kratosgrpc.Address(config.GRPCAddress))
	application := NewApplication()
	application.RegisterHTTP(httpServer)
	application.RegisterGRPC(grpcServer)
	registerHealthRoutes(httpServer)

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
	}, nil
}

func registerHealthRoutes(server *kratoshttp.Server) {
	router := server.Route("/")
	router.GET("/healthz", func(ctx kratoshttp.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	router.GET("/readyz", func(ctx kratoshttp.Context) error {
		return ctx.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
}

// Run starts both transports and blocks until shutdown or an error.
func (r *Runtime) Run() error {
	return r.app.Run()
}

// Stop requests graceful shutdown of both transports.
func (r *Runtime) Stop() error {
	return r.app.Stop()
}

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

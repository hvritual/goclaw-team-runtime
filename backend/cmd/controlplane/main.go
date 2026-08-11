package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	controlplane "github.com/hvritual/workspace/internal/controlplane"
)

func main() {
	address := os.Getenv("CONTROLPLANE_HTTP_ADDRESS")
	if address == "" {
		address = ":8080"
	}

	databasePath := os.Getenv("CONTROLPLANE_SQLITE_PATH")
	if databasePath == "" {
		databasePath = "controlplane.db"
	}
	repository, err := controlplane.OpenSQLite(context.Background(), databasePath)
	if err != nil {
		slog.Error("open control plane repository", "error", err)
		os.Exit(1)
	}
	defer repository.Close()
	service, err := controlplane.NewService(repository, nil)
	if err != nil {
		slog.Error("create control plane service", "error", err)
		os.Exit(1)
	}
	store, err := controlplane.KernelStoreFrom(repository)
	if err != nil {
		slog.Error("create kernel store", "error", err)
		os.Exit(1)
	}
	kernel, err := controlplane.NewDeliveryKernel(store, nil, service.Authorize)
	if err != nil {
		slog.Error("create delivery kernel", "error", err)
		os.Exit(1)
	}
	flows, err := controlplane.NewP2Flows(kernel)
	if err != nil {
		slog.Error("create P2 flows", "error", err)
		os.Exit(1)
	}
	api, err := controlplane.NewHTTPAPI(kernel, flows, func(request *http.Request) (controlplane.Actor, error) {
		if os.Getenv("CONTROLPLANE_ALLOW_HEADER_IDENTITY") != "true" {
			return controlplane.Actor{}, controlplane.ErrDenied
		}
		kind := controlplane.ActorKind(request.Header.Get("X-Actor-Kind"))
		return controlplane.Actor{ID: request.Header.Get("X-Actor-ID"), WorkspaceID: request.Header.Get("X-Workspace-ID"), Kind: kind}, nil
	})
	if err != nil {
		slog.Error("create HTTP API", "error", err)
		os.Exit(1)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown control plane", "error", err)
		}
	}()

	slog.Info("control plane listening", "address", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("serve control plane", "error", err)
		os.Exit(1)
	}
}

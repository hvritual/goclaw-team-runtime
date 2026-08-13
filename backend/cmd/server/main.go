package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/hvritual/workspace/internal/bootstrap"
)

const defaultVersion = "dev"

func main() {
	config, err := parseConfig(os.Args[1:], os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	runtime, err := bootstrap.NewRuntime(config, logger)
	if err != nil {
		logger.Error("configure backend", "error", err)
		os.Exit(1)
	}
	logger.Info(
		"backend starting",
		"name", config.Name,
		"version", config.Version,
		"http", config.HTTPAddress,
		"grpc", config.GRPCAddress,
	)
	if err := runtime.Run(); err != nil {
		logger.Error("backend stopped", "error", err)
		os.Exit(1)
	}
}

func parseConfig(arguments []string, output io.Writer) (bootstrap.Config, error) {
	flags := flag.NewFlagSet("backend-server", flag.ContinueOnError)
	flags.SetOutput(output)
	config := bootstrap.Config{}
	flags.StringVar(&config.Name, "name", "hvritual-workspace-backend", "service name")
	flags.StringVar(&config.Version, "version", defaultVersion, "service version")
	flags.StringVar(&config.HTTPAddress, "http-addr", "127.0.0.1:8000", "HTTP listen address")
	flags.StringVar(&config.GRPCAddress, "grpc-addr", "127.0.0.1:9000", "gRPC listen address")
	flags.StringVar(&config.SQLitePath, "sqlite-path", "data/multica-canonical.db", "Canonical product SQLite path")
	config.WorkspaceDependencies = bootstrap.FailClosedWorkspaceDependencies()
	if err := flags.Parse(arguments); err != nil {
		return bootstrap.Config{}, err
	}
	if flags.NArg() != 0 {
		return bootstrap.Config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if err := config.Validate(); err != nil {
		return bootstrap.Config{}, err
	}
	return config, nil
}

package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/multica-ai/multica/server/internal/sqlitelocal"
)

func main() {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production") {
		log.Fatal("SQLite local server cannot run with APP_ENV=production")
	}
	databasePath := strings.TrimSpace(os.Getenv("SQLITE_DATABASE_PATH"))
	if databasePath == "" {
		databasePath = "../data/multica-local.db"
	}
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	listenAddress := strings.TrimSpace(os.Getenv("SQLITE_LISTEN_ADDRESS"))
	if listenAddress == "" {
		listenAddress = "127.0.0.1"
	}

	app, err := sqlitelocal.Open(databasePath, sqlitelocal.Options{
		VerificationCode:       os.Getenv("MULTICA_DEV_VERIFICATION_CODE"),
		FrontendOrigin:         os.Getenv("FRONTEND_ORIGIN"),
		KnowledgeDatabasePath:  strings.TrimSpace(os.Getenv("MULTICA_KNOWLEDGE_SQLITE_PATH")),
		DisableKnowledge:       strings.EqualFold(strings.TrimSpace(os.Getenv("MULTICA_KNOWLEDGE_ENABLED")), "false"),
		PublicURL:              strings.TrimSpace(os.Getenv("MULTICA_PUBLIC_URL")),
		MCPAuthorizationServers: splitCommaSeparated(
			os.Getenv("MULTICA_MCP_AUTHORIZATION_SERVERS"),
		),
	})
	if err != nil {
		log.Fatalf("open SQLite local server: %v", err)
	}
	defer app.Close()

	server := &http.Server{
		Addr:              listenAddress + ":" + port,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("SQLite local server shutdown: %v", err)
		}
	}()

	log.Printf("Multica SQLite local server listening on http://%s:%s (database: %s)", listenAddress, port, databasePath)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("serve SQLite local API: %v", err)
	}
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}

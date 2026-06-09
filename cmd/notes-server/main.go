package main

import (
	"context"
	"fmt"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/version"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "notes-server configuration error:", err)
		os.Exit(1)
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: config.LogLevel}))
	slog.SetDefault(log)
	log.Info("starting notes server",
		"app", version.AppName,
		"version", version.Version,
		"revision", version.Revision,
		"build", version.BuildStamp,
		"listen", config.ListenAddress,
		"auth_mode", config.AuthMode,
	)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 15*time.Second)
	app, err := newApplication(startupCtx, config, log)
	startupCancel()
	if err != nil {
		log.Error("failed to initialize notes server", "error", err)
		os.Exit(1)
	}
	defer app.close()

	listener, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		log.Error("failed to listen", "error", err)
		os.Exit(1)
	}
	log.Info("notes server listening", "address", listener.Addr().String())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := app.serve(ctx, listener, config.ShutdownPeriod); err != nil {
		log.Error("notes server stopped with error", "error", err)
		os.Exit(1)
	}
	log.Info("notes server stopped")
}

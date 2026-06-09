package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListenAddress  = "127.0.0.1:8080"
	defaultAuthMode       = "jwt"
	defaultShutdownPeriod = 10 * time.Second
	defaultMaxConnections = 10
)

type serverConfig struct {
	ListenAddress  string
	DatabaseURL    string
	AuthMode       string
	DevToken       string
	DevUserID      int64
	DevUserEmail   string
	DevDisplayName string
	LogLevel       slog.Level
	MaxConnections int32
	ShutdownPeriod time.Duration
}

func loadConfig() (serverConfig, error) {
	databaseURL, err := databaseURLFromEnv()
	if err != nil {
		return serverConfig{}, err
	}

	listenAddress := envOrDefault("NOTES_LISTEN_ADDRESS", defaultListenAddress)
	if _, _, err := net.SplitHostPort(listenAddress); err != nil {
		return serverConfig{}, fmt.Errorf("NOTES_LISTEN_ADDRESS must be host:port: %w", err)
	}

	authMode := strings.ToLower(strings.TrimSpace(envOrDefault("NOTES_AUTH_MODE", defaultAuthMode)))
	if authMode != "jwt" && authMode != "dev" {
		return serverConfig{}, fmt.Errorf("NOTES_AUTH_MODE must be jwt or dev")
	}

	devUserID, err := envInt64("NOTES_DEV_USER_ID", 1)
	if err != nil {
		return serverConfig{}, err
	}
	maxConnections, err := envInt64("NOTES_DB_MAX_CONNECTIONS", defaultMaxConnections)
	if err != nil || maxConnections < 1 || maxConnections > 1000 {
		return serverConfig{}, fmt.Errorf("NOTES_DB_MAX_CONNECTIONS must be between 1 and 1000")
	}
	shutdownSeconds, err := envInt64("NOTES_SHUTDOWN_TIMEOUT_SECONDS", int64(defaultShutdownPeriod/time.Second))
	if err != nil || shutdownSeconds < 1 || shutdownSeconds > 300 {
		return serverConfig{}, fmt.Errorf("NOTES_SHUTDOWN_TIMEOUT_SECONDS must be between 1 and 300")
	}
	logLevel, err := parseLogLevel(envOrDefault("LOG_LEVEL", "info"))
	if err != nil {
		return serverConfig{}, err
	}

	config := serverConfig{
		ListenAddress:  listenAddress,
		DatabaseURL:    databaseURL,
		AuthMode:       authMode,
		DevToken:       os.Getenv("NOTES_DEV_TOKEN"),
		DevUserID:      devUserID,
		DevUserEmail:   envOrDefault("NOTES_DEV_USER_EMAIL", "dev@localhost"),
		DevDisplayName: envOrDefault("NOTES_DEV_USER_NAME", "Local Notes User"),
		LogLevel:       logLevel,
		MaxConnections: int32(maxConnections),
		ShutdownPeriod: time.Duration(shutdownSeconds) * time.Second,
	}
	if config.AuthMode == "dev" && config.DevToken == "" {
		return serverConfig{}, fmt.Errorf("NOTES_DEV_TOKEN is required when NOTES_AUTH_MODE=dev")
	}
	return config, nil
}

func databaseURLFromEnv() (string, error) {
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
			return "", fmt.Errorf("DATABASE_URL is invalid")
		}
		return value, nil
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("DATABASE_URL or DB_PASSWORD is required")
	}
	host := envOrDefault("DB_HOST", "127.0.0.1")
	port := envOrDefault("DB_PORT", "5432")
	name := envOrDefault("DB_NAME", "go_mcp_notes")
	user := envOrDefault("DB_USER", "go_mcp_notes")
	sslMode := envOrDefault("DB_SSL_MODE", "prefer")
	result := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(host, port),
		Path:     name,
		RawQuery: url.Values{"sslmode": []string{sslMode}}.Encode(),
		User:     url.UserPassword(user, password),
	}
	return result.String(), nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt64(name string, fallback int64) (int64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return value, nil
}

func parseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
}

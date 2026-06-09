package main

import (
	"strings"
	"testing"
)

func TestLoadConfigDevMode(t *testing.T) {
	setMinimalConfigEnv(t)
	t.Setenv("NOTES_AUTH_MODE", "dev")
	t.Setenv("NOTES_DEV_TOKEN", "test-token")
	t.Setenv("NOTES_DEV_USER_ID", "42")

	config, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.ListenAddress != defaultListenAddress || config.DevUserID != 42 || config.AuthMode != "dev" {
		t.Fatalf("unexpected config: %#v", config)
	}
}

func TestLoadConfigRejectsUnsafeValues(t *testing.T) {
	setMinimalConfigEnv(t)
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "auth mode", key: "NOTES_AUTH_MODE", value: "none"},
		{name: "listen address", key: "NOTES_LISTEN_ADDRESS", value: "8080"},
		{name: "pool size", key: "NOTES_DB_MAX_CONNECTIONS", value: "0"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setMinimalConfigEnv(t)
			t.Setenv(test.key, test.value)
			if _, err := loadConfig(); err == nil {
				t.Fatal("loadConfig() succeeded, want error")
			}
		})
	}
}

func TestDatabaseURLFromComponentsEscapesCredentials(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "notes")
	t.Setenv("DB_USER", "notes-user")
	t.Setenv("DB_PASSWORD", "p@ss:word")
	t.Setenv("DB_SSL_MODE", "disable")
	value, err := databaseURLFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(value, "notes-user:p%40ss%3Aword@") {
		t.Fatalf("credentials were not escaped: %s", value)
	}
}

func TestDatabaseURLRejectsNonPostgresScheme(t *testing.T) {
	t.Setenv("DATABASE_URL", "mysql://notes:secret@127.0.0.1/notes")
	if _, err := databaseURLFromEnv(); err == nil {
		t.Fatal("databaseURLFromEnv() accepted a non-PostgreSQL URL")
	}
}

func setMinimalConfigEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://notes:secret@127.0.0.1:5432/notes?sslmode=disable")
	t.Setenv("NOTES_AUTH_MODE", "jwt")
	t.Setenv("NOTES_DEV_TOKEN", "")
	t.Setenv("NOTES_LISTEN_ADDRESS", "")
	t.Setenv("NOTES_DB_MAX_CONNECTIONS", "")
	t.Setenv("NOTES_SHUTDOWN_TIMEOUT_SECONDS", "")
	t.Setenv("LOG_LEVEL", "")
}

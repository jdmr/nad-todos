package main

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL       string
	ListenAddr        string
	WebAuthnRPID      string
	WebAuthnRPOrigin  string
	WebAuthnRPName    string
	SessionCookieName string
	SessionDuration   time.Duration
}

func LoadConfig() Config {
	return Config{
		DatabaseURL:       envOr("DATABASE_URL", "postgres://localhost:5432/todos"),
		ListenAddr:        envOr("LISTEN_ADDR", ":8080"),
		WebAuthnRPID:      envOr("WEBAUTHN_RP_ID", "localhost"),
		WebAuthnRPOrigin:  envOr("WEBAUTHN_RP_ORIGIN", "http://localhost:5173"),
		WebAuthnRPName:    envOr("WEBAUTHN_RP_NAME", "Todos"),
		SessionCookieName: envOr("SESSION_COOKIE_NAME", "todos_session"),
		SessionDuration:   envDuration("SESSION_DURATION", 8*time.Hour),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

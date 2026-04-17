package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	cfg := LoadConfig()

	log.Printf("Connecting to database...")
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	wa, err := NewWebAuthn(cfg)
	if err != nil {
		log.Fatalf("Failed to init WebAuthn: %v", err)
	}

	todoStore := NewPostgresTodoStore(pool)
	userStore := NewPostgresUserStore(pool)
	credStore := NewPostgresCredentialStore(pool)
	sessStore := NewPostgresSessionStore(pool)
	inviteStore := NewPostgresInvitationStore(pool)

	sessions := NewSessionService(sessStore, cfg.SessionDuration)
	cache := NewChallengeCache()

	todoHdl := NewTodoHandler(todoStore)
	authHdl := NewAuthHandler(cfg, wa, cache, userStore, credStore, inviteStore, sessions)
	adminHdl := NewAdminHandler(userStore, inviteStore)

	if err := MaybeBootstrapAdmin(context.Background(), cfg, userStore, inviteStore); err != nil {
		log.Fatalf("Bootstrap failed: %v", err)
	}

	authMW := RequireSession(sessions, cfg.SessionCookieName)
	adminMW := RequireRole(RoleAdmin)

	mux := http.NewServeMux()

	// Public auth routes
	mux.HandleFunc("GET /api/v1/auth/invitations/{token}", authHdl.GetInvitation)
	mux.HandleFunc("POST /api/v1/auth/register/options", authHdl.RegisterOptions)
	mux.HandleFunc("POST /api/v1/auth/register/verify", authHdl.RegisterVerify)
	mux.HandleFunc("POST /api/v1/auth/login/options", authHdl.LoginOptions)
	mux.HandleFunc("POST /api/v1/auth/login/verify", authHdl.LoginVerify)

	// Authenticated routes
	authed := http.NewServeMux()
	authed.HandleFunc("GET /api/v1/auth/me", authHdl.Me)
	authed.HandleFunc("POST /api/v1/auth/logout", authHdl.Logout)
	authed.HandleFunc("GET /api/v1/auth/sessions", authHdl.ListSessions)
	authed.HandleFunc("POST /api/v1/auth/sessions/revoke", authHdl.RevokeSession)
	authed.HandleFunc("GET /api/v1/auth/credentials", authHdl.ListCredentials)
	authed.HandleFunc("POST /api/v1/auth/credentials/options", authHdl.AddCredentialOptions)
	authed.HandleFunc("POST /api/v1/auth/credentials/verify", authHdl.AddCredentialVerify)
	authed.HandleFunc("DELETE /api/v1/auth/credentials/{credID}", authHdl.DeleteCredential)

	authed.HandleFunc("GET /api/v1/todos", todoHdl.ListTodos)
	authed.HandleFunc("POST /api/v1/todos", todoHdl.CreateTodo)
	authed.HandleFunc("GET /api/v1/todos/{todoID}", todoHdl.GetTodo)
	authed.HandleFunc("PUT /api/v1/todos/{todoID}", todoHdl.UpdateTodo)
	authed.HandleFunc("DELETE /api/v1/todos/{todoID}", todoHdl.DeleteTodo)

	// Admin routes
	authed.Handle("GET /api/v1/admin/users", adminMW(http.HandlerFunc(adminHdl.ListUsers)))
	authed.Handle("PUT /api/v1/admin/users/{userID}/role", adminMW(http.HandlerFunc(adminHdl.UpdateUserRole)))
	authed.Handle("POST /api/v1/admin/invitations", adminMW(http.HandlerFunc(adminHdl.CreateInvitation)))
	authed.Handle("GET /api/v1/admin/invitations", adminMW(http.HandlerFunc(adminHdl.ListInvitations)))
	authed.Handle("DELETE /api/v1/admin/invitations/{invitationID}", adminMW(http.HandlerFunc(adminHdl.RevokeInvitation)))

	mux.Handle("/", authMW(authed))

	log.Printf("API server listening on %s (RP %s)", cfg.ListenAddr, cfg.WebAuthnRPID)
	log.Fatal(http.ListenAndServe(cfg.ListenAddr, mux))
}

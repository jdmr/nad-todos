package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type AuthHandler struct {
	cfg           Config
	wa            *webauthn.WebAuthn
	cache         *ChallengeCache
	users         UserStore
	creds         CredentialStore
	invites       InvitationStore
	sessions      *SessionService
}

func NewAuthHandler(cfg Config, wa *webauthn.WebAuthn, cache *ChallengeCache,
	users UserStore, creds CredentialStore, invites InvitationStore, sessions *SessionService) *AuthHandler {
	return &AuthHandler{cfg: cfg, wa: wa, cache: cache, users: users, creds: creds, invites: invites, sessions: sessions}
}

// --- Invitation lookup (public)

func (h *AuthHandler) GetInvitation(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	inv, err := h.invites.FindByToken(r.Context(), token)
	if err != nil || !inv.IsValid() {
		http.Error(w, "Invalid or expired invitation", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"email":        inv.Email,
		"default_role": inv.DefaultRole,
		"is_bootstrap": inv.IsBootstrap,
	})
}

// --- Registration

type registerOptionsReq struct {
	InvitationToken string `json:"invitation_token"`
	Name            string `json:"name"`
	Email           string `json:"email"` // honored only for bootstrap invitations
	DeviceName      string `json:"device_name"`
}

func (h *AuthHandler) RegisterOptions(w http.ResponseWriter, r *http.Request) {
	var req registerOptionsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Name required", http.StatusBadRequest)
		return
	}

	inv, err := h.invites.FindByToken(r.Context(), req.InvitationToken)
	if err != nil || !inv.IsValid() {
		http.Error(w, "Invalid or expired invitation", http.StatusForbidden)
		return
	}

	// Bootstrap invitations let the registrant pick their own email. All other
	// invitations are email-locked to what the admin specified.
	email := inv.Email
	if inv.IsBootstrap {
		if req.Email == "" {
			http.Error(w, "Email required", http.StatusBadRequest)
			return
		}
		email = req.Email
	}

	if existing, _ := h.users.FindByEmail(r.Context(), email); existing != nil {
		http.Error(w, "User already registered", http.StatusConflict)
		return
	}

	user := &User{
		ID:     uuid.New(),
		Email:  email,
		Name:   req.Name,
		Role:   inv.DefaultRole,
		Status: UserStatusActive,
	}

	creation, sessionData, err := h.wa.BeginRegistration(user)
	if err != nil {
		log.Printf("BeginRegistration: %v", err)
		http.Error(w, "Failed to start registration", http.StatusInternalServerError)
		return
	}

	challengeID := uuid.NewString()
	h.cache.Put(challengeID, sessionData, map[string]string{
		"invitation_token": req.InvitationToken,
		"user_id":          user.ID.String(),
		"name":             req.Name,
		"email":            email,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"challenge_id": challengeID,
		"challenge":    creation,
	})
}

type registerVerifyReq struct {
	ChallengeID string          `json:"challenge_id"`
	DeviceName  string          `json:"device_name"`
	Credential  json.RawMessage `json:"credential"`
}

func (h *AuthHandler) RegisterVerify(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	var req registerVerifyReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sessionData, state, err := h.cache.Take(req.ChallengeID)
	if err != nil {
		http.Error(w, "Challenge expired", http.StatusBadRequest)
		return
	}

	inv, err := h.invites.FindByToken(r.Context(), state["invitation_token"])
	if err != nil || !inv.IsValid() {
		http.Error(w, "Invalid invitation", http.StatusForbidden)
		return
	}

	userID, err := uuid.Parse(state["user_id"])
	if err != nil {
		http.Error(w, "Bad challenge state", http.StatusInternalServerError)
		return
	}

	email := state["email"]
	if email == "" {
		email = inv.Email
	}

	user := &User{
		ID:     userID,
		Email:  email,
		Name:   state["name"],
		Role:   inv.DefaultRole,
		Status: UserStatusActive,
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(req.Credential))
	if err != nil {
		http.Error(w, "Invalid credential: "+err.Error(), http.StatusBadRequest)
		return
	}
	cred, err := h.wa.CreateCredential(user, *sessionData, parsed)
	if err != nil {
		http.Error(w, "Verification failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	user.InvitedBy = inv.InvitedBy
	if err := h.users.Create(r.Context(), user); err != nil {
		log.Printf("create user: %v", err)
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	stored := CredentialFromWebAuthn(user.ID, cred, req.DeviceName)
	if err := h.creds.Create(r.Context(), stored); err != nil {
		log.Printf("create credential: %v", err)
		http.Error(w, "Failed to store credential", http.StatusInternalServerError)
		return
	}

	if err := h.invites.MarkUsed(r.Context(), inv.ID, user.ID); err != nil {
		log.Printf("mark invitation used: %v", err)
	}

	sess, err := h.sessions.Create(r.Context(), user, clientIP(r), r.UserAgent(), req.DeviceName)
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	h.setSessionCookie(w, sess)

	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"role":    user.Role,
	})
}

// --- Login

type loginOptionsReq struct {
	Email string `json:"email"`
}

func (h *AuthHandler) LoginOptions(w http.ResponseWriter, r *http.Request) {
	var req loginOptionsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	user, err := h.users.FindByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	creds, err := h.creds.FindByUserID(r.Context(), user.ID)
	if err != nil || len(creds) == 0 {
		http.Error(w, "No credentials", http.StatusNotFound)
		return
	}
	user.Credentials = creds

	assertion, sessionData, err := h.wa.BeginLogin(user)
	if err != nil {
		http.Error(w, "Failed to start login", http.StatusInternalServerError)
		return
	}

	challengeID := uuid.NewString()
	h.cache.Put(challengeID, sessionData, map[string]string{"email": user.Email})

	writeJSON(w, http.StatusOK, map[string]any{
		"challenge_id": challengeID,
		"challenge":    assertion,
	})
}

type loginVerifyReq struct {
	ChallengeID string          `json:"challenge_id"`
	Credential  json.RawMessage `json:"credential"`
}

func (h *AuthHandler) LoginVerify(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	var req loginVerifyReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sessionData, state, err := h.cache.Take(req.ChallengeID)
	if err != nil {
		http.Error(w, "Challenge expired", http.StatusBadRequest)
		return
	}

	user, err := h.users.FindByEmail(r.Context(), state["email"])
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	creds, err := h.creds.FindByUserID(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Failed to load credentials", http.StatusInternalServerError)
		return
	}
	user.Credentials = creds

	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(req.Credential))
	if err != nil {
		http.Error(w, "Invalid credential: "+err.Error(), http.StatusBadRequest)
		return
	}
	verified, err := h.wa.ValidateLogin(user, *sessionData, parsed)
	if err != nil {
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	if err := h.creds.UpdateCounter(r.Context(), verified.ID, verified.Authenticator.SignCount); err != nil {
		log.Printf("update counter: %v", err)
	}
	if err := h.creds.UpdateLastUsed(r.Context(), verified.ID); err != nil {
		log.Printf("update last used: %v", err)
	}

	sess, err := h.sessions.Create(r.Context(), user, clientIP(r), r.UserAgent(), "")
	if err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}
	h.setSessionCookie(w, sess)

	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"role":    user.Role,
	})
}

// --- Session lifecycle

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	user, err := h.users.FindByID(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": user.ID,
		"email":   user.Email,
		"name":    user.Name,
		"role":    user.Role,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	if err := h.sessions.Revoke(r.Context(), sess.ID, &sess.UserID); err != nil {
		log.Printf("revoke session: %v", err)
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	sessions, err := h.sessions.ListByUser(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, "Failed to list sessions", http.StatusInternalServerError)
		return
	}
	type item struct {
		ID             string    `json:"id"`
		DeviceName     string    `json:"device_name"`
		IPAddress      string    `json:"ip_address"`
		CreatedAt      time.Time `json:"created_at"`
		LastActivityAt time.Time `json:"last_activity_at"`
		IsCurrent      bool      `json:"is_current"`
	}
	out := make([]item, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, item{
			ID: s.ID, DeviceName: s.DeviceName, IPAddress: s.IPAddress,
			CreatedAt: s.CreatedAt, LastActivityAt: s.LastActivityAt,
			IsCurrent: s.ID == sess.ID,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": out})
}

type revokeSessionReq struct {
	SessionID string `json:"session_id"`
}

func (h *AuthHandler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	var req revokeSessionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	target, err := h.sessions.store.FindByID(r.Context(), req.SessionID)
	if err != nil || target.UserID != sess.UserID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err := h.sessions.Revoke(r.Context(), req.SessionID, &sess.UserID); err != nil {
		http.Error(w, "Failed to revoke", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Credentials self-management

func (h *AuthHandler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	creds, err := h.creds.FindByUserID(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, "Failed to list credentials", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"credentials": creds})
}

type addCredentialOptionsReq struct {
	DeviceName string `json:"device_name"`
}

func (h *AuthHandler) AddCredentialOptions(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	user, err := h.users.FindByID(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	existing, _ := h.creds.FindByUserID(r.Context(), user.ID)
	user.Credentials = existing

	creation, sessionData, err := h.wa.BeginRegistration(user)
	if err != nil {
		http.Error(w, "Failed to start registration", http.StatusInternalServerError)
		return
	}
	challengeID := uuid.NewString()
	h.cache.Put(challengeID, sessionData, map[string]string{
		"user_id": user.ID.String(),
		"mode":    "add_credential",
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"challenge_id": challengeID,
		"challenge":    creation,
	})
}

func (h *AuthHandler) AddCredentialVerify(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	var req registerVerifyReq
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	sessionData, state, err := h.cache.Take(req.ChallengeID)
	if err != nil || state["mode"] != "add_credential" {
		http.Error(w, "Challenge expired", http.StatusBadRequest)
		return
	}
	if state["user_id"] != sess.UserID.String() {
		http.Error(w, "Challenge mismatch", http.StatusForbidden)
		return
	}

	user, err := h.users.FindByID(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(req.Credential))
	if err != nil {
		http.Error(w, "Invalid credential: "+err.Error(), http.StatusBadRequest)
		return
	}
	cred, err := h.wa.CreateCredential(user, *sessionData, parsed)
	if err != nil {
		http.Error(w, "Verification failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	stored := CredentialFromWebAuthn(user.ID, cred, req.DeviceName)
	if err := h.creds.Create(r.Context(), stored); err != nil {
		http.Error(w, "Failed to store credential", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, stored)
}

func (h *AuthHandler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("credID"))
	if err != nil {
		http.Error(w, "Invalid credential ID", http.StatusBadRequest)
		return
	}
	creds, err := h.creds.FindByUserID(r.Context(), sess.UserID)
	if err != nil {
		http.Error(w, "Failed", http.StatusInternalServerError)
		return
	}
	if len(creds) <= 1 {
		http.Error(w, "Cannot delete the last credential", http.StatusBadRequest)
		return
	}
	if err := h.creds.Delete(r.Context(), id, sess.UserID); err != nil {
		if errors.Is(err, ErrCredentialNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Cookie helpers

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, sess *Session) {
	secure := strings.HasPrefix(h.cfg.WebAuthnRPOrigin, "https://")
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    sess.ID,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cfg.SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode json: %v", err)
	}
}

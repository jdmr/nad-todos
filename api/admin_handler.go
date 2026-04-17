package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const invitationDuration = 7 * 24 * time.Hour

type AdminHandler struct {
	users   UserStore
	invites InvitationStore
}

func NewAdminHandler(users UserStore, invites InvitationStore) *AdminHandler {
	return &AdminHandler{users: users, invites: invites}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.users.List(r.Context())
	if err != nil {
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []User{}
	}
	writeJSON(w, http.StatusOK, users)
}

type updateRoleReq struct {
	Role Role `json:"role"`
}

func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("userID"))
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	var req updateRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if !req.Role.IsValid() {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}

	// Prevent demoting the last admin.
	if req.Role == RoleUser {
		target, err := h.users.FindByID(r.Context(), id)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		if target.Role == RoleAdmin {
			adminCount, err := h.users.CountAdmins(r.Context())
			if err != nil {
				http.Error(w, "Failed", http.StatusInternalServerError)
				return
			}
			if adminCount <= 1 {
				http.Error(w, "Cannot demote the last admin", http.StatusBadRequest)
				return
			}
		}
	}

	if err := h.users.UpdateRole(r.Context(), id, req.Role); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed", http.StatusInternalServerError)
		return
	}
	_ = sess // kept for future audit logging
	w.WriteHeader(http.StatusNoContent)
}

type createInvitationReq struct {
	Email string `json:"email"`
	Role  Role   `json:"role"`
}

func (h *AdminHandler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	var req createInvitationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	if req.Email == "" {
		http.Error(w, "Email required", http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = RoleUser
	}
	if !req.Role.IsValid() {
		http.Error(w, "Invalid role", http.StatusBadRequest)
		return
	}
	inv := &Invitation{
		Email:       req.Email,
		Token:       generateInvitationToken(),
		DefaultRole: req.Role,
		InvitedBy:   &sess.UserID,
		Status:      InvitationPending,
		ExpiresAt:   time.Now().Add(invitationDuration),
	}
	if err := h.invites.Create(r.Context(), inv); err != nil {
		http.Error(w, "Failed to create invitation", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":           inv.ID,
		"email":        inv.Email,
		"token":        inv.Token,
		"default_role": inv.DefaultRole,
		"expires_at":   inv.ExpiresAt,
	})
}

func (h *AdminHandler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	invitations, err := h.invites.List(r.Context())
	if err != nil {
		http.Error(w, "Failed to list invitations", http.StatusInternalServerError)
		return
	}
	if invitations == nil {
		invitations = []Invitation{}
	}
	type item struct {
		ID          uuid.UUID        `json:"id"`
		Email       string           `json:"email"`
		Token       string           `json:"token"`
		DefaultRole Role             `json:"default_role"`
		Status      InvitationStatus `json:"status"`
		CreatedAt   time.Time        `json:"created_at"`
		ExpiresAt   time.Time        `json:"expires_at"`
	}
	out := make([]item, 0, len(invitations))
	for _, inv := range invitations {
		out = append(out, item{
			ID: inv.ID, Email: inv.Email, Token: inv.Token,
			DefaultRole: inv.DefaultRole, Status: inv.Status,
			CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AdminHandler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("invitationID"))
	if err != nil {
		http.Error(w, "Invalid invitation ID", http.StatusBadRequest)
		return
	}
	if err := h.invites.Revoke(r.Context(), id); err != nil {
		if errors.Is(err, ErrInvitationNotFound) {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func generateInvitationToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

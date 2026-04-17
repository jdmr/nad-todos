package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memUserStore struct {
	mu    sync.Mutex
	users map[uuid.UUID]*User
}

func newMemUserStore() *memUserStore { return &memUserStore{users: make(map[uuid.UUID]*User)} }

func (m *memUserStore) Create(_ context.Context, u *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *u
	m.users[u.ID] = &cp
	return nil
}

func (m *memUserStore) FindByEmail(_ context.Context, email string) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *memUserStore) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memUserStore) List(_ context.Context) ([]User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []User
	for _, u := range m.users {
		out = append(out, *u)
	}
	return out, nil
}

func (m *memUserStore) UpdateRole(_ context.Context, id uuid.UUID, role Role) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return ErrUserNotFound
	}
	u.Role = role
	return nil
}

func (m *memUserStore) CountAdmins(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, u := range m.users {
		if u.Role == RoleAdmin && u.Status == UserStatusActive {
			n++
		}
	}
	return n, nil
}

type memInvitationStore struct {
	mu     sync.Mutex
	byID   map[uuid.UUID]*Invitation
	byTok  map[string]*Invitation
}

func newMemInvitationStore() *memInvitationStore {
	return &memInvitationStore{byID: make(map[uuid.UUID]*Invitation), byTok: make(map[string]*Invitation)}
}

func (m *memInvitationStore) Create(_ context.Context, i *Invitation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[i.ID] = i
	m.byTok[i.Token] = i
	return nil
}

func (m *memInvitationStore) FindByToken(_ context.Context, token string) (*Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	i, ok := m.byTok[token]
	if !ok {
		return nil, ErrInvitationNotFound
	}
	cp := *i
	return &cp, nil
}

func (m *memInvitationStore) List(_ context.Context) ([]Invitation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Invitation
	for _, i := range m.byID {
		out = append(out, *i)
	}
	return out, nil
}

func (m *memInvitationStore) MarkUsed(_ context.Context, id, userID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if i, ok := m.byID[id]; ok {
		i.Status = InvitationUsed
		now := time.Now()
		i.UsedAt = &now
		i.UsedBy = &userID
	}
	return nil
}

func (m *memInvitationStore) Revoke(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	i, ok := m.byID[id]
	if !ok || i.Status != InvitationPending {
		return ErrInvitationNotFound
	}
	i.Status = InvitationRevoked
	return nil
}

func (m *memInvitationStore) CountAny(_ context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byID), nil
}

func adminContext(role Role) context.Context {
	sess := &Session{UserID: uuid.New(), Role: role, ExpiresAt: time.Now().Add(time.Hour)}
	return context.WithValue(context.Background(), sessionCtxKey{}, sess)
}

func TestCreateInvitation(t *testing.T) {
	users := newMemUserStore()
	invites := newMemInvitationStore()
	hdl := NewAdminHandler(users, invites)

	body := `{"email":"new@example.com","role":"user"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/invitations", bytes.NewBufferString(body)).
		WithContext(adminContext(RoleAdmin))
	w := httptest.NewRecorder()

	hdl.CreateInvitation(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body %q)", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["email"] != "new@example.com" {
		t.Fatalf("email mismatch")
	}
	if resp["token"] == "" || resp["token"] == nil {
		t.Fatalf("token missing")
	}
}

func TestUpdateUserRole_LastAdminRefuses(t *testing.T) {
	users := newMemUserStore()
	invites := newMemInvitationStore()
	hdl := NewAdminHandler(users, invites)

	admin := &User{ID: uuid.New(), Email: "a@x", Name: "A", Role: RoleAdmin, Status: UserStatusActive}
	users.Create(context.Background(), admin)

	body := `{"role":"user"}`
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(body)).
		WithContext(adminContext(RoleAdmin))
	req.SetPathValue("userID", admin.ID.String())
	w := httptest.NewRecorder()

	hdl.UpdateUserRole(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 protecting last admin, got %d", w.Code)
	}
}

func TestUpdateUserRole_PromotesUser(t *testing.T) {
	users := newMemUserStore()
	invites := newMemInvitationStore()
	hdl := NewAdminHandler(users, invites)

	admin := &User{ID: uuid.New(), Email: "a@x", Name: "A", Role: RoleAdmin, Status: UserStatusActive}
	regular := &User{ID: uuid.New(), Email: "b@x", Name: "B", Role: RoleUser, Status: UserStatusActive}
	users.Create(context.Background(), admin)
	users.Create(context.Background(), regular)

	body := `{"role":"admin"}`
	req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(body)).
		WithContext(adminContext(RoleAdmin))
	req.SetPathValue("userID", regular.ID.String())
	w := httptest.NewRecorder()

	hdl.UpdateUserRole(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body %q)", w.Code, w.Body.String())
	}

	got, _ := users.FindByID(context.Background(), regular.ID)
	if got.Role != RoleAdmin {
		t.Fatalf("expected role admin, got %s", got.Role)
	}
}

func TestRevokeInvitation(t *testing.T) {
	users := newMemUserStore()
	invites := newMemInvitationStore()
	hdl := NewAdminHandler(users, invites)

	inv := &Invitation{
		ID: uuid.New(), Email: "x@x", Token: "tok", DefaultRole: RoleUser,
		Status: InvitationPending, ExpiresAt: time.Now().Add(time.Hour),
	}
	invites.Create(context.Background(), inv)

	req := httptest.NewRequest(http.MethodDelete, "/", nil).WithContext(adminContext(RoleAdmin))
	req.SetPathValue("invitationID", inv.ID.String())
	w := httptest.NewRecorder()
	hdl.RevokeInvitation(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	got, _ := invites.FindByToken(context.Background(), "tok")
	if got.Status != InvitationRevoked {
		t.Fatalf("expected revoked, got %s", got.Status)
	}
}

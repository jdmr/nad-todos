package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memSessionStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
}

func newMemSessionStore() *memSessionStore {
	return &memSessionStore{sessions: make(map[string]*Session)}
}

func (m *memSessionStore) Create(_ context.Context, s *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	return nil
}

func (m *memSessionStore) FindByID(_ context.Context, id string) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *memSessionStore) Revoke(_ context.Context, id string, by *uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}
	now := time.Now()
	s.RevokedAt = &now
	s.RevokedBy = by
	return nil
}

func (m *memSessionStore) FindByUserID(_ context.Context, userID uuid.UUID) ([]Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Session
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (m *memSessionStore) UpdateActivity(_ context.Context, id string, t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s, ok := m.sessions[id]; ok {
		s.LastActivityAt = t
	}
	return nil
}

func newTestUser() *User {
	return &User{ID: uuid.New(), Email: "u@example.com", Name: "U", Role: RoleUser, Status: UserStatusActive}
}

func TestSessionService_CreateAndValidate(t *testing.T) {
	store := newMemSessionStore()
	svc := NewSessionService(store, time.Hour)
	user := newTestUser()

	sess, err := svc.Create(context.Background(), user, "127.0.0.1", "test-agent", "laptop")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sess.ID == "" {
		t.Fatal("expected session ID")
	}
	if sess.Role != RoleUser {
		t.Fatalf("expected role user, got %s", sess.Role)
	}

	got, err := svc.Validate(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got.UserID != user.ID {
		t.Fatalf("user id mismatch")
	}
}

func TestSessionService_Validate_Expired(t *testing.T) {
	store := newMemSessionStore()
	svc := NewSessionService(store, time.Hour)
	user := newTestUser()
	sess, _ := svc.Create(context.Background(), user, "", "", "")

	store.mu.Lock()
	store.sessions[sess.ID].ExpiresAt = time.Now().Add(-time.Minute)
	store.mu.Unlock()

	_, err := svc.Validate(context.Background(), sess.ID)
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestSessionService_Validate_Revoked(t *testing.T) {
	store := newMemSessionStore()
	svc := NewSessionService(store, time.Hour)
	user := newTestUser()
	sess, _ := svc.Create(context.Background(), user, "", "", "")

	if err := svc.Revoke(context.Background(), sess.ID, &user.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	_, err := svc.Validate(context.Background(), sess.ID)
	if !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("expected ErrSessionInvalid, got %v", err)
	}
}

func TestSessionService_Validate_NotFound(t *testing.T) {
	store := newMemSessionStore()
	svc := NewSessionService(store, time.Hour)
	_, err := svc.Validate(context.Background(), "nope")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

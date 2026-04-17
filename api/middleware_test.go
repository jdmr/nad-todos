package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRequireSession_NoCookie(t *testing.T) {
	svc := NewSessionService(newMemSessionStore(), time.Hour)
	mw := RequireSession(svc, "todos_session")
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if called {
		t.Fatal("next handler should not have been called")
	}
}

func TestRequireSession_ValidCookie(t *testing.T) {
	store := newMemSessionStore()
	svc := NewSessionService(store, time.Hour)
	user := newTestUser()
	sess, _ := svc.Create(context.Background(), user, "", "", "")

	mw := RequireSession(svc, "todos_session")
	var ctxSess *Session
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxSess, _ = SessionFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "todos_session", Value: sess.ID})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body %q)", w.Code, w.Body.String())
	}
	if ctxSess == nil || ctxSess.UserID != user.ID {
		t.Fatal("expected session in context")
	}
}

func TestRequireRole_Forbidden(t *testing.T) {
	mw := RequireRole(RoleAdmin)
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))

	sess := &Session{UserID: uuid.New(), Role: RoleUser, ExpiresAt: time.Now().Add(time.Hour)}
	ctx := context.WithValue(context.Background(), sessionCtxKey{}, sess)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequireRole_Allowed(t *testing.T) {
	mw := RequireRole(RoleAdmin)
	called := false
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))

	sess := &Session{UserID: uuid.New(), Role: RoleAdmin, ExpiresAt: time.Now().Add(time.Hour)}
	ctx := context.WithValue(context.Background(), sessionCtxKey{}, sess)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if !called {
		t.Fatal("expected next handler to be called")
	}
}

package main

import (
	"errors"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

func TestChallengeCache_PutTake(t *testing.T) {
	c := NewChallengeCache()
	data := &webauthn.SessionData{Challenge: "abc"}
	c.Put("id-1", data, map[string]string{"k": "v"})

	got, state, err := c.Take("id-1")
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if got.Challenge != "abc" {
		t.Fatalf("challenge mismatch")
	}
	if state["k"] != "v" {
		t.Fatalf("state mismatch")
	}
}

func TestChallengeCache_TakeConsumes(t *testing.T) {
	c := NewChallengeCache()
	c.Put("id-2", &webauthn.SessionData{}, nil)

	if _, _, err := c.Take("id-2"); err != nil {
		t.Fatalf("first take failed: %v", err)
	}
	_, _, err := c.Take("id-2")
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound on second take, got %v", err)
	}
}

func TestChallengeCache_Expired(t *testing.T) {
	c := NewChallengeCache()
	c.Put("id-3", &webauthn.SessionData{}, nil)

	c.mu.Lock()
	entry := c.entries["id-3"]
	entry.expiresAt = time.Now().Add(-time.Second)
	c.entries["id-3"] = entry
	c.mu.Unlock()

	_, _, err := c.Take("id-3")
	if !errors.Is(err, ErrChallengeNotFound) {
		t.Fatalf("expected ErrChallengeNotFound, got %v", err)
	}
}

func TestChallengeCache_GC(t *testing.T) {
	c := NewChallengeCache()
	c.Put("id-4", &webauthn.SessionData{}, nil)

	c.mu.Lock()
	entry := c.entries["id-4"]
	entry.expiresAt = time.Now().Add(-time.Second)
	c.entries["id-4"] = entry
	c.mu.Unlock()

	c.gc()

	c.mu.Lock()
	_, exists := c.entries["id-4"]
	c.mu.Unlock()
	if exists {
		t.Fatal("expected entry to be GC'd")
	}
}

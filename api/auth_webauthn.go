package main

import (
	"errors"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

const challengeTTL = 5 * time.Minute

var ErrChallengeNotFound = errors.New("challenge not found or expired")

type challengeEntry struct {
	data      *webauthn.SessionData
	state     map[string]string // arbitrary string state (e.g., invitation token, email)
	expiresAt time.Time
}

// ChallengeCache holds short-lived WebAuthn ceremony state in-process.
// Restart-tolerance is sacrificed for simplicity (no Redis dependency).
type ChallengeCache struct {
	mu      sync.Mutex
	entries map[string]challengeEntry
}

func NewChallengeCache() *ChallengeCache {
	c := &ChallengeCache{entries: make(map[string]challengeEntry)}
	go c.gcLoop()
	return c
}

func (c *ChallengeCache) Put(id string, data *webauthn.SessionData, state map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[id] = challengeEntry{data: data, state: state, expiresAt: time.Now().Add(challengeTTL)}
}

func (c *ChallengeCache) Take(id string) (*webauthn.SessionData, map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.entries[id]
	if !ok {
		return nil, nil, ErrChallengeNotFound
	}
	delete(c.entries, id)
	if time.Now().After(entry.expiresAt) {
		return nil, nil, ErrChallengeNotFound
	}
	return entry.data, entry.state, nil
}

func (c *ChallengeCache) gcLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		c.gc()
	}
}

func (c *ChallengeCache) gc() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, id)
		}
	}
}

func NewWebAuthn(cfg Config) (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{
		RPDisplayName: cfg.WebAuthnRPName,
		RPID:          cfg.WebAuthnRPID,
		RPOrigins:     []string{cfg.WebAuthnRPOrigin},
	})
}

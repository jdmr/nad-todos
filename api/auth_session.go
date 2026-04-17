package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

const activityFlushInterval = 30 * time.Minute

var ErrSessionInvalid = errors.New("session invalid")

type SessionService struct {
	store    SessionStore
	duration time.Duration
}

func NewSessionService(store SessionStore, duration time.Duration) *SessionService {
	return &SessionService{store: store, duration: duration}
}

func (s *SessionService) Create(ctx context.Context, user *User, ip, userAgent, deviceName string) (*Session, error) {
	now := time.Now()
	sess := &Session{
		ID:             generateSecureToken(),
		UserID:         user.ID,
		Role:           user.Role,
		IPAddress:      ip,
		UserAgent:      userAgent,
		DeviceName:     deviceName,
		CreatedAt:      now,
		ExpiresAt:      now.Add(s.duration),
		LastActivityAt: now,
	}
	if err := s.store.Create(ctx, sess); err != nil {
		return nil, err
	}
	return sess, nil
}

func (s *SessionService) Validate(ctx context.Context, id string) (*Session, error) {
	sess, err := s.store.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !sess.IsValid() {
		return nil, ErrSessionInvalid
	}
	now := time.Now()
	if now.Sub(sess.LastActivityAt) >= activityFlushInterval {
		go func(id string, t time.Time) {
			if err := s.store.UpdateActivity(context.Background(), id, t); err != nil {
				log.Printf("session activity update failed: %v", err)
			}
		}(sess.ID, now)
	}
	return sess, nil
}

func (s *SessionService) Revoke(ctx context.Context, id string, by *uuid.UUID) error {
	return s.store.Revoke(ctx, id, by)
}

func (s *SessionService) ListByUser(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	return s.store.FindByUserID(ctx, userID)
}

func generateSecureToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

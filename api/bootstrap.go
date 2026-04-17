package main

import (
	"context"
	"log"
	"time"
)

// MaybeBootstrapAdmin creates a bootstrap admin invitation when the users
// table is empty, and prints the registration URL to stdout.
func MaybeBootstrapAdmin(ctx context.Context, cfg Config, users UserStore, invites InvitationStore) error {
	count, err := users.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// Don't print a fresh token on every startup if a bootstrap invite already exists.
	existing, err := invites.List(ctx)
	if err == nil {
		for _, inv := range existing {
			if inv.IsBootstrap && inv.Status == InvitationPending && time.Now().Before(inv.ExpiresAt) {
				logBootstrapURL(cfg, inv.Token)
				return nil
			}
		}
	}

	inv := &Invitation{
		Email:       "admin@bootstrap.local",
		Token:       generateInvitationToken(),
		DefaultRole: RoleAdmin,
		Status:      InvitationPending,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		IsBootstrap: true,
	}
	if err := invites.Create(ctx, inv); err != nil {
		return err
	}
	logBootstrapURL(cfg, inv.Token)
	return nil
}

func logBootstrapURL(cfg Config, token string) {
	log.Printf("================================================================")
	log.Printf("BOOTSTRAP: no admin found. Register the first admin via:")
	log.Printf("  %s/register?token=%s", cfg.WebAuthnRPOrigin, token)
	log.Printf("Token: %s", token)
	log.Printf("================================================================")
}

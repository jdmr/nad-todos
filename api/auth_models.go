package main

import (
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func (r Role) IsValid() bool {
	return r == RoleUser || r == RoleAdmin
}

type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusSuspended UserStatus = "suspended"
)

type User struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	Name      string     `json:"name"`
	Role      Role       `json:"role"`
	Status    UserStatus `json:"status"`
	InvitedBy *uuid.UUID `json:"invited_by,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`

	Credentials []Credential `json:"-"`
}

func (u *User) WebAuthnID() []byte          { return u.ID[:] }
func (u *User) WebAuthnName() string        { return u.Email }
func (u *User) WebAuthnDisplayName() string { return u.Name }
func (u *User) WebAuthnIcon() string        { return "" }
func (u *User) WebAuthnCredentials() []webauthn.Credential {
	out := make([]webauthn.Credential, len(u.Credentials))
	for i, c := range u.Credentials {
		out[i] = c.ToWebAuthn()
	}
	return out
}

type Credential struct {
	ID              uuid.UUID  `json:"id"`
	UserID          uuid.UUID  `json:"user_id"`
	CredentialID    []byte     `json:"-"`
	PublicKey       []byte     `json:"-"`
	AttestationType string     `json:"attestation_type"`
	AAGUID          []byte     `json:"-"`
	SignCount       uint32     `json:"sign_count"`
	CloneWarning    bool       `json:"clone_warning"`
	BackupEligible  bool       `json:"backup_eligible"`
	Transports      []string   `json:"transports"`
	DeviceName      string     `json:"device_name"`
	CreatedAt       time.Time  `json:"created_at"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
}

func (c *Credential) ToWebAuthn() webauthn.Credential {
	transports := make([]protocol.AuthenticatorTransport, len(c.Transports))
	for i, t := range c.Transports {
		transports[i] = protocol.AuthenticatorTransport(t)
	}
	return webauthn.Credential{
		ID:              c.CredentialID,
		PublicKey:       c.PublicKey,
		AttestationType: c.AttestationType,
		Transport:       transports,
		Flags: webauthn.CredentialFlags{
			UserPresent:    true,
			UserVerified:   true,
			BackupEligible: c.BackupEligible,
		},
		Authenticator: webauthn.Authenticator{
			AAGUID:       c.AAGUID,
			SignCount:    c.SignCount,
			CloneWarning: c.CloneWarning,
		},
	}
}

func CredentialFromWebAuthn(userID uuid.UUID, wc *webauthn.Credential, deviceName string) *Credential {
	transports := make([]string, len(wc.Transport))
	for i, t := range wc.Transport {
		transports[i] = string(t)
	}
	return &Credential{
		ID:              uuid.New(),
		UserID:          userID,
		CredentialID:    wc.ID,
		PublicKey:       wc.PublicKey,
		AttestationType: wc.AttestationType,
		AAGUID:          wc.Authenticator.AAGUID,
		SignCount:       wc.Authenticator.SignCount,
		CloneWarning:    wc.Authenticator.CloneWarning,
		BackupEligible:  wc.Flags.BackupEligible,
		Transports:      transports,
		DeviceName:      deviceName,
		CreatedAt:       time.Now(),
	}
}

type Session struct {
	ID             string     `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Role           Role       `json:"role"`
	IPAddress      string     `json:"ip_address"`
	UserAgent      string     `json:"user_agent"`
	DeviceName     string     `json:"device_name"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	LastActivityAt time.Time  `json:"last_activity_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	RevokedBy      *uuid.UUID `json:"revoked_by,omitempty"`
}

func (s *Session) IsValid() bool {
	if s.RevokedAt != nil {
		return false
	}
	return time.Now().Before(s.ExpiresAt)
}

type InvitationStatus string

const (
	InvitationPending InvitationStatus = "pending"
	InvitationUsed    InvitationStatus = "used"
	InvitationExpired InvitationStatus = "expired"
	InvitationRevoked InvitationStatus = "revoked"
)

type Invitation struct {
	ID          uuid.UUID        `json:"id"`
	Email       string           `json:"email"`
	Token       string           `json:"-"`
	DefaultRole Role             `json:"default_role"`
	InvitedBy   *uuid.UUID       `json:"invited_by,omitempty"`
	Status      InvitationStatus `json:"status"`
	CreatedAt   time.Time        `json:"created_at"`
	ExpiresAt   time.Time        `json:"expires_at"`
	UsedAt      *time.Time       `json:"used_at,omitempty"`
	UsedBy      *uuid.UUID       `json:"used_by,omitempty"`
	IsBootstrap bool             `json:"is_bootstrap"`
}

func (i *Invitation) IsValid() bool {
	return i.Status == InvitationPending && time.Now().Before(i.ExpiresAt)
}

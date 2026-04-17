package main

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrCredentialNotFound = errors.New("credential not found")
	ErrSessionNotFound    = errors.New("session not found")
	ErrInvitationNotFound = errors.New("invitation not found")
)

type UserStore interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	List(ctx context.Context) ([]User, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role Role) error
	CountAdmins(ctx context.Context) (int, error)
}

type CredentialStore interface {
	Create(ctx context.Context, c *Credential) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]Credential, error)
	UpdateCounter(ctx context.Context, credID []byte, counter uint32) error
	UpdateLastUsed(ctx context.Context, credID []byte) error
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type SessionStore interface {
	Create(ctx context.Context, s *Session) error
	FindByID(ctx context.Context, id string) (*Session, error)
	Revoke(ctx context.Context, id string, revokedBy *uuid.UUID) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error)
	UpdateActivity(ctx context.Context, id string, t time.Time) error
}

type InvitationStore interface {
	Create(ctx context.Context, i *Invitation) error
	FindByToken(ctx context.Context, token string) (*Invitation, error)
	List(ctx context.Context) ([]Invitation, error)
	MarkUsed(ctx context.Context, id, userID uuid.UUID) error
	Revoke(ctx context.Context, id uuid.UUID) error
	CountAny(ctx context.Context) (int, error)
}

// Postgres impls

type PostgresUserStore struct{ pool *pgxpool.Pool }

func NewPostgresUserStore(p *pgxpool.Pool) *PostgresUserStore { return &PostgresUserStore{pool: p} }

func (s *PostgresUserStore) Create(ctx context.Context, u *User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (id, email, name, role, status, invited_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		u.ID, u.Email, u.Name, string(u.Role), string(u.Status), u.InvitedBy, u.CreatedAt, u.UpdatedAt)
	return err
}

func (s *PostgresUserStore) FindByEmail(ctx context.Context, email string) (*User, error) {
	return s.scanOne(ctx, `SELECT id,email,name,role,status,invited_by,created_at,updated_at FROM users WHERE email=$1`, email)
}

func (s *PostgresUserStore) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.scanOne(ctx, `SELECT id,email,name,role,status,invited_by,created_at,updated_at FROM users WHERE id=$1`, id)
}

func (s *PostgresUserStore) scanOne(ctx context.Context, q string, args ...any) (*User, error) {
	var u User
	var role, status string
	err := s.pool.QueryRow(ctx, q, args...).Scan(&u.ID, &u.Email, &u.Name, &role, &status, &u.InvitedBy, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	u.Role, u.Status = Role(role), UserStatus(status)
	return &u, nil
}

func (s *PostgresUserStore) List(ctx context.Context) ([]User, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,email,name,role,status,invited_by,created_at,updated_at FROM users ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var role, status string
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &role, &status, &u.InvitedBy, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.Role, u.Status = Role(role), UserStatus(status)
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *PostgresUserStore) UpdateRole(ctx context.Context, id uuid.UUID, role Role) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET role=$1, updated_at=NOW() WHERE id=$2`, string(role), id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *PostgresUserStore) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role='admin' AND status='active'`).Scan(&n)
	return n, err
}

type PostgresCredentialStore struct{ pool *pgxpool.Pool }

func NewPostgresCredentialStore(p *pgxpool.Pool) *PostgresCredentialStore {
	return &PostgresCredentialStore{pool: p}
}

func (s *PostgresCredentialStore) Create(ctx context.Context, c *Credential) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	transports, _ := json.Marshal(c.Transports)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO credentials (id,user_id,credential_id,public_key,attestation_type,aaguid,sign_count,clone_warning,backup_eligible,transports,device_name,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		c.ID, c.UserID, c.CredentialID, c.PublicKey, c.AttestationType, c.AAGUID,
		c.SignCount, c.CloneWarning, c.BackupEligible, transports, c.DeviceName, c.CreatedAt)
	return err
}

func (s *PostgresCredentialStore) FindByUserID(ctx context.Context, userID uuid.UUID) ([]Credential, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id,user_id,credential_id,public_key,attestation_type,aaguid,sign_count,clone_warning,backup_eligible,transports,device_name,created_at,last_used_at
		FROM credentials WHERE user_id=$1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Credential
	for rows.Next() {
		var c Credential
		var transports []byte
		var attestation, deviceName *string
		if err := rows.Scan(&c.ID, &c.UserID, &c.CredentialID, &c.PublicKey, &attestation, &c.AAGUID,
			&c.SignCount, &c.CloneWarning, &c.BackupEligible, &transports, &deviceName, &c.CreatedAt, &c.LastUsedAt); err != nil {
			return nil, err
		}
		if attestation != nil {
			c.AttestationType = *attestation
		}
		if deviceName != nil {
			c.DeviceName = *deviceName
		}
		if len(transports) > 0 {
			_ = json.Unmarshal(transports, &c.Transports)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PostgresCredentialStore) UpdateCounter(ctx context.Context, credID []byte, counter uint32) error {
	_, err := s.pool.Exec(ctx, `UPDATE credentials SET sign_count=$1 WHERE credential_id=$2`, counter, credID)
	return err
}

func (s *PostgresCredentialStore) UpdateLastUsed(ctx context.Context, credID []byte) error {
	_, err := s.pool.Exec(ctx, `UPDATE credentials SET last_used_at=NOW() WHERE credential_id=$1`, credID)
	return err
}

func (s *PostgresCredentialStore) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM credentials WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrCredentialNotFound
	}
	return nil
}

type PostgresSessionStore struct{ pool *pgxpool.Pool }

func NewPostgresSessionStore(p *pgxpool.Pool) *PostgresSessionStore {
	return &PostgresSessionStore{pool: p}
}

func (s *PostgresSessionStore) Create(ctx context.Context, sess *Session) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (id,user_id,ip_address,user_agent,device_name,created_at,expires_at,last_activity_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		sess.ID, sess.UserID, sess.IPAddress, sess.UserAgent, sess.DeviceName,
		sess.CreatedAt, sess.ExpiresAt, sess.LastActivityAt)
	return err
}

func (s *PostgresSessionStore) FindByID(ctx context.Context, id string) (*Session, error) {
	var sess Session
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT s.id, s.user_id, u.role, s.ip_address, s.user_agent, s.device_name,
		       s.created_at, s.expires_at, s.last_activity_at, s.revoked_at, s.revoked_by
		FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.id=$1`, id).Scan(
		&sess.ID, &sess.UserID, &role, &sess.IPAddress, &sess.UserAgent, &sess.DeviceName,
		&sess.CreatedAt, &sess.ExpiresAt, &sess.LastActivityAt, &sess.RevokedAt, &sess.RevokedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	sess.Role = Role(role)
	return &sess, nil
}

func (s *PostgresSessionStore) Revoke(ctx context.Context, id string, by *uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at=NOW(), revoked_by=$1 WHERE id=$2 AND revoked_at IS NULL`, by, id)
	return err
}

func (s *PostgresSessionStore) FindByUserID(ctx context.Context, userID uuid.UUID) ([]Session, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT s.id, s.user_id, u.role, s.ip_address, s.user_agent, s.device_name,
		       s.created_at, s.expires_at, s.last_activity_at, s.revoked_at, s.revoked_by
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.user_id=$1 AND s.revoked_at IS NULL AND s.expires_at > NOW()
		ORDER BY s.last_activity_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var sess Session
		var role string
		if err := rows.Scan(&sess.ID, &sess.UserID, &role, &sess.IPAddress, &sess.UserAgent, &sess.DeviceName,
			&sess.CreatedAt, &sess.ExpiresAt, &sess.LastActivityAt, &sess.RevokedAt, &sess.RevokedBy); err != nil {
			return nil, err
		}
		sess.Role = Role(role)
		out = append(out, sess)
	}
	return out, rows.Err()
}

func (s *PostgresSessionStore) UpdateActivity(ctx context.Context, id string, t time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET last_activity_at=$1 WHERE id=$2`, t, id)
	return err
}

type PostgresInvitationStore struct{ pool *pgxpool.Pool }

func NewPostgresInvitationStore(p *pgxpool.Pool) *PostgresInvitationStore {
	return &PostgresInvitationStore{pool: p}
}

func (s *PostgresInvitationStore) Create(ctx context.Context, i *Invitation) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO invitations (id,email,token,default_role,invited_by,status,created_at,expires_at,is_bootstrap)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		i.ID, i.Email, i.Token, string(i.DefaultRole), i.InvitedBy, string(i.Status),
		i.CreatedAt, i.ExpiresAt, i.IsBootstrap)
	return err
}

func (s *PostgresInvitationStore) FindByToken(ctx context.Context, token string) (*Invitation, error) {
	var i Invitation
	var role, status string
	err := s.pool.QueryRow(ctx, `
		SELECT id,email,token,default_role,invited_by,status,created_at,expires_at,used_at,used_by,is_bootstrap
		FROM invitations WHERE token=$1`, token).Scan(
		&i.ID, &i.Email, &i.Token, &role, &i.InvitedBy, &status,
		&i.CreatedAt, &i.ExpiresAt, &i.UsedAt, &i.UsedBy, &i.IsBootstrap)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInvitationNotFound
	}
	if err != nil {
		return nil, err
	}
	i.DefaultRole, i.Status = Role(role), InvitationStatus(status)
	return &i, nil
}

func (s *PostgresInvitationStore) List(ctx context.Context) ([]Invitation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id,email,token,default_role,invited_by,status,created_at,expires_at,used_at,used_by,is_bootstrap
		FROM invitations ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invitation
	for rows.Next() {
		var i Invitation
		var role, status string
		if err := rows.Scan(&i.ID, &i.Email, &i.Token, &role, &i.InvitedBy, &status,
			&i.CreatedAt, &i.ExpiresAt, &i.UsedAt, &i.UsedBy, &i.IsBootstrap); err != nil {
			return nil, err
		}
		i.DefaultRole, i.Status = Role(role), InvitationStatus(status)
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *PostgresInvitationStore) MarkUsed(ctx context.Context, id, userID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `UPDATE invitations SET status='used', used_at=NOW(), used_by=$1 WHERE id=$2`, userID, id)
	return err
}

func (s *PostgresInvitationStore) Revoke(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `UPDATE invitations SET status='revoked' WHERE id=$1 AND status='pending'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInvitationNotFound
	}
	return nil
}

func (s *PostgresInvitationStore) CountAny(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM invitations`).Scan(&n)
	return n, err
}

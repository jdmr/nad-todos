# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go REST API + Vue.js frontend for managing todos, with WebAuthn/FIDO2 authentication and `user`/`admin` roles. Backed by PostgreSQL.

## Prerequisites

- Go 1.25+
- PostgreSQL running locally on port 5432
- Database named `todos`. Initialize schema with `psql todos -f api/todos.sql && psql todos -f api/migrations/001_auth.sql`.
- Node 20.19+ or 22.12+ for the frontend
- [air](https://github.com/air-verse/air) for live reload (optional)

## Common Commands

API (run from `api/`):

```bash
go build -o ./tmp/main .   # build
go run .                   # run (default :8080)
air                        # live reload
go test ./...              # all tests
go test -run TestName ./...  # one test
```

Frontend (run from `web/`):

```bash
npm run dev          # vite on :5173, proxies /api -> :8080
npm run build
npm run type-check
```

## Configuration (env vars, with defaults)

- `DATABASE_URL` — `postgres://localhost:5432/todos`
- `LISTEN_ADDR` — `:8080`
- `WEBAUTHN_RP_ID` — `localhost`
- `WEBAUTHN_RP_ORIGIN` — `http://localhost:5173`
- `WEBAUTHN_RP_NAME` — `Todos`
- `SESSION_COOKIE_NAME` — `todos_session`
- `SESSION_DURATION` — `8h`

## Bootstrap (first run)

When the `users` table has no admins, the API creates a one-time bootstrap invitation on startup and prints the registration URL to stdout. Visit it to register the first admin. Subsequent admins are created via the admin UI (Invitations page).

## Architecture

Single-module Go API (`api/`, `package main`) using only the standard library `net/http` router (Go 1.22+ method-pattern syntax), `pgx/v5` for PostgreSQL, `go-webauthn/webauthn` for FIDO2.

### API layout (all `package main`)

- `main.go` — wires pool, stores, services, handlers, middleware; registers routes
- `config.go` — env-var config loader
- `handler.go`, `store.go` — todos CRUD (interface-backed for tests)
- `auth_models.go` — `User`, `Credential`, `Session`, `Invitation`, `Role` (`user`/`admin`)
- `auth_store.go` — `UserStore`, `CredentialStore`, `SessionStore`, `InvitationStore` interfaces + Postgres impls
- `auth_session.go` — `SessionService`: secure tokens, validate, revoke, throttled activity flush
- `auth_webauthn.go` — go-webauthn config + in-process `ChallengeCache` (5-min TTL, no Redis)
- `auth_handler.go` — public auth routes + authed self-management of sessions/credentials
- `admin_handler.go` — admin-only user/role/invitation management
- `middleware.go` — `RequireSession`, `RequireRole`, `SessionFromContext`
- `bootstrap.go` — first-run admin invitation
- `migrations/001_auth.sql` — auth schema (depends on `pgcrypto`)

### Routes

Public (no session needed):
- `GET /api/v1/auth/invitations/{token}`
- `POST /api/v1/auth/{register,login}/{options,verify}`

Authenticated:
- `/api/v1/todos` CRUD (existing)
- `GET /api/v1/auth/me`, `POST /api/v1/auth/logout`
- `GET /api/v1/auth/sessions`, `POST /api/v1/auth/sessions/revoke`
- `GET /api/v1/auth/credentials`, `POST /api/v1/auth/credentials/{options,verify}`, `DELETE /api/v1/auth/credentials/{id}`

Admin-only:
- `GET /api/v1/admin/users`, `PUT /api/v1/admin/users/{id}/role`
- `GET|POST /api/v1/admin/invitations`, `DELETE /api/v1/admin/invitations/{id}`

### Notable invariants

- **Last admin protection** — `UpdateUserRole` refuses to demote the only active admin.
- **Last credential protection** — `DeleteCredential` refuses to remove the user's only passkey.
- **Challenge cache is in-process** — restarting the API mid-ceremony invalidates outstanding challenges. Acceptable for now; swap for Redis if multi-instance.
- **Session cookie is `SameSite=Lax`** — frontend dev runs through Vite's `/api` proxy so the cookie is same-origin. `Secure` is auto-set when `WEBAUTHN_RP_ORIGIN` starts with `https://`.

### Frontend layout (`web/src/`)

- `api/client.ts` — axios instance, `withCredentials: true`, 401 interceptor
- `api/auth.ts` — WebAuthn ceremonies via `@simplewebauthn/browser`, session/credential helpers
- `api/admin.ts` — admin user/invitation calls
- `stores/auth.ts` — pinia store, persisted via `pinia-plugin-persistedstate`
- `views/{Login,Register,Account}View.vue`, `views/admin/{Users,Invitations}View.vue`
- `router/index.ts` — guards: `requiresAuth`, `requiresAdmin`; verifies session on first nav after rehydrate

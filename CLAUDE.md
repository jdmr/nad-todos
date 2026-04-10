# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Go REST API for managing todos, backed by PostgreSQL. Single-service architecture with all code in the `api/` directory.

## Prerequisites

- Go 1.25+
- PostgreSQL running locally on port 5432
- Database named `todos` (create table with `api/todos.sql`)
- [air](https://github.com/air-verse/air) for live reload (optional)

## Common Commands

All commands run from the `api/` directory:

```bash
# Build
go build -o ./tmp/main .

# Run (listens on :8080)
go run .

# Live reload (uses .air.toml config)
air

# Run tests
go test ./...

# Run a single test
go test -run TestName ./...
```

## Architecture

Single-file Go API (`api/main.go`) using only the standard library `net/http` router (Go 1.22+ method-pattern syntax) and `pgx/v5` for PostgreSQL.

- **No framework** — routes registered on `http.NewServeMux` with method+path patterns (e.g., `GET /api/v1/todos/{todoID}`)
- **Database** — `pgxpool` connection pool; hardcoded connection string `postgres://localhost:5432/todos`
- **All routes** under `/api/v1/todos` — standard CRUD (list, create, get, update, delete)
- **`TodoHandler`** struct holds the `pgxpool.Pool` and all handler methods
- **Binary output** — `api/tmp/main` (gitignored via `tmp/`)

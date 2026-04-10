package main

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestStore(t *testing.T) *PostgresTodoStore {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, "postgres://localhost:5432/todos")
	if err != nil {
		t.Skip("skipping integration test: cannot connect to database:", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skip("skipping integration test: cannot ping database:", err)
	}

	// Ensure table exists
	_, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS todos (
		id serial primary key,
		title text not null,
		completed boolean not null default false
	)`)
	if err != nil {
		pool.Close()
		t.Fatalf("failed to create table: %v", err)
	}

	// Clean slate for each test
	_, err = pool.Exec(ctx, "DELETE FROM todos")
	if err != nil {
		pool.Close()
		t.Fatalf("failed to truncate table: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
	})

	return NewPostgresTodoStore(pool)
}

func TestStoreCreate(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	todo := &Todo{Title: "Buy milk", Completed: false}
	if err := store.Create(ctx, todo); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if todo.ID == 0 {
		t.Error("expected ID to be set after create")
	}
}

func TestStoreGet(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	created := &Todo{Title: "Buy milk", Completed: false}
	if err := store.Create(ctx, created); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	got, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get todo: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, got.ID)
	}
	if got.Title != "Buy milk" {
		t.Errorf("expected title 'Buy milk', got %q", got.Title)
	}
	if got.Completed != false {
		t.Error("expected completed to be false")
	}
}

func TestStoreGet_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	_, err := store.Get(ctx, 99999)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStoreUpdate(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	created := &Todo{Title: "Buy milk", Completed: false}
	if err := store.Create(ctx, created); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	created.Title = "Buy oat milk"
	created.Completed = true
	if err := store.Update(ctx, *created); err != nil {
		t.Fatalf("failed to update todo: %v", err)
	}

	got, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get updated todo: %v", err)
	}
	if got.Title != "Buy oat milk" {
		t.Errorf("expected title 'Buy oat milk', got %q", got.Title)
	}
	if !got.Completed {
		t.Error("expected completed to be true")
	}
}

func TestStoreUpdate_NotFound(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	err := store.Update(ctx, Todo{ID: 99999, Title: "Nope"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStoreDelete(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	created := &Todo{Title: "Buy milk", Completed: false}
	if err := store.Create(ctx, created); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("failed to delete todo: %v", err)
	}

	_, err := store.Get(ctx, created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestStoreList(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	if err := store.Create(ctx, &Todo{Title: "First"}); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}
	if err := store.Create(ctx, &Todo{Title: "Second", Completed: true}); err != nil {
		t.Fatalf("failed to create todo: %v", err)
	}

	todos, err := store.List(ctx)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("expected 2 todos, got %d", len(todos))
	}
}

func TestStoreList_Empty(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	todos, err := store.List(ctx)
	if err != nil {
		t.Fatalf("failed to list todos: %v", err)
	}
	if todos != nil {
		t.Errorf("expected nil for empty list, got %v", todos)
	}
}

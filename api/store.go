package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type TodoStore interface {
	Create(ctx context.Context, todo *Todo) error
	Get(ctx context.Context, id int) (Todo, error)
	Update(ctx context.Context, todo Todo) error
	Delete(ctx context.Context, id int) error
	List(ctx context.Context) ([]Todo, error)
}

type PostgresTodoStore struct {
	pool *pgxpool.Pool
}

func NewPostgresTodoStore(pool *pgxpool.Pool) *PostgresTodoStore {
	return &PostgresTodoStore{pool: pool}
}

func (s *PostgresTodoStore) Create(ctx context.Context, todo *Todo) error {
	return s.pool.QueryRow(ctx, "INSERT INTO todos (title, completed) VALUES ($1, $2) RETURNING id", todo.Title, todo.Completed).Scan(&todo.ID)
}

func (s *PostgresTodoStore) Get(ctx context.Context, id int) (Todo, error) {
	var todo Todo
	err := s.pool.QueryRow(ctx, "SELECT id, title, completed FROM todos WHERE id = $1", id).Scan(&todo.ID, &todo.Title, &todo.Completed)
	if err != nil {
		return Todo{}, ErrNotFound
	}
	return todo, nil
}

func (s *PostgresTodoStore) Update(ctx context.Context, todo Todo) error {
	result, err := s.pool.Exec(ctx, "UPDATE todos SET title = $1, completed = $2 WHERE id = $3", todo.Title, todo.Completed, todo.ID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *PostgresTodoStore) Delete(ctx context.Context, id int) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM todos WHERE id = $1", id)
	return err
}

func (s *PostgresTodoStore) List(ctx context.Context) ([]Todo, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, title, completed FROM todos")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed); err != nil {
			return nil, err
		}
		todos = append(todos, t)
	}
	return todos, nil
}

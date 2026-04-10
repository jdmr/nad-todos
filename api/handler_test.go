package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockTodoStore struct {
	todos  map[int]Todo
	nextID int
}

func newMockTodoStore() *mockTodoStore {
	return &mockTodoStore{todos: make(map[int]Todo), nextID: 1}
}

func (m *mockTodoStore) Create(_ context.Context, todo *Todo) error {
	todo.ID = m.nextID
	m.todos[todo.ID] = *todo
	m.nextID++
	return nil
}

func (m *mockTodoStore) Get(_ context.Context, id int) (Todo, error) {
	todo, ok := m.todos[id]
	if !ok {
		return Todo{}, ErrNotFound
	}
	return todo, nil
}

func (m *mockTodoStore) Update(_ context.Context, todo Todo) error {
	if _, ok := m.todos[todo.ID]; !ok {
		return ErrNotFound
	}
	m.todos[todo.ID] = todo
	return nil
}

func (m *mockTodoStore) Delete(_ context.Context, id int) error {
	delete(m.todos, id)
	return nil
}

func (m *mockTodoStore) List(_ context.Context) ([]Todo, error) {
	todos := make([]Todo, 0, len(m.todos))
	for _, t := range m.todos {
		todos = append(todos, t)
	}
	return todos, nil
}

func TestCreateTodo(t *testing.T) {
	store := newMockTodoStore()
	hdl := NewTodoHandler(store)

	body := `{"title":"Buy milk","completed":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	hdl.CreateTodo(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var todo Todo
	if err := json.NewDecoder(w.Body).Decode(&todo); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if todo.ID != 1 {
		t.Errorf("expected ID 1, got %d", todo.ID)
	}
	if todo.Title != "Buy milk" {
		t.Errorf("expected title 'Buy milk', got %q", todo.Title)
	}
}

func TestCreateTodo_InvalidJSON(t *testing.T) {
	store := newMockTodoStore()
	hdl := NewTodoHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/todos", bytes.NewBufferString(`{bad`))
	w := httptest.NewRecorder()

	hdl.CreateTodo(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestGetTodo(t *testing.T) {
	store := newMockTodoStore()
	store.todos[1] = Todo{ID: 1, Title: "Buy milk", Completed: false}
	hdl := NewTodoHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/1", nil)
	req.SetPathValue("todoID", "1")
	w := httptest.NewRecorder()

	hdl.GetTodo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var todo Todo
	if err := json.NewDecoder(w.Body).Decode(&todo); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if todo.Title != "Buy milk" {
		t.Errorf("expected title 'Buy milk', got %q", todo.Title)
	}
}

func TestGetTodo_NotFound(t *testing.T) {
	store := newMockTodoStore()
	hdl := NewTodoHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos/999", nil)
	req.SetPathValue("todoID", "999")
	w := httptest.NewRecorder()

	hdl.GetTodo(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUpdateTodo(t *testing.T) {
	store := newMockTodoStore()
	store.todos[1] = Todo{ID: 1, Title: "Buy milk", Completed: false}
	hdl := NewTodoHandler(store)

	body := `{"title":"Buy oat milk","completed":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/1", bytes.NewBufferString(body))
	req.SetPathValue("todoID", "1")
	w := httptest.NewRecorder()

	hdl.UpdateTodo(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var todo Todo
	if err := json.NewDecoder(w.Body).Decode(&todo); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if todo.Title != "Buy oat milk" {
		t.Errorf("expected title 'Buy oat milk', got %q", todo.Title)
	}
	if !todo.Completed {
		t.Error("expected completed to be true")
	}
}

func TestUpdateTodo_NotFound(t *testing.T) {
	store := newMockTodoStore()
	hdl := NewTodoHandler(store)

	body := `{"title":"Nope"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/999", bytes.NewBufferString(body))
	req.SetPathValue("todoID", "999")
	w := httptest.NewRecorder()

	hdl.UpdateTodo(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestUpdateTodo_IDMismatch(t *testing.T) {
	store := newMockTodoStore()
	store.todos[1] = Todo{ID: 1, Title: "Buy milk", Completed: false}
	hdl := NewTodoHandler(store)

	body := `{"id":2,"title":"Buy milk"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/todos/1", bytes.NewBufferString(body))
	req.SetPathValue("todoID", "1")
	w := httptest.NewRecorder()

	hdl.UpdateTodo(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDeleteTodo(t *testing.T) {
	store := newMockTodoStore()
	store.todos[1] = Todo{ID: 1, Title: "Buy milk", Completed: false}
	hdl := NewTodoHandler(store)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/todos/1", nil)
	req.SetPathValue("todoID", "1")
	w := httptest.NewRecorder()

	hdl.DeleteTodo(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	if _, ok := store.todos[1]; ok {
		t.Error("expected todo to be deleted from store")
	}
}

func TestListTodos(t *testing.T) {
	store := newMockTodoStore()
	store.todos[1] = Todo{ID: 1, Title: "Buy milk", Completed: false}
	store.todos[2] = Todo{ID: 2, Title: "Walk dog", Completed: true}
	hdl := NewTodoHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	w := httptest.NewRecorder()

	hdl.ListTodos(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var todos []Todo
	if err := json.NewDecoder(w.Body).Decode(&todos); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("expected 2 todos, got %d", len(todos))
	}
}

func TestListTodos_Empty(t *testing.T) {
	store := newMockTodoStore()
	hdl := NewTodoHandler(store)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/todos", nil)
	w := httptest.NewRecorder()

	hdl.ListTodos(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var todos []Todo
	if err := json.NewDecoder(w.Body).Decode(&todos); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(todos) != 0 {
		t.Errorf("expected 0 todos, got %d", len(todos))
	}
}

package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting API server...")

	log.Printf("Connecting to database...")
	pool, err := pgxpool.New(context.Background(), "postgres://localhost:5432/todos")
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer func() {
		log.Printf("Closing database connection...")
		pool.Close()
	}()

	log.Printf("Database connected successfully.")

	hdl := &TodoHandler{pool: pool}
	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/todos", hdl.ListTodos)
	router.HandleFunc("POST /api/v1/todos", hdl.CreateTodo)
	router.HandleFunc("GET /api/v1/todos/{todoID}", hdl.GetTodo)
	router.HandleFunc("PUT /api/v1/todos/{todoID}", hdl.UpdateTodo)
	router.HandleFunc("DELETE /api/v1/todos/{todoID}", hdl.DeleteTodo)
	log.Printf("API server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

type TodoHandler struct {
	pool *pgxpool.Pool
}

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func (h *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating a new todo
	log.Printf("Creating a new todo...")
	todo := Todo{}
	log.Printf("Decoding request body into Todo struct...")
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	log.Printf("Decoded todo: %+v", todo)

	log.Printf("Inserting todo into database...")
	err := h.pool.QueryRow(context.Background(), "INSERT INTO todos (title, completed) VALUES ($1, $2) RETURNING id", todo.Title, todo.Completed).Scan(&todo.ID)
	if err != nil {
		http.Error(w, "Failed to create todo", http.StatusInternalServerError)
		return
	}

	log.Printf("Todo created successfully: %+v", todo)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(todo); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *TodoHandler) GetTodo(w http.ResponseWriter, r *http.Request) {
	log.Printf("Getting a todo by ID...")
	todoID := r.PathValue("todoID")
	if todoID == "" {
		http.Error(w, "Missing todo ID", http.StatusBadRequest)
		return
	}

	log.Printf("Querying database for todo with ID: %s", todoID)
	var todo Todo
	err := h.pool.QueryRow(context.Background(), "SELECT id, title, completed FROM todos WHERE id = $1", todoID).Scan(&todo.ID, &todo.Title, &todo.Completed)
	if err != nil {
		http.Error(w, "Todo not found", http.StatusNotFound)
		return
	}

	log.Printf("Todo found: %+v", todo)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(todo); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *TodoHandler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	todoID := r.PathValue("todoID")
	if todoID == "" {
		http.Error(w, "Missing todo ID", http.StatusBadRequest)
		return
	}
	todoIDInt, err := strconv.Atoi(todoID)
	if err != nil {
		http.Error(w, "Invalid todo ID", http.StatusBadRequest)
		return
	}

	todo := Todo{}
	if err := json.NewDecoder(r.Body).Decode(&todo); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if todo.ID != 0 && todo.ID != todoIDInt {
		http.Error(w, "Todo ID in payload does not match URL", http.StatusBadRequest)
		return
	}
}

func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	todoID := r.PathValue("todoID")
	if todoID == "" {
		http.Error(w, "Missing todo ID", http.StatusBadRequest)
		return
	}

	_, err := h.pool.Exec(context.Background(), "DELETE FROM todos WHERE id = $1", todoID)
	if err != nil {
		http.Error(w, "Failed to delete todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TodoHandler) ListTodos(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(context.Background(), "SELECT id, title, completed FROM todos")
	if err != nil {
		http.Error(w, "Failed to query todos", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	todos := []Todo{}
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed); err != nil {
			http.Error(w, "Failed to scan todo", http.StatusInternalServerError)
			return
		}
		todos = append(todos, t)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(todos); err != nil {
		http.Error(w, "Failed to encode todos", http.StatusInternalServerError)
		return
	}
}

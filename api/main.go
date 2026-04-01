package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

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
	router.HandleFunc("/api/v1/todos", hdl.ListTodos)
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

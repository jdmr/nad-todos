package main

import (
	"context"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

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

	store := NewPostgresTodoStore(pool)
	hdl := NewTodoHandler(store)

	router := http.NewServeMux()
	router.HandleFunc("GET /api/v1/todos", hdl.ListTodos)
	router.HandleFunc("POST /api/v1/todos", hdl.CreateTodo)
	router.HandleFunc("GET /api/v1/todos/{todoID}", hdl.GetTodo)
	router.HandleFunc("PUT /api/v1/todos/{todoID}", hdl.UpdateTodo)
	router.HandleFunc("DELETE /api/v1/todos/{todoID}", hdl.DeleteTodo)

	log.Printf("API server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

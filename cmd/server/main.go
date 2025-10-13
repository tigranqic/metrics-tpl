package main

import (
	"log"
	"net/http"

	"github.com/tigranqic/metrics-tpl/internal/handler"
	"github.com/tigranqic/metrics-tpl/internal/repository"
)

func main() {
	store := repository.NewMemStorage()
	h := handler.NewHandler(store)

	log.Println("Server is running on :8080")
	if err := http.ListenAndServe(":8080", h.Router()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}


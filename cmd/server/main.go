package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tigranqic/metrics-tpl/internal/handler"
	"github.com/tigranqic/metrics-tpl/internal/repository"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")

	flag.Parse()
	if len(flag.Args()) > 0 {
		fmt.Fprintf(os.Stderr, "Unknown arguments: %v\n", flag.Args())
		os.Exit(1)
	}

	store := repository.NewMemStorage()
	h := handler.NewHandler(store)

	log.Printf("Server is running on %s\n", *addr)

	if err := http.ListenAndServe(*addr, h.Router()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

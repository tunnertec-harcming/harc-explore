package main

import (
	"log"
	"net/http"
	"os"

	"github.com/harc/soundscape/apps/api/internal/handlers"
)

func main() {
	addr := ":8000"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}
	log.Printf("Harc cloud API listening on %s", addr)
	if err := http.ListenAndServe(addr, handlers.NewMux()); err != nil {
		log.Fatal(err)
	}
}

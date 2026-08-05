package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/harc/soundscape/apps/api/internal/data"
	"github.com/harc/soundscape/apps/api/internal/handlers"
)

func main() {
	addr := ":8000"
	if v := os.Getenv("PORT"); v != "" {
		addr = ":" + v
	}

	// Prefer running from apps/api so assets/stems resolves.
	if _, err := os.Stat("assets/stems"); err != nil {
		candidates := []string{"apps/api", filepath.Join("..", "api"), "."}
		for _, c := range candidates {
			if st, err := os.Stat(filepath.Join(c, "assets", "stems")); err == nil && st.IsDir() {
				_ = os.Chdir(c)
				break
			}
		}
	}
	log.Printf("Harc cloud API listening on %s (stems=%s)", addr, data.StemsRoot())
	if err := http.ListenAndServe(addr, handlers.NewMux()); err != nil {
		log.Fatal(err)
	}
}

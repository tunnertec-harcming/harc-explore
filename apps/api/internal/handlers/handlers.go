package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/harc/soundscape/apps/api/internal/data"
	"github.com/harc/soundscape/apps/api/internal/models"
)

func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func Modes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"modes": data.Modes()})
}

func Packs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		writeJSON(w, http.StatusOK, map[string]any{"packs": data.Packs()})
		return
	}
	packs := data.PacksByMode(models.ModeID(mode))
	writeJSON(w, http.StatusOK, map[string]any{"packs": packs})
}

func PackByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/packs/")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "invalid pack id")
		return
	}
	pack, ok := data.PackByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "pack not found")
		return
	}
	writeJSON(w, http.StatusOK, pack)
}

func SpeakerBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defaults := data.DefaultPackIDs()
	packs := make([]models.Pack, 0, len(defaults))
	for _, id := range defaults {
		if p, ok := data.PackByID(id); ok {
			packs = append(packs, p)
		}
	}
	writeJSON(w, http.StatusOK, models.SpeakerBootstrap{
		APIVersion:     "v1",
		DefaultPackIDs: defaults,
		Modes:          data.Modes(),
		Packs:          packs,
	})
}

func NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", withCORS(Health))
	mux.HandleFunc("/modes", withCORS(Modes))
	mux.HandleFunc("/packs", withCORS(Packs))
	mux.HandleFunc("/packs/", withCORS(PackByID))
	mux.HandleFunc("/v1/speaker/bootstrap", withCORS(SpeakerBootstrap))
	return mux
}

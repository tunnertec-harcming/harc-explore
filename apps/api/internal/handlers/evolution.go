package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/harc/soundscape/apps/api/internal/evolution"
	"github.com/harc/soundscape/apps/api/internal/models"
)

var (
	evoOnce  sync.Once
	evoStore *evolution.Store
)

func store() *evolution.Store {
	evoOnce.Do(func() {
		evoStore = evolution.NewStore()
	})
	return evoStore
}

func evolutionEnabled() bool {
	v := os.Getenv("EVOLUTION_ENABLED")
	return v != "0" && v != "false"
}

type eventsRequest struct {
	DeviceID  string             `json:"device_id"`
	SessionID string             `json:"session_id"`
	Events    []evolution.Event  `json:"events"`
}

type recommendRequest struct {
	DeviceID  string         `json:"device_id"`
	Mode      models.ModeID  `json:"mode"`
	HourLocal *int           `json:"hour_local"`
}

type personalizeRequest struct {
	DeviceID  string `json:"device_id"`
	PackID    string `json:"pack_id"`
	HourLocal *int   `json:"hour_local"`
}

func EvolutionEvents(w http.ResponseWriter, r *http.Request) {
	if !evolutionEnabled() {
		writeError(w, http.StatusServiceUnavailable, "evolution disabled")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req eventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.DeviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id required")
		return
	}
	n := store().Ingest(req.DeviceID, req.Events)
	writeJSON(w, http.StatusOK, map[string]any{"accepted": n})
}

func EvolutionRecommend(w http.ResponseWriter, r *http.Request) {
	if !evolutionEnabled() {
		writeError(w, http.StatusServiceUnavailable, "evolution disabled")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req recommendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.DeviceID == "" || req.Mode == "" {
		writeError(w, http.StatusBadRequest, "device_id and mode required")
		return
	}
	hour := evolution.HourLocal()
	if req.HourLocal != nil {
		hour = *req.HourLocal
	}
	writeJSON(w, http.StatusOK, store().Recommend(req.DeviceID, req.Mode, hour))
}

func EvolutionPersonalize(w http.ResponseWriter, r *http.Request) {
	if !evolutionEnabled() {
		writeError(w, http.StatusServiceUnavailable, "evolution disabled")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req personalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.DeviceID == "" || req.PackID == "" {
		writeError(w, http.StatusBadRequest, "device_id and pack_id required")
		return
	}
	hour := evolution.HourLocal()
	if req.HourLocal != nil {
		hour = *req.HourLocal
	}
	writeJSON(w, http.StatusOK, store().Personalize(req.DeviceID, req.PackID, hour))
}

func EvolutionInsights(w http.ResponseWriter, r *http.Request) {
	if !evolutionEnabled() {
		writeError(w, http.StatusServiceUnavailable, "evolution disabled")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := r.URL.Query().Get("device_id")
	if deviceID == "" {
		writeError(w, http.StatusBadRequest, "device_id required")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"device_id": deviceID,
		"stats":     store().Insights(deviceID),
	})
}

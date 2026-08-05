package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harc/soundscape/apps/api/internal/handlers"
)

func TestEvolutionLoop(t *testing.T) {
	mux := handlers.NewMux()
	device := "test-device-1"

	body := map[string]any{
		"device_id": device,
		"events": []map[string]any{
			{"type": "play_start", "ts": 1, "mode": "sleep", "pack_id": "sleep-soft-rain"},
			{"type": "heartbeat", "ts": 2, "mode": "sleep", "pack_id": "sleep-soft-rain", "listened_sec": 120},
			{"type": "play_stop", "ts": 3, "mode": "sleep", "pack_id": "sleep-soft-rain", "listened_sec": 60},
		},
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/evolution/events", bytes.NewReader(b))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("events %d %s", rr.Code, rr.Body.String())
	}

	recBody, _ := json.Marshal(map[string]any{
		"device_id": device, "mode": "sleep", "hour_local": 23,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/evolution/recommend", bytes.NewReader(recBody))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("recommend %d %s", rr.Code, rr.Body.String())
	}
	var rec map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &rec)
	if rec["pack_id"] != "sleep-soft-rain" {
		t.Fatalf("expected soft-rain, got %v", rec)
	}

	perBody, _ := json.Marshal(map[string]any{
		"device_id": device, "pack_id": "sleep-soft-rain", "hour_local": 23,
	})
	req = httptest.NewRequest(http.MethodPost, "/v1/evolution/personalize", bytes.NewReader(perBody))
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("personalize %d %s", rr.Code, rr.Body.String())
	}
	var per struct {
		ProfileDelta struct {
			Energy float64 `json:"energy"`
		} `json:"profile_delta"`
		Source string `json:"source"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &per)
	if per.ProfileDelta.Energy >= 0 {
		t.Fatalf("expected softer energy at night, got %v", per)
	}
}

package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harc/soundscape/apps/api/internal/handlers"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	handlers.NewMux().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
}

func TestModesAndPacks(t *testing.T) {
	mux := handlers.NewMux()

	req := httptest.NewRequest(http.MethodGet, "/modes", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("modes status %d", rr.Code)
	}
	var modesBody map[string][]any
	if err := json.Unmarshal(rr.Body.Bytes(), &modesBody); err != nil {
		t.Fatal(err)
	}
	if len(modesBody["modes"]) != 4 {
		t.Fatalf("expected 4 modes, got %d", len(modesBody["modes"]))
	}

	req = httptest.NewRequest(http.MethodGet, "/packs?mode=sleep", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	var packsBody struct {
		Packs []struct {
			Mode string `json:"mode"`
			ID   string `json:"id"`
		} `json:"packs"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &packsBody); err != nil {
		t.Fatal(err)
	}
	if len(packsBody.Packs) == 0 {
		t.Fatal("expected sleep packs")
	}
	for _, p := range packsBody.Packs {
		if p.Mode != "sleep" {
			t.Fatalf("unexpected mode %s", p.Mode)
		}
	}

	req = httptest.NewRequest(http.MethodGet, "/packs/sleep-deep-night", nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("pack detail status %d body %s", rr.Code, rr.Body.String())
	}
}

func TestBootstrap(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/speaker/bootstrap", nil)
	rr := httptest.NewRecorder()
	handlers.NewMux().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
}

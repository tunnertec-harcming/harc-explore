package cache_test

import (
	"testing"

	"github.com/harc/soundscape/apps/speaker/internal/cache"
	"github.com/harc/soundscape/apps/speaker/internal/models"
)

func TestApplyDeltaClamp(t *testing.T) {
	p := models.Pack{EngineProfile: models.EngineProfile{Energy: 0.2, Brightness: 0.9}}
	out := cache.ApplyDelta(p, models.ProfileDelta{Energy: -0.5, Brightness: 0.5})
	if out.EngineProfile.Energy != 0 {
		t.Fatalf("energy %v", out.EngineProfile.Energy)
	}
	if out.EngineProfile.Brightness != 1 {
		t.Fatalf("brightness %v", out.EngineProfile.Brightness)
	}
}

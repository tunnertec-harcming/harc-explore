package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/harc/soundscape/apps/speaker/internal/cache"
	"github.com/harc/soundscape/apps/speaker/internal/cloud"
	"github.com/harc/soundscape/apps/speaker/internal/engine"
	"github.com/harc/soundscape/apps/speaker/internal/models"
)

func main() {
	var (
		apiBase    = flag.String("api", env("HARC_API", "http://127.0.0.1:8000"), "cloud API base URL")
		deviceID   = flag.String("device", env("HARC_DEVICE", "speaker-dev-1"), "device id for evolution")
		mode       = flag.String("mode", "sleep", "mode: sleep|focus|atmosphere|relax")
		packID     = flag.String("pack", "", "pack id (empty = recommend/default)")
		cacheDir   = flag.String("cache", env("HARC_CACHE", filepath.Join(os.TempDir(), "harc-speaker-cache")), "stem cache dir")
		maxCacheMB = flag.Int64("cache-mb", 150, "max cache size MB (2G device budget)")
		duration   = flag.Int("duration", 1, "play duration minutes (0=infinite where supported)")
		dryRun     = flag.Bool("dry-run", false, "download + plan only, do not play")
		render     = flag.String("render", "", "render mixed audio to this file instead of playing")
		renderSec  = flag.Int("render-sec", 8, "seconds to render when -render is set")
	)
	flag.Parse()

	client := cloud.New(*apiBase, *deviceID)
	boot, err := client.Bootstrap()
	must(err)
	fmt.Printf("bootstrap api=%s defaults=%v\n", boot.APIVersion, boot.DefaultPackIDs)

	modeID := models.ModeID(*mode)
	chosen := *packID
	hour := time.Now().Hour()
	if chosen == "" {
		if rec, err := client.Recommend(modeID, hour); err == nil && rec.PackID != "" {
			chosen = rec.PackID
			fmt.Printf("recommend %s (%s: %s)\n", rec.PackID, rec.Source, rec.Reason)
		} else if def, ok := boot.DefaultPackIDs[modeID]; ok {
			chosen = def
			fmt.Printf("default pack %s\n", chosen)
		}
	}
	if chosen == "" {
		fail("no pack selected")
	}

	pack, err := client.Pack(chosen)
	must(err)

	if per, err := client.Personalize(chosen, hour); err == nil {
		pack = cache.ApplyDelta(pack, per.ProfileDelta)
		fmt.Printf("personalize %s (%s)\n", per.Reason, per.Source)
	}

	store := cache.New(*cacheDir, (*maxCacheMB)*1024*1024, client)
	stems, err := store.EnsurePack(chosen)
	must(err)
	fmt.Printf("cached %d stems under %s\n", len(stems), *cacheDir)

	plan := engine.BuildPlan(pack, stems, *duration)
	planPath := filepath.Join(*cacheDir, "last-plan.json")
	must(engine.New().WritePlanJSON(plan, planPath))
	fmt.Printf("plan -> %s (%d layers)\n", planPath, len(plan.Layers))

	session := fmt.Sprintf("spk-%d", time.Now().Unix())
	_ = client.PostEvents(session, []map[string]any{{
		"type": "play_start", "ts": time.Now().Unix(), "mode": modeID, "pack_id": chosen, "duration_min": *duration,
	}})

	player := engine.New()
	switch {
	case *render != "":
		must(player.RenderFile(plan, *render, *renderSec))
	default:
		must(player.Play(plan, *dryRun))
	}

	_ = client.PostEvents(session, []map[string]any{{
		"type": "play_stop", "ts": time.Now().Unix(), "mode": modeID, "pack_id": chosen, "listened_sec": float64(*renderSec),
	}})
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func must(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}

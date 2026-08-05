package engine

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/harc/soundscape/apps/speaker/internal/models"
)

// Player mixes local opus stems with ffmpeg amix (reference for 2G Linux speakers).
// Production devices can replace this with on-device DSP while keeping the same plan.
type Player struct {
	FFmpeg string
}

type LayerPlan struct {
	LayerID string  `json:"layer_id"`
	Kind    string  `json:"kind"`
	Path    string  `json:"path"`
	Gain    float64 `json:"gain"`
}

type PlayPlan struct {
	PackID   string             `json:"pack_id"`
	Mode     models.ModeID      `json:"mode"`
	Profile  models.EngineProfile `json:"profile"`
	Layers   []LayerPlan        `json:"layers"`
	Duration time.Duration      `json:"-"`
}

func New() *Player {
	bin := "ffmpeg"
	if v := os.Getenv("FFMPEG"); v != "" {
		bin = v
	}
	return &Player{FFmpeg: bin}
}

func BuildPlan(pack models.Pack, stems map[string]string, durationMin int) PlayPlan {
	profile := pack.EngineProfile
	hourMul := 1.0
	h := time.Now().Hour()
	if h >= 22 || h < 5 {
		hourMul = 0.9
	}
	var layers []LayerPlan
	for _, l := range pack.Layers {
		path, ok := stems[l.ID]
		if !ok {
			continue
		}
		kindMul := 1.0
		switch l.Kind {
		case "event":
			kindMul = math.Max(0.15, profile.EventDensity)
		case "bed", "noise":
			kindMul = 0.55 + profile.Masking*0.4
		case "pulse":
			kindMul = 0.75
		default:
			kindMul = 0.9
		}
		gain := l.Gain * (0.28 + profile.Energy*0.7) * kindMul * hourMul
		layers = append(layers, LayerPlan{
			LayerID: l.ID,
			Kind:    l.Kind,
			Path:    path,
			Gain:    gain,
		})
	}
	d := time.Duration(0)
	if durationMin > 0 {
		d = time.Duration(durationMin) * time.Minute
	}
	return PlayPlan{PackID: pack.ID, Mode: pack.Mode, Profile: profile, Layers: layers, Duration: d}
}

func (p *Player) WritePlanJSON(plan PlayPlan, path string) error {
	// lightweight without importing encoding in hot path repeatedly
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString(fmt.Sprintf("  \"pack_id\": %q,\n", plan.PackID))
	b.WriteString(fmt.Sprintf("  \"mode\": %q,\n", plan.Mode))
	b.WriteString(fmt.Sprintf("  \"energy\": %.3f,\n", plan.Profile.Energy))
	b.WriteString(fmt.Sprintf("  \"brightness\": %.3f,\n", plan.Profile.Brightness))
	b.WriteString("  \"layers\": [\n")
	for i, l := range plan.Layers {
		comma := ","
		if i == len(plan.Layers)-1 {
			comma = ""
		}
		b.WriteString(fmt.Sprintf("    {\"layer_id\":%q,\"kind\":%q,\"path\":%q,\"gain\":%.4f}%s\n",
			l.LayerID, l.Kind, l.Path, l.Gain, comma))
	}
	b.WriteString("  ]\n}\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// Play starts ffmpeg amix. If dryRun, only validates inputs.
func (p *Player) Play(plan PlayPlan, dryRun bool) error {
	if len(plan.Layers) == 0 {
		return fmt.Errorf("no layers to play")
	}
	for _, l := range plan.Layers {
		if _, err := os.Stat(l.Path); err != nil {
			return fmt.Errorf("missing stem %s: %w", l.Path, err)
		}
	}
	if dryRun {
		fmt.Printf("dry-run play pack=%s layers=%d\n", plan.PackID, len(plan.Layers))
		return nil
	}
	if _, err := exec.LookPath(p.FFmpeg); err != nil {
		return fmt.Errorf("ffmpeg not found (set FFMPEG or install): %w", err)
	}

	args := []string{"-hide_banner", "-loglevel", "error"}
	var filters []string
	var amixInputs []string
	for i, l := range plan.Layers {
		args = append(args, "-stream_loop", "-1", "-i", l.Path)
		// volume + soft lowpass from brightness
		cut := 280 + plan.Profile.Brightness*3200
		filters = append(filters, fmt.Sprintf("[%d:a]volume=%.4f,lowpass=f=%.0f[a%d]", i, l.Gain, cut, i))
		amixInputs = append(amixInputs, fmt.Sprintf("[a%d]", i))
	}
	filter := strings.Join(filters, ";") + ";" +
		fmt.Sprintf("%samix=inputs=%d:dropout_transition=2:normalize=0[out]", strings.Join(amixInputs, ""), len(plan.Layers))
	args = append(args, "-filter_complex", filter, "-map", "[out]")
	if plan.Duration > 0 {
		args = append(args, "-t", fmt.Sprintf("%.0f", plan.Duration.Seconds()))
	}
	args = append(args, "-f", "alsa", "default")

	cmd := exec.Command(p.FFmpeg, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	fmt.Printf("playing %s (%d layers)\n", plan.PackID, len(plan.Layers))
	return cmd.Run()
}

// RenderFile mixes to a wav/ogg file for CI or headless verification.
func (p *Player) RenderFile(plan PlayPlan, outPath string, seconds int) error {
	if seconds <= 0 {
		seconds = 8
	}
	args := []string{"-y", "-hide_banner", "-loglevel", "error"}
	var filters []string
	var amixInputs []string
	for i, l := range plan.Layers {
		args = append(args, "-stream_loop", "-1", "-i", l.Path)
		cut := 280 + plan.Profile.Brightness*3200
		filters = append(filters, fmt.Sprintf("[%d:a]volume=%.4f,lowpass=f=%.0f[a%d]", i, l.Gain, cut, i))
		amixInputs = append(amixInputs, fmt.Sprintf("[a%d]", i))
	}
	filter := strings.Join(filters, ";") + ";" +
		fmt.Sprintf("%samix=inputs=%d:dropout_transition=2:normalize=0[out]", strings.Join(amixInputs, ""), len(plan.Layers))
	args = append(args,
		"-filter_complex", filter, "-map", "[out]",
		"-t", fmt.Sprintf("%d", seconds),
		outPath,
	)
	cmd := exec.Command(p.FFmpeg, args...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	abs, _ := filepath.Abs(outPath)
	fmt.Printf("rendered %s\n", abs)
	return nil
}

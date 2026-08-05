package models

// Re-export cloud shapes used by speaker via local copies to keep speaker module independent.
// Speaker talks JSON HTTP; these structs mirror apps/api/internal/models.

type ModeID string

const (
	ModeSleep      ModeID = "sleep"
	ModeFocus      ModeID = "focus"
	ModeAtmosphere ModeID = "atmosphere"
	ModeRelax      ModeID = "relax"
)

type Phase struct {
	ID            string  `json:"id"`
	DurationRatio float64 `json:"duration_ratio"`
	EnergyMul     float64 `json:"energy_mul"`
	DensityMul    float64 `json:"density_mul"`
}

type EngineProfile struct {
	Energy             float64 `json:"energy"`
	EventDensity       float64 `json:"event_density"`
	TempoBPM           float64 `json:"tempo_bpm"`
	Brightness         float64 `json:"brightness"`
	Masking            float64 `json:"masking"`
	Space              float64 `json:"space"`
	VariationPeriodSec float64 `json:"variation_period_sec"`
	Phases             []Phase `json:"phases"`
}

type Layer struct {
	ID      string  `json:"id"`
	Kind    string  `json:"kind"`
	Gain    float64 `json:"gain"`
	StemURL *string `json:"stem_url,omitempty"`
	Loop    bool    `json:"loop"`
}

type Pack struct {
	ID            string        `json:"id"`
	Mode          ModeID        `json:"mode"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Tier          string        `json:"tier"`
	Version       string        `json:"version"`
	EngineProfile EngineProfile `json:"engine_profile"`
	Layers        []Layer       `json:"layers"`
	ApproxBytes   int64         `json:"approx_bytes"`
}

type Mode struct {
	ID                 ModeID `json:"id"`
	Name               string `json:"name"`
	Tagline            string `json:"tagline"`
	DurationOptionsMin []int  `json:"duration_options_min"`
	DefaultDurationMin int    `json:"default_duration_min"`
}

type PackFile struct {
	LayerID string `json:"layer_id"`
	Path    string `json:"path"`
	Bytes   int64  `json:"bytes"`
}

type PackManifest struct {
	PackID     string     `json:"pack_id"`
	Version    string     `json:"version"`
	StemFormat string     `json:"stem_format"`
	Files      []PackFile `json:"files"`
	TotalBytes int64      `json:"total_bytes"`
}

type SpeakerBootstrap struct {
	APIVersion     string            `json:"api_version"`
	DefaultPackIDs map[ModeID]string `json:"default_pack_ids"`
	Modes          []Mode            `json:"modes"`
	Packs          []Pack            `json:"packs"`
}

type ProfileDelta struct {
	Energy       float64 `json:"energy"`
	Brightness   float64 `json:"brightness"`
	Masking      float64 `json:"masking"`
	Space        float64 `json:"space"`
	EventDensity float64 `json:"event_density"`
}

type RecommendResult struct {
	PackID string  `json:"pack_id"`
	Score  float64 `json:"score"`
	Source string  `json:"source"`
	Reason string  `json:"reason"`
}

type PersonalizeResult struct {
	PackID       string       `json:"pack_id"`
	ProfileDelta ProfileDelta `json:"profile_delta"`
	Source       string       `json:"source"`
	Reason       string       `json:"reason"`
}

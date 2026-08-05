package models

type ModeID string

const (
	ModeSleep      ModeID = "sleep"
	ModeFocus      ModeID = "focus"
	ModeAtmosphere ModeID = "atmosphere"
	ModeRelax      ModeID = "relax"
)

type Palette struct {
	BG0    string `json:"bg0"`
	BG1    string `json:"bg1"`
	Accent string `json:"accent"`
	Glow   string `json:"glow"`
}

type Mode struct {
	ID                  ModeID  `json:"id"`
	Name                string  `json:"name"`
	Tagline             string  `json:"tagline"`
	DurationOptionsMin  []int   `json:"duration_options_min"`
	DefaultDurationMin  int     `json:"default_duration_min"`
	Palette             Palette `json:"palette"`
}

type PhaseID string

const (
	PhaseIntro   PhaseID = "intro"
	PhaseSustain PhaseID = "sustain"
	PhaseOutro   PhaseID = "outro"
)

type Phase struct {
	ID            PhaseID `json:"id"`
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

type LayerKind string

const (
	LayerDrone   LayerKind = "drone"
	LayerNoise   LayerKind = "noise"
	LayerPulse   LayerKind = "pulse"
	LayerTexture LayerKind = "texture"
	LayerEvent   LayerKind = "event"
	LayerBed     LayerKind = "bed"
)

type SynthType string

const (
	SynthSine     SynthType = "sine"
	SynthTriangle SynthType = "triangle"
	SynthSawtooth SynthType = "sawtooth"
	SynthSquare   SynthType = "square"
	SynthBrown    SynthType = "brown"
	SynthPink     SynthType = "pink"
	SynthWhite    SynthType = "white"
)

type SynthParams struct {
	Type     SynthType `json:"type"`
	FreqHz   *float64  `json:"freq_hz,omitempty"`
	LfoHz    *float64  `json:"lfo_hz,omitempty"`
	LfoDepth *float64  `json:"lfo_depth,omitempty"`
}

type Layer struct {
	ID      string       `json:"id"`
	Kind    LayerKind    `json:"kind"`
	Gain    float64      `json:"gain"`
	Synth   *SynthParams `json:"synth,omitempty"`
	StemURL *string      `json:"stem_url,omitempty"`
	Loop    bool         `json:"loop"`
}

type Tier string

const (
	TierFree    Tier = "free"
	TierPremium Tier = "premium"
)

type Pack struct {
	ID            string        `json:"id"`
	Mode          ModeID        `json:"mode"`
	Name          string        `json:"name"`
	Description   string        `json:"description"`
	Tier          Tier          `json:"tier"`
	Version       string        `json:"version"`
	EngineProfile EngineProfile `json:"engine_profile"`
	Layers        []Layer       `json:"layers"`
	ApproxBytes   int64         `json:"approx_bytes"`
}

type SpeakerBootstrap struct {
	APIVersion     string            `json:"api_version"`
	DefaultPackIDs map[ModeID]string `json:"default_pack_ids"`
	Modes          []Mode            `json:"modes"`
	Packs          []Pack            `json:"packs"`
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

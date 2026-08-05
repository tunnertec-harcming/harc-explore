package data

import "github.com/harc/soundscape/apps/api/internal/models"

func f64(v float64) *float64 { return &v }

func Modes() []models.Mode {
	return []models.Mode{
		{
			ID:                 models.ModeSleep,
			Name:               "睡眠",
			Tagline:            "低频缓慢，陪你入夜",
			DurationOptionsMin: []int{30, 60, 480},
			DefaultDurationMin: 60,
			Palette: models.Palette{
				BG0: "#070B14", BG1: "#152238", Accent: "#7EB6FF", Glow: "#3A6EA5",
			},
		},
		{
			ID:                 models.ModeFocus,
			Name:               "办公",
			Tagline:            "稳定脉冲，进入心流",
			DurationOptionsMin: []int{25, 50, 90},
			DefaultDurationMin: 50,
			Palette: models.Palette{
				BG0: "#0A1210", BG1: "#163028", Accent: "#5CDBA8", Glow: "#2A8F6E",
			},
		},
		{
			ID:                 models.ModeAtmosphere,
			Name:               "氛围",
			Tagline:            "把空间变成一个地方",
			DurationOptionsMin: []int{-1},
			DefaultDurationMin: -1,
			Palette: models.Palette{
				BG0: "#120E0A", BG1: "#2A2118", Accent: "#E0A86C", Glow: "#A66B3A",
			},
		},
		{
			ID:                 models.ModeRelax,
			Name:               "放松",
			Tagline:            "柔和起伏，慢慢下来",
			DurationOptionsMin: []int{10, 20, 45},
			DefaultDurationMin: 20,
			Palette: models.Palette{
				BG0: "#0C1014", BG1: "#1A2830", Accent: "#8EC5C8", Glow: "#4A8A90",
			},
		},
	}
}

func Packs() []models.Pack {
	return []models.Pack{
		sleepDeepNight(),
		sleepSoftRain(),
		focusClear(),
		focusDeepWork(),
		atmosphereRainCafe(),
		atmosphereForestDusk(),
		relaxWarmTide(),
		relaxStillAir(),
	}
}

func DefaultPackIDs() map[models.ModeID]string {
	return map[models.ModeID]string{
		models.ModeSleep:      "sleep-deep-night",
		models.ModeFocus:      "focus-clear",
		models.ModeAtmosphere: "atmosphere-rain-cafe",
		models.ModeRelax:      "relax-warm-tide",
	}
}

func PackByID(id string) (models.Pack, bool) {
	for _, p := range Packs() {
		if p.ID == id {
			return p, true
		}
	}
	return models.Pack{}, false
}

func PacksByMode(mode models.ModeID) []models.Pack {
	out := make([]models.Pack, 0)
	for _, p := range Packs() {
		if p.Mode == mode {
			out = append(out, p)
		}
	}
	return out
}

func sleepDeepNight() models.Pack {
	return models.Pack{
		ID: "sleep-deep-night", Mode: models.ModeSleep, Name: "深夜静海",
		Description: "低沉棕噪与缓慢低频，适合长夜维持。",
		Tier: models.TierFree, Version: "1.0.0", ApproxBytes: 48_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.18, EventDensity: 0.08, TempoBPM: 0, Brightness: 0.22,
			Masking: 0.78, Space: 0.65, VariationPeriodSec: 90,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.08, EnergyMul: 0.7, DensityMul: 0.5},
				{ID: models.PhaseSustain, DurationRatio: 0.84, EnergyMul: 1.0, DensityMul: 1.0},
				{ID: models.PhaseOutro, DurationRatio: 0.08, EnergyMul: 0.4, DensityMul: 0.3},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_brown", Kind: models.LayerBed, Gain: 0.55, Loop: true, Synth: &models.SynthParams{Type: models.SynthBrown}},
			{ID: "drone_low", Kind: models.LayerDrone, Gain: 0.28, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(55), LfoHz: f64(0.04), LfoDepth: f64(0.15)}},
			{ID: "drone_sub", Kind: models.LayerDrone, Gain: 0.18, Loop: true, Synth: &models.SynthParams{Type: models.SynthTriangle, FreqHz: f64(82), LfoHz: f64(0.03), LfoDepth: f64(0.1)}},
			{ID: "texture_soft", Kind: models.LayerTexture, Gain: 0.12, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
			{ID: "event_drop", Kind: models.LayerEvent, Gain: 0.06, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(220), LfoHz: f64(0.012), LfoDepth: f64(0.8)}},
		},
	}
}

func sleepSoftRain() models.Pack {
	return models.Pack{
		ID: "sleep-soft-rain", Mode: models.ModeSleep, Name: "细雨窗边",
		Description: "柔粉噪雨感纹理，遮掩城市噪声。",
		Tier: models.TierFree, Version: "1.0.0", ApproxBytes: 52_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.22, EventDensity: 0.15, TempoBPM: 0, Brightness: 0.3,
			Masking: 0.7, Space: 0.55, VariationPeriodSec: 70,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.1, EnergyMul: 0.6, DensityMul: 0.6},
				{ID: models.PhaseSustain, DurationRatio: 0.82, EnergyMul: 1.0, DensityMul: 1.0},
				{ID: models.PhaseOutro, DurationRatio: 0.08, EnergyMul: 0.35, DensityMul: 0.4},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_pink", Kind: models.LayerBed, Gain: 0.48, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
			{ID: "drone_warm", Kind: models.LayerDrone, Gain: 0.2, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(98), LfoHz: f64(0.05), LfoDepth: f64(0.12)}},
			{ID: "texture_rain", Kind: models.LayerTexture, Gain: 0.22, Loop: true, Synth: &models.SynthParams{Type: models.SynthWhite, LfoHz: f64(0.08), LfoDepth: f64(0.25)}},
			{ID: "event_drip", Kind: models.LayerEvent, Gain: 0.08, Loop: true, Synth: &models.SynthParams{Type: models.SynthTriangle, FreqHz: f64(480), LfoHz: f64(0.02), LfoDepth: f64(0.9)}},
		},
	}
}

func focusClear() models.Pack {
	return models.Pack{
		ID: "focus-clear", Mode: models.ModeFocus, Name: "清透专注",
		Description: "轻脉冲与稳定床层，适合深度办公。",
		Tier: models.TierFree, Version: "1.0.0", ApproxBytes: 45_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.48, EventDensity: 0.2, TempoBPM: 72, Brightness: 0.48,
			Masking: 0.45, Space: 0.35, VariationPeriodSec: 45,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.06, EnergyMul: 0.75, DensityMul: 0.8},
				{ID: models.PhaseSustain, DurationRatio: 0.88, EnergyMul: 1.0, DensityMul: 1.0},
				{ID: models.PhaseOutro, DurationRatio: 0.06, EnergyMul: 0.55, DensityMul: 0.6},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_soft", Kind: models.LayerBed, Gain: 0.32, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
			{ID: "drone_mid", Kind: models.LayerDrone, Gain: 0.22, Loop: true, Synth: &models.SynthParams{Type: models.SynthTriangle, FreqHz: f64(110), LfoHz: f64(0.06), LfoDepth: f64(0.08)}},
			{ID: "pulse_soft", Kind: models.LayerPulse, Gain: 0.18, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(146), LfoHz: f64(1.2), LfoDepth: f64(0.55)}},
			{ID: "texture_air", Kind: models.LayerTexture, Gain: 0.1, Loop: true, Synth: &models.SynthParams{Type: models.SynthWhite}},
			{ID: "event_chime", Kind: models.LayerEvent, Gain: 0.05, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(660), LfoHz: f64(0.03), LfoDepth: f64(0.85)}},
		},
	}
}

func focusDeepWork() models.Pack {
	return models.Pack{
		ID: "focus-deep-work", Mode: models.ModeFocus, Name: "深潜工作",
		Description: "更沉的节奏床，减少高频干扰。",
		Tier: models.TierPremium, Version: "1.0.0", ApproxBytes: 55_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.55, EventDensity: 0.12, TempoBPM: 64, Brightness: 0.38,
			Masking: 0.5, Space: 0.3, VariationPeriodSec: 55,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.05, EnergyMul: 0.8, DensityMul: 0.7},
				{ID: models.PhaseSustain, DurationRatio: 0.9, EnergyMul: 1.0, DensityMul: 1.0},
				{ID: models.PhaseOutro, DurationRatio: 0.05, EnergyMul: 0.5, DensityMul: 0.5},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_brown", Kind: models.LayerBed, Gain: 0.36, Loop: true, Synth: &models.SynthParams{Type: models.SynthBrown}},
			{ID: "drone_low", Kind: models.LayerDrone, Gain: 0.26, Loop: true, Synth: &models.SynthParams{Type: models.SynthSawtooth, FreqHz: f64(73), LfoHz: f64(0.04), LfoDepth: f64(0.06)}},
			{ID: "pulse_deep", Kind: models.LayerPulse, Gain: 0.2, Loop: true, Synth: &models.SynthParams{Type: models.SynthTriangle, FreqHz: f64(98), LfoHz: f64(1.06), LfoDepth: f64(0.5)}},
			{ID: "texture_grain", Kind: models.LayerTexture, Gain: 0.08, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
		},
	}
}

func atmosphereRainCafe() models.Pack {
	return models.Pack{
		ID: "atmosphere-rain-cafe", Mode: models.ModeAtmosphere, Name: "雨夜咖啡馆",
		Description: "窗外雨声与室内暖调低频，场所感更强。",
		Tier: models.TierFree, Version: "1.0.0", ApproxBytes: 60_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.4, EventDensity: 0.42, TempoBPM: 0, Brightness: 0.42,
			Masking: 0.35, Space: 0.7, VariationPeriodSec: 40,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.05, EnergyMul: 0.85, DensityMul: 0.8},
				{ID: models.PhaseSustain, DurationRatio: 0.95, EnergyMul: 1.0, DensityMul: 1.0},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_pink", Kind: models.LayerBed, Gain: 0.3, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
			{ID: "drone_warm", Kind: models.LayerDrone, Gain: 0.24, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(130), LfoHz: f64(0.07), LfoDepth: f64(0.1)}},
			{ID: "texture_rain", Kind: models.LayerTexture, Gain: 0.28, Loop: true, Synth: &models.SynthParams{Type: models.SynthWhite, LfoHz: f64(0.15), LfoDepth: f64(0.3)}},
			{ID: "event_cup", Kind: models.LayerEvent, Gain: 0.07, Loop: true, Synth: &models.SynthParams{Type: models.SynthTriangle, FreqHz: f64(520), LfoHz: f64(0.04), LfoDepth: f64(0.9)}},
			{ID: "event_murmur", Kind: models.LayerEvent, Gain: 0.05, Loop: true, Synth: &models.SynthParams{Type: models.SynthBrown, LfoHz: f64(0.025), LfoDepth: f64(0.7)}},
		},
	}
}

func atmosphereForestDusk() models.Pack {
	return models.Pack{
		ID: "atmosphere-forest-dusk", Mode: models.ModeAtmosphere, Name: "黄昏林间",
		Description: "疏朗事件与空气感纹理，适合空间氛围。",
		Tier: models.TierPremium, Version: "1.0.0", ApproxBytes: 58_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.35, EventDensity: 0.5, TempoBPM: 0, Brightness: 0.5,
			Masking: 0.25, Space: 0.8, VariationPeriodSec: 35,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.08, EnergyMul: 0.7, DensityMul: 0.6},
				{ID: models.PhaseSustain, DurationRatio: 0.92, EnergyMul: 1.0, DensityMul: 1.0},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_air", Kind: models.LayerBed, Gain: 0.22, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
			{ID: "drone_green", Kind: models.LayerDrone, Gain: 0.2, Loop: true, Synth: &models.SynthParams{Type: models.SynthTriangle, FreqHz: f64(164), LfoHz: f64(0.05), LfoDepth: f64(0.12)}},
			{ID: "texture_leaves", Kind: models.LayerTexture, Gain: 0.18, Loop: true, Synth: &models.SynthParams{Type: models.SynthWhite, LfoHz: f64(0.2), LfoDepth: f64(0.35)}},
			{ID: "event_bird", Kind: models.LayerEvent, Gain: 0.09, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(880), LfoHz: f64(0.035), LfoDepth: f64(0.95)}},
		},
	}
}

func relaxWarmTide() models.Pack {
	return models.Pack{
		ID: "relax-warm-tide", Mode: models.ModeRelax, Name: "暖潮起伏",
		Description: "缓慢起伏能量曲线，帮助身体放松。",
		Tier: models.TierFree, Version: "1.0.0", ApproxBytes: 50_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.28, EventDensity: 0.18, TempoBPM: 48, Brightness: 0.32,
			Masking: 0.4, Space: 0.72, VariationPeriodSec: 50,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.12, EnergyMul: 0.85, DensityMul: 0.7},
				{ID: models.PhaseSustain, DurationRatio: 0.7, EnergyMul: 1.0, DensityMul: 1.0},
				{ID: models.PhaseOutro, DurationRatio: 0.18, EnergyMul: 0.35, DensityMul: 0.4},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_brown", Kind: models.LayerBed, Gain: 0.34, Loop: true, Synth: &models.SynthParams{Type: models.SynthBrown}},
			{ID: "drone_wave", Kind: models.LayerDrone, Gain: 0.26, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(90), LfoHz: f64(0.08), LfoDepth: f64(0.35)}},
			{ID: "pulse_breath", Kind: models.LayerPulse, Gain: 0.12, Loop: true, Synth: &models.SynthParams{Type: models.SynthTriangle, FreqHz: f64(120), LfoHz: f64(0.15), LfoDepth: f64(0.6)}},
			{ID: "texture_soft", Kind: models.LayerTexture, Gain: 0.1, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
		},
	}
}

func relaxStillAir() models.Pack {
	return models.Pack{
		ID: "relax-still-air", Mode: models.ModeRelax, Name: "静空气",
		Description: "更少事件、更多空间感，适合短时恢复。",
		Tier: models.TierFree, Version: "1.0.0", ApproxBytes: 42_000_000,
		EngineProfile: models.EngineProfile{
			Energy: 0.2, EventDensity: 0.1, TempoBPM: 0, Brightness: 0.28,
			Masking: 0.35, Space: 0.85, VariationPeriodSec: 60,
			Phases: []models.Phase{
				{ID: models.PhaseIntro, DurationRatio: 0.15, EnergyMul: 0.7, DensityMul: 0.5},
				{ID: models.PhaseSustain, DurationRatio: 0.65, EnergyMul: 1.0, DensityMul: 1.0},
				{ID: models.PhaseOutro, DurationRatio: 0.2, EnergyMul: 0.25, DensityMul: 0.3},
			},
		},
		Layers: []models.Layer{
			{ID: "bed_air", Kind: models.LayerBed, Gain: 0.28, Loop: true, Synth: &models.SynthParams{Type: models.SynthPink}},
			{ID: "drone_soft", Kind: models.LayerDrone, Gain: 0.22, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(108), LfoHz: f64(0.045), LfoDepth: f64(0.18)}},
			{ID: "texture_haze", Kind: models.LayerTexture, Gain: 0.12, Loop: true, Synth: &models.SynthParams{Type: models.SynthWhite}},
			{ID: "event_bell", Kind: models.LayerEvent, Gain: 0.04, Loop: true, Synth: &models.SynthParams{Type: models.SynthSine, FreqHz: f64(740), LfoHz: f64(0.018), LfoDepth: f64(0.92)}},
		},
	}
}

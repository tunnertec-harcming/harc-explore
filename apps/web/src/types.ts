export type ModeId = "sleep" | "focus" | "atmosphere" | "relax";

export interface Palette {
  bg0: string;
  bg1: string;
  accent: string;
  glow: string;
}

export interface Mode {
  id: ModeId;
  name: string;
  tagline: string;
  duration_options_min: number[];
  default_duration_min: number;
  palette: Palette;
}

export interface Phase {
  id: "intro" | "sustain" | "outro";
  duration_ratio: number;
  energy_mul: number;
  density_mul: number;
}

export interface EngineProfile {
  energy: number;
  event_density: number;
  tempo_bpm: number;
  brightness: number;
  masking: number;
  space: number;
  variation_period_sec: number;
  phases: Phase[];
}

export type SynthType =
  | "sine"
  | "triangle"
  | "sawtooth"
  | "square"
  | "brown"
  | "pink"
  | "white";

export interface SynthParams {
  type: SynthType;
  freq_hz?: number;
  lfo_hz?: number;
  lfo_depth?: number;
}

export interface Layer {
  id: string;
  kind: string;
  gain: number;
  synth?: SynthParams;
  stem_url?: string;
  loop: boolean;
}

export interface Pack {
  id: string;
  mode: ModeId;
  name: string;
  description: string;
  tier: "free" | "premium";
  version: string;
  engine_profile: EngineProfile;
  layers: Layer[];
  approx_bytes: number;
}

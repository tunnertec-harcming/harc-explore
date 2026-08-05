import type { EngineProfile, Layer, Pack, SynthParams } from "../types";
import { API_BASE } from "../api/client";

type LiveValues = {
  energy: number;
  density: number;
  brightness: number;
  phase: string;
  source: "stem" | "synth" | "mixed";
};

type Listener = (v: LiveValues) => void;

function resolveURL(path: string) {
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  return `${API_BASE}${path}`;
}

function createNoiseBuffer(
  ctx: AudioContext,
  type: "white" | "pink" | "brown",
  seconds = 2,
): AudioBuffer {
  const sampleRate = ctx.sampleRate;
  const length = sampleRate * seconds;
  const buffer = ctx.createBuffer(1, length, sampleRate);
  const data = buffer.getChannelData(0);

  if (type === "white") {
    for (let i = 0; i < length; i++) data[i] = Math.random() * 2 - 1;
    return buffer;
  }

  if (type === "pink") {
    let b0 = 0,
      b1 = 0,
      b2 = 0,
      b3 = 0,
      b4 = 0,
      b5 = 0,
      b6 = 0;
    for (let i = 0; i < length; i++) {
      const white = Math.random() * 2 - 1;
      b0 = 0.99886 * b0 + white * 0.0555179;
      b1 = 0.99332 * b1 + white * 0.0750759;
      b2 = 0.969 * b2 + white * 0.153852;
      b3 = 0.8665 * b3 + white * 0.3104856;
      b4 = 0.55 * b4 + white * 0.5329522;
      b5 = -0.7616 * b5 - white * 0.016898;
      data[i] = (b0 + b1 + b2 + b3 + b4 + b5 + b6 + white * 0.5362) * 0.11;
      b6 = white * 0.115926;
    }
    return buffer;
  }

  let last = 0;
  for (let i = 0; i < length; i++) {
    const white = Math.random() * 2 - 1;
    last = (last + 0.02 * white) / 1.02;
    data[i] = last * 3.5;
  }
  return buffer;
}

type LayerNodes = {
  sources: AudioScheduledSourceNode[];
  gain: GainNode;
  filter: BiquadFilterNode;
  baseGain: number;
  kind: string;
  lfo?: OscillatorNode;
  lfoGain?: GainNode;
};

export class SoundscapeEngine {
  private ctx: AudioContext | null = null;
  private master: GainNode | null = null;
  private compressor: DynamicsCompressorNode | null = null;
  private air: BiquadFilterNode | null = null;
  private warmth: BiquadFilterNode | null = null;
  private layers: LayerNodes[] = [];
  private spaceNodes: AudioNode[] = [];
  private pack: Pack | null = null;
  private startedAt = 0;
  private durationSec = 0;
  private raf = 0;
  private listeners = new Set<Listener>();
  private playing = false;
  private bufferCache = new Map<string, AudioBuffer>();
  private lastSource: LiveValues["source"] = "synth";

  isPlaying() {
    return this.playing;
  }

  subscribe(fn: Listener) {
    this.listeners.add(fn);
    return () => {
      this.listeners.delete(fn);
    };
  }

  private emit(v: LiveValues) {
    for (const fn of this.listeners) fn(v);
  }

  async ensureContext() {
    if (!this.ctx) {
      this.ctx = new AudioContext();
      this.master = this.ctx.createGain();
      this.master.gain.value = 0.0001;

      // gentle tone shaping: warm lowpass + soft presence dip
      this.air = this.ctx.createBiquadFilter();
      this.air.type = "highshelf";
      this.air.frequency.value = 5200;
      this.air.gain.value = -3.5;

      this.warmth = this.ctx.createBiquadFilter();
      this.warmth.type = "lowshelf";
      this.warmth.frequency.value = 180;
      this.warmth.gain.value = 1.8;

      this.compressor = this.ctx.createDynamicsCompressor();
      this.compressor.threshold.value = -28;
      this.compressor.knee.value = 24;
      this.compressor.ratio.value = 2.4;
      this.compressor.attack.value = 0.02;
      this.compressor.release.value = 0.35;

      this.master.connect(this.warmth);
      this.warmth.connect(this.air);
      this.air.connect(this.compressor);
      this.compressor.connect(this.ctx.destination);
    }
    if (this.ctx.state === "suspended") await this.ctx.resume();
  }

  async play(pack: Pack, durationMin: number) {
    await this.ensureContext();
    await this.crossfadeTo(pack, durationMin);
  }

  async stop(fadeSec = 2.0) {
    if (!this.ctx || !this.master) return;
    const now = this.ctx.currentTime;
    this.master.gain.cancelScheduledValues(now);
    this.master.gain.setValueAtTime(this.master.gain.value, now);
    this.master.gain.linearRampToValueAtTime(0.0001, now + fadeSec);
    window.setTimeout(() => this.teardownLayers(), fadeSec * 1000 + 50);
    this.playing = false;
    cancelAnimationFrame(this.raf);
  }

  private async crossfadeTo(pack: Pack, durationMin: number) {
    if (!this.ctx || !this.master) return;
    const now = this.ctx.currentTime;
    const fade = 2.2;

    this.master.gain.cancelScheduledValues(now);
    this.master.gain.setValueAtTime(Math.max(this.master.gain.value, 0.0001), now);
    this.master.gain.linearRampToValueAtTime(0.0001, now + fade);

    await new Promise((r) => setTimeout(r, fade * 1000));
    this.teardownLayers();
    // bust cache when pack version/path changes
    this.bufferCache.clear();

    this.pack = pack;
    this.durationSec = durationMin < 0 ? 0 : durationMin * 60;
    this.startedAt = this.ctx.currentTime;
    this.lastSource = await this.buildGraph(pack);
    const t = this.ctx.currentTime;
    this.master.gain.setValueAtTime(0.0001, t);
    this.master.gain.linearRampToValueAtTime(0.62, t + fade);
    this.playing = true;
    this.tick();
  }

  private teardownLayers() {
    for (const layer of this.layers) {
      try {
        layer.lfo?.stop();
      } catch {
        /* ignore */
      }
      for (const s of layer.sources) {
        try {
          s.stop();
        } catch {
          /* ignore */
        }
        try {
          s.disconnect();
        } catch {
          /* ignore */
        }
      }
      try {
        layer.gain.disconnect();
        layer.filter.disconnect();
      } catch {
        /* ignore */
      }
    }
    this.layers = [];
    for (const n of this.spaceNodes) {
      try {
        n.disconnect();
      } catch {
        /* ignore */
      }
    }
    this.spaceNodes = [];
  }

  private async loadStem(url: string): Promise<AudioBuffer | null> {
    if (!this.ctx) return null;
    const full = resolveURL(url);
    const cached = this.bufferCache.get(full);
    if (cached) return cached;
    try {
      const res = await fetch(full);
      if (!res.ok) return null;
      const arr = await res.arrayBuffer();
      const buf = await this.ctx.decodeAudioData(arr.slice(0));
      this.bufferCache.set(full, buf);
      return buf;
    } catch {
      return null;
    }
  }

  private connectLayerShell(profile: EngineProfile, layer: Layer) {
    if (!this.ctx || !this.master) throw new Error("no audio context");
    const filter = this.ctx.createBiquadFilter();
    filter.type = "lowpass";
    // darker, slower rolloff — avoids harsh tops
    const baseCut = 280 + profile.brightness * 3200;
    filter.frequency.value = baseCut;
    filter.Q.value = 0.45;

    const gain = this.ctx.createGain();
    const kindMul =
      layer.kind === "event"
        ? Math.max(0.15, profile.event_density * 0.85)
        : layer.kind === "bed" || layer.kind === "noise"
          ? 0.55 + profile.masking * 0.4
          : layer.kind === "pulse"
            ? 0.75
            : 0.9;
    const base = layer.gain * (0.28 + profile.energy * 0.7) * kindMul;
    gain.gain.value = base;
    filter.connect(gain);
    gain.connect(this.master);
    return { filter, gain, base };
  }

  private startSynthLayer(
    profile: EngineProfile,
    layer: Layer,
    synth: SynthParams,
  ) {
    if (!this.ctx) return;
    const { filter, gain, base } = this.connectLayerShell(profile, layer);
    const sources: AudioScheduledSourceNode[] = [];

    if (synth.type === "white" || synth.type === "pink" || synth.type === "brown") {
      const buf = createNoiseBuffer(this.ctx, synth.type);
      const src = this.ctx.createBufferSource();
      src.buffer = buf;
      src.loop = true;
      src.connect(filter);
      src.start();
      sources.push(src);
      this.layers.push({ sources, gain, filter, baseGain: base, kind: layer.kind });
      return;
    }

    const osc = this.ctx.createOscillator();
    osc.type = synth.type;
    osc.frequency.value = synth.freq_hz ?? 110;
    osc.connect(filter);
    osc.start();
    sources.push(osc);

    if (synth.lfo_hz && synth.lfo_depth) {
      const lfo = this.ctx.createOscillator();
      const lfoGain = this.ctx.createGain();
      lfo.frequency.value = synth.lfo_hz;
      const depth =
        layer.kind === "pulse" || layer.kind === "event"
          ? base * synth.lfo_depth
          : base * synth.lfo_depth * 0.35;
      lfoGain.gain.value = depth;
      lfo.connect(lfoGain);
      lfoGain.connect(gain.gain);
      lfo.start();
      this.layers.push({
        sources,
        gain,
        filter,
        baseGain: base,
        kind: layer.kind,
        lfo,
        lfoGain,
      });
      return;
    }

    this.layers.push({ sources, gain, filter, baseGain: base, kind: layer.kind });
  }

  private async buildGraph(pack: Pack): Promise<LiveValues["source"]> {
    if (!this.ctx || !this.master) return "synth";
    const profile = pack.engine_profile;
    let stemCount = 0;
    let synthCount = 0;

    for (const layer of pack.layers) {
      let usedStem = false;
      if (layer.stem_url) {
        const buf = await this.loadStem(layer.stem_url);
        if (buf) {
          const { filter, gain, base } = this.connectLayerShell(profile, layer);
          const src = this.ctx.createBufferSource();
          src.buffer = buf;
          src.loop = layer.loop !== false;
          src.connect(filter);
          src.start();
          this.layers.push({
            sources: [src],
            gain,
            filter,
            baseGain: base,
            kind: layer.kind,
          });
          stemCount += 1;
          usedStem = true;
        }
      }
      if (!usedStem && layer.synth) {
        this.startSynthLayer(profile, layer, layer.synth);
        synthCount += 1;
      }
    }

    if (profile.space > 0.4 && this.compressor) {
      const delay = this.ctx.createDelay(1.2);
      delay.delayTime.value = 0.18 + profile.space * 0.22;
      const feedback = this.ctx.createGain();
      feedback.gain.value = 0.08 + profile.space * 0.1;
      const wet = this.ctx.createGain();
      wet.gain.value = 0.06 + profile.space * 0.1;
      const damp = this.ctx.createBiquadFilter();
      damp.type = "lowpass";
      damp.frequency.value = 2400;
      this.master.connect(delay);
      delay.connect(damp);
      damp.connect(feedback);
      feedback.connect(delay);
      damp.connect(wet);
      wet.connect(this.compressor);
      this.spaceNodes.push(delay, feedback, wet, damp);
    }

    if (stemCount > 0 && synthCount > 0) return "mixed";
    if (stemCount > 0) return "stem";
    return "synth";
  }

  private currentPhase(profile: EngineProfile, elapsed: number): {
    id: string;
    energyMul: number;
    densityMul: number;
  } {
    if (this.durationSec <= 0) {
      const sustain = profile.phases.find((p) => p.id === "sustain") ?? profile.phases[0];
      return {
        id: sustain?.id ?? "sustain",
        energyMul: sustain?.energy_mul ?? 1,
        densityMul: sustain?.density_mul ?? 1,
      };
    }
    const ratio = Math.min(1, elapsed / this.durationSec);
    let acc = 0;
    for (const p of profile.phases) {
      acc += p.duration_ratio;
      if (ratio <= acc) {
        return { id: p.id, energyMul: p.energy_mul, densityMul: p.density_mul };
      }
    }
    const last = profile.phases[profile.phases.length - 1];
    return {
      id: last?.id ?? "outro",
      energyMul: last?.energy_mul ?? 0.4,
      densityMul: last?.density_mul ?? 0.4,
    };
  }

  private tick = () => {
    if (!this.playing || !this.ctx || !this.pack) return;
    const profile = this.pack.engine_profile;
    const elapsed = this.ctx.currentTime - this.startedAt;
    const phase = this.currentPhase(profile, elapsed);

    const period = Math.max(12, profile.variation_period_sec);
    const wobble = 1 + Math.sin((elapsed / period) * Math.PI * 2) * 0.035;

    const energy = profile.energy * phase.energyMul * wobble;
    const density = profile.event_density * phase.densityMul;

    for (const layer of this.layers) {
      let mul = energy / Math.max(profile.energy, 0.05);
      if (layer.kind === "event") mul *= 0.35 + density;
      const target = layer.baseGain * mul;
      const g = layer.gain.gain;
      const now = this.ctx.currentTime;
      g.setTargetAtTime(Math.max(0.0001, target), now, 1.6);
      layer.filter.frequency.setTargetAtTime(
        260 + profile.brightness * energy * 3400,
        now,
        1.8,
      );
    }

    if (this.durationSec > 0 && elapsed >= this.durationSec) {
      void this.stop(2.5);
      this.emit({
        energy: 0,
        density: 0,
        brightness: profile.brightness,
        phase: "outro",
        source: this.lastSource,
      });
      return;
    }

    this.emit({
      energy: Math.min(1, energy),
      density: Math.min(1, density),
      brightness: profile.brightness,
      phase: phase.id,
      source: this.lastSource,
    });
    this.raf = requestAnimationFrame(this.tick);
  };
}

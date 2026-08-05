import type { EngineProfile, Pack } from "../types";

type LiveValues = {
  energy: number;
  density: number;
  brightness: number;
  phase: string;
};

type Listener = (v: LiveValues) => void;

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

  // brown
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
  private layers: LayerNodes[] = [];
  private spaceNodes: AudioNode[] = [];
  private pack: Pack | null = null;
  private startedAt = 0;
  private durationSec = 0;
  private raf = 0;
  private listeners = new Set<Listener>();
  private playing = false;

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
      this.compressor = this.ctx.createDynamicsCompressor();
      this.compressor.threshold.value = -24;
      this.compressor.knee.value = 18;
      this.compressor.ratio.value = 6;
      this.master.connect(this.compressor);
      this.compressor.connect(this.ctx.destination);
    }
    if (this.ctx.state === "suspended") await this.ctx.resume();
  }

  async play(pack: Pack, durationMin: number) {
    await this.ensureContext();
    await this.crossfadeTo(pack, durationMin);
  }

  async stop(fadeSec = 1.2) {
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
    const fade = 1.4;

    // fade out current
    this.master.gain.cancelScheduledValues(now);
    this.master.gain.setValueAtTime(Math.max(this.master.gain.value, 0.0001), now);
    this.master.gain.linearRampToValueAtTime(0.0001, now + fade);

    await new Promise((r) => setTimeout(r, fade * 1000));
    this.teardownLayers();

    this.pack = pack;
    this.durationSec = durationMin < 0 ? 0 : durationMin * 60;
    this.startedAt = this.ctx.currentTime;
    this.buildGraph(pack);
    const t = this.ctx.currentTime;
    this.master.gain.setValueAtTime(0.0001, t);
    this.master.gain.linearRampToValueAtTime(0.85, t + fade);
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

  private buildGraph(pack: Pack) {
    if (!this.ctx || !this.master) return;
    const profile = pack.engine_profile;

    for (const layer of pack.layers) {
      if (!layer.synth) continue;
      const filter = this.ctx.createBiquadFilter();
      filter.type = "lowpass";
      filter.frequency.value = 400 + profile.brightness * 6000;
      filter.Q.value = 0.7;

      const gain = this.ctx.createGain();
      const kindMul =
        layer.kind === "event"
          ? profile.event_density
          : layer.kind === "bed" || layer.kind === "noise"
            ? 0.7 + profile.masking * 0.5
            : 1;
      const base = layer.gain * (0.35 + profile.energy * 0.9) * kindMul;
      gain.gain.value = base;

      filter.connect(gain);
      gain.connect(this.master);

      const sources: AudioScheduledSourceNode[] = [];
      const synth = layer.synth;

      if (synth.type === "white" || synth.type === "pink" || synth.type === "brown") {
        const buf = createNoiseBuffer(this.ctx, synth.type);
        const src = this.ctx.createBufferSource();
        src.buffer = buf;
        src.loop = true;
        src.connect(filter);
        src.start();
        sources.push(src);
      } else {
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
          // amplitude tremolo for pulse/event; gentle for drones
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
          continue;
        }
      }

      this.layers.push({ sources, gain, filter, baseGain: base, kind: layer.kind });
    }

    if (profile.space > 0.5 && this.ctx && this.master && this.compressor) {
      const delay = this.ctx.createDelay(1.0);
      delay.delayTime.value = 0.12 + profile.space * 0.18;
      const feedback = this.ctx.createGain();
      feedback.gain.value = 0.12 + profile.space * 0.15;
      const wet = this.ctx.createGain();
      wet.gain.value = 0.08 + profile.space * 0.12;
      this.master.connect(delay);
      delay.connect(feedback);
      feedback.connect(delay);
      delay.connect(wet);
      wet.connect(this.compressor);
      this.spaceNodes.push(delay, feedback, wet);
    }
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

    // slow variation
    const period = Math.max(8, profile.variation_period_sec);
    const wobble = 1 + Math.sin((elapsed / period) * Math.PI * 2) * 0.06;

    const energy = profile.energy * phase.energyMul * wobble;
    const density = profile.event_density * phase.densityMul;

    for (const layer of this.layers) {
      let mul = energy / Math.max(profile.energy, 0.05);
      if (layer.kind === "event") mul *= 0.5 + density;
      const target = layer.baseGain * mul;
      const g = layer.gain.gain;
      const now = this.ctx.currentTime;
      g.setTargetAtTime(Math.max(0.0001, target), now, 0.8);
      layer.filter.frequency.setTargetAtTime(
        350 + profile.brightness * energy * 6500,
        now,
        0.8,
      );
    }

    // auto stop at end for timed sessions
    if (this.durationSec > 0 && elapsed >= this.durationSec) {
      void this.stop(2.5);
      this.emit({
        energy: 0,
        density: 0,
        brightness: profile.brightness,
        phase: "outro",
      });
      return;
    }

    this.emit({
      energy: Math.min(1, energy),
      density: Math.min(1, density),
      brightness: profile.brightness,
      phase: phase.id,
    });
    this.raf = requestAnimationFrame(this.tick);
  };
}

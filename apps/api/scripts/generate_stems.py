#!/usr/bin/env python3
"""Generate refined seamless preview stems for Harc (softer, more layered)."""

from __future__ import annotations

import math
import subprocess
import struct
import wave
from pathlib import Path

SR = 44100
OUT = Path(__file__).resolve().parents[1] / "assets" / "stems"
VERSION = "v2"


def clamp(x: float, lo: float = -1.0, hi: float = 1.0) -> float:
    return lo if x < lo else hi if x > hi else x


def one_pole_lp(samples: list[float], cutoff_hz: float) -> list[float]:
    # y[n] = y[n-1] + a*(x[n]-y[n-1])
    a = 1.0 - math.exp(-2.0 * math.pi * cutoff_hz / SR)
    y = 0.0
    out = []
    for x in samples:
        y += a * (x - y)
        out.append(y)
    return out


def one_pole_hp(samples: list[float], cutoff_hz: float) -> list[float]:
    lp = one_pole_lp(samples, cutoff_hz)
    return [x - y for x, y in zip(samples, lp)]


def soft_saturate(samples: list[float], drive: float = 0.35) -> list[float]:
    out = []
    for x in samples:
        out.append(math.tanh(x * (1.0 + drive)) / math.tanh(1.0 + drive))
    return out


def write_wav(path: Path, samples: list[float], peak_target: float = 0.45) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    n = len(samples)
    fade = min(int(0.25 * SR), n // 6)
    # equal-power crossfade seam
    for i in range(fade):
        w = 0.5 - 0.5 * math.cos(math.pi * i / fade)
        a = samples[i]
        b = samples[n - fade + i]
        samples[i] = a * w + b * (1 - w)
        samples[n - fade + i] = samples[i]

    peak = max(1e-12, max(abs(x) for x in samples))
    norm = peak_target / peak
    with wave.open(str(path), "w") as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(SR)
        frames = b"".join(
            struct.pack("<h", int(clamp(x * norm) * 32767)) for x in samples
        )
        wf.writeframes(frames)


def white(n: int, seed: int = 1234567) -> list[float]:
    x = seed & 0x7FFFFFFF
    out = []
    for _ in range(n):
        x = (1103515245 * x + 12345) & 0x7FFFFFFF
        out.append((x / 0x7FFFFFFF) * 2 - 1)
    return out


def pink(n: int, seed: int = 42) -> list[float]:
    w = white(n, seed)
    b0 = b1 = b2 = b3 = b4 = b5 = b6 = 0.0
    out = []
    for x in w:
        b0 = 0.99886 * b0 + x * 0.0555179
        b1 = 0.99332 * b1 + x * 0.0750759
        b2 = 0.969 * b2 + x * 0.153852
        b3 = 0.8665 * b3 + x * 0.3104856
        b4 = 0.55 * b4 + x * 0.5329522
        b5 = -0.7616 * b5 - x * 0.016898
        out.append((b0 + b1 + b2 + b3 + b4 + b5 + b6 + x * 0.5362) * 0.11)
        b6 = x * 0.115926
    return out


def brown(n: int, seed: int = 7) -> list[float]:
    w = white(n, seed)
    last = 0.0
    out = []
    for x in w:
        last = (last + 0.015 * x) / 1.015
        out.append(last * 3.2)
    return out


def soft_pad(n: int, freqs: list[tuple[float, float]], vibrato_hz: float = 0.04, vibrato_cents: float = 8.0) -> list[float]:
    """Stack soft partials with tiny vibrato — no harsh saws."""
    out = [0.0] * n
    for fi, (freq, amp) in enumerate(freqs):
        phase = 0.0
        for i in range(n):
            t = i / SR
            # slow random-ish vibrato via two sines
            cents = vibrato_cents * (
                0.6 * math.sin(2 * math.pi * vibrato_hz * t + fi)
                + 0.4 * math.sin(2 * math.pi * vibrato_hz * 0.37 * t + fi * 1.7)
            )
            inst = freq * (2 ** (cents / 1200.0))
            phase += 2 * math.pi * inst / SR
            # soft triangle-ish via sine + weak 3rd
            y = math.sin(phase) + 0.12 * math.sin(3 * phase)
            # gentle amplitude drift
            drift = 1.0 + 0.04 * math.sin(2 * math.pi * (0.02 + fi * 0.007) * t)
            out[i] += y * amp * drift
    return soft_saturate(out, 0.2)


def am_smooth(samples: list[float], lfo_hz: float, depth: float) -> list[float]:
    out = []
    for i, x in enumerate(samples):
        t = i / SR
        # raised-cosine AM — less "wobble buzz"
        lfo = 0.5 - 0.5 * math.cos(2 * math.pi * lfo_hz * t)
        env = 1.0 - depth + depth * lfo
        out.append(x * env)
    return out


def rain_texture(n: int, seed: int = 99) -> list[float]:
    """Filtered noise with sparse soft droplet clicks."""
    base = one_pole_lp(pink(n, seed), 1800)
    base = one_pole_hp(base, 220)
    clicks = [0.0] * n
    x = seed
    i = int(0.4 * SR)
    while i < n - int(0.05 * SR):
        x = (1103515245 * x + 12345) & 0x7FFFFFFF
        gap = int((0.35 + (x / 0x7FFFFFFF) * 1.8) * SR)
        dur = int((0.012 + (x % 1000) / 1000 * 0.03) * SR)
        freq = 900 + (x % 700)
        for k in range(dur):
            if i + k >= n:
                break
            env = math.exp(-k / (0.008 * SR)) * math.sin(math.pi * k / max(1, dur))
            clicks[i + k] += math.sin(2 * math.pi * freq * (k / SR)) * env * 0.35
        i += gap
    mixed = [0.82 * a + 0.18 * b for a, b in zip(base, clicks)]
    return one_pole_lp(mixed, 2400)


def soft_blips(n: int, freq: float, rate_hz: float, brightness: float = 0.35) -> list[float]:
    out = [0.0] * n
    period = max(1, int(SR / max(rate_hz, 0.008)))
    # irregularize spacing slightly
    pos = period // 3
    idx = 0
    while pos < n - int(0.2 * SR):
        blip = int((0.08 + 0.06 * math.sin(idx)) * SR)
        # soft bell: fund + quiet octave, long exp decay
        for k in range(blip):
            t = k / SR
            env = math.exp(-t * 6.5) * (0.5 - 0.5 * math.cos(math.pi * min(1.0, k / (0.01 * SR))))
            y = math.sin(2 * math.pi * freq * t)
            y += brightness * 0.25 * math.sin(2 * math.pi * freq * 2.01 * t)
            y += brightness * 0.08 * math.sin(2 * math.pi * freq * 3.02 * t)
            out[pos + k] += y * env
        jitter = int(0.15 * period * math.sin(idx * 1.7))
        pos += period + jitter
        idx += 1
    return one_pole_lp(out, 3500)


def breath_pulse(n: int, freq: float, bpm: float) -> list[float]:
    """Very soft pulse — more breath than metronome."""
    hz = bpm / 60.0
    pad = soft_pad(n, [(freq, 0.55), (freq * 1.5, 0.12), (freq * 2.0, 0.06)], vibrato_hz=0.03, vibrato_cents=5)
    return am_smooth(pad, hz, 0.42)


def seconds_for_freq(freq: float, target: float = 20.0) -> float:
    cycles = max(1, round(target * freq))
    return cycles / freq


def add(a: list[float], b: list[float], g: float = 1.0) -> list[float]:
    n = min(len(a), len(b))
    return [a[i] + b[i] * g for i in range(n)]


STEM_SPECS: dict[str, list[tuple[str, dict]]] = {
    "sleep-deep-night": [
        ("bed_brown", {"gen": "brown_bed", "sec": 24, "lp": 900}),
        ("drone_low", {"gen": "pad", "freqs": [(55, 0.55), (82.5, 0.22), (110, 0.08)], "sec": 24}),
        ("drone_sub", {"gen": "pad", "freqs": [(41.25, 0.4), (82.5, 0.18)], "sec": 24}),
        ("texture_soft", {"gen": "air", "sec": 22, "lp": 1400}),
        ("event_drop", {"gen": "blip", "freq": 196, "rate": 0.009, "bright": 0.2}),
    ],
    "sleep-soft-rain": [
        ("bed_pink", {"gen": "pink_bed", "sec": 24, "lp": 1200}),
        ("drone_warm", {"gen": "pad", "freqs": [(98, 0.45), (147, 0.15), (196, 0.06)], "sec": 22}),
        ("texture_rain", {"gen": "rain", "sec": 26}),
        ("event_drip", {"gen": "blip", "freq": 420, "rate": 0.016, "bright": 0.25}),
    ],
    "focus-clear": [
        ("bed_soft", {"gen": "pink_bed", "sec": 20, "lp": 1600}),
        ("drone_mid", {"gen": "pad", "freqs": [(110, 0.4), (165, 0.14), (220, 0.05)], "sec": 20}),
        ("pulse_soft", {"gen": "breath", "freq": 146.8, "bpm": 66, "sec": 20}),
        ("texture_air", {"gen": "air", "sec": 18, "lp": 2200}),
        ("event_chime", {"gen": "blip", "freq": 523.25, "rate": 0.022, "bright": 0.3}),
    ],
    "focus-deep-work": [
        ("bed_brown", {"gen": "brown_bed", "sec": 22, "lp": 1100}),
        ("drone_low", {"gen": "pad", "freqs": [(73.4, 0.5), (110, 0.16), (146.8, 0.05)], "sec": 22}),
        ("pulse_deep", {"gen": "breath", "freq": 98, "bpm": 58, "sec": 22}),
        ("texture_grain", {"gen": "air", "sec": 20, "lp": 1500}),
    ],
    "atmosphere-rain-cafe": [
        ("bed_pink", {"gen": "pink_bed", "sec": 24, "lp": 1500}),
        ("drone_warm", {"gen": "pad", "freqs": [(130.8, 0.4), (196, 0.14), (261.6, 0.05)], "sec": 22}),
        ("texture_rain", {"gen": "rain", "sec": 28}),
        ("event_cup", {"gen": "blip", "freq": 392, "rate": 0.028, "bright": 0.28}),
        ("event_murmur", {"gen": "murmur", "sec": 24}),
    ],
    "atmosphere-forest-dusk": [
        ("bed_air", {"gen": "air", "sec": 24, "lp": 1800}),
        ("drone_green", {"gen": "pad", "freqs": [(164.8, 0.38), (246, 0.12), (329.6, 0.05)], "sec": 22}),
        ("texture_leaves", {"gen": "leaves", "sec": 26}),
        ("event_bird", {"gen": "blip", "freq": 784, "rate": 0.024, "bright": 0.4}),
    ],
    "relax-warm-tide": [
        ("bed_brown", {"gen": "brown_bed", "sec": 26, "lp": 1000}),
        ("drone_wave", {"gen": "pad", "freqs": [(90, 0.48), (135, 0.16), (180, 0.06)], "sec": 24}),
        ("pulse_breath", {"gen": "breath", "freq": 120, "bpm": 48, "sec": 24}),
        ("texture_soft", {"gen": "air", "sec": 22, "lp": 1300}),
    ],
    "relax-still-air": [
        ("bed_air", {"gen": "air", "sec": 24, "lp": 1200}),
        ("drone_soft", {"gen": "pad", "freqs": [(108, 0.42), (162, 0.12), (216, 0.04)], "sec": 24}),
        ("texture_haze", {"gen": "pink_bed", "sec": 22, "lp": 1000}),
        ("event_bell", {"gen": "blip", "freq": 587.3, "rate": 0.012, "bright": 0.22}),
    ],
}


def render(spec: dict) -> list[float]:
    gen = spec["gen"]
    sec = float(spec.get("sec", 22))
    n = int(sec * SR)

    if gen == "pad":
        # force near-integer cycles on fundamental
        fund = spec["freqs"][0][0]
        sec = seconds_for_freq(fund, sec)
        n = int(sec * SR)
        return soft_pad(n, spec["freqs"])

    if gen == "breath":
        freq = float(spec["freq"])
        sec = seconds_for_freq(freq, sec)
        n = int(sec * SR)
        return breath_pulse(n, freq, float(spec["bpm"]))

    if gen == "blip":
        n = int(22 * SR)
        return soft_blips(n, float(spec["freq"]), float(spec["rate"]), float(spec.get("bright", 0.3)))

    if gen == "rain":
        return rain_texture(n, seed=911)

    if gen == "leaves":
        base = one_pole_lp(pink(n, 77), 2000)
        base = one_pole_hp(base, 400)
        rustle = am_smooth(base, 0.11, 0.22)
        return one_pole_lp(rustle, float(spec.get("lp", 2200)))

    if gen == "murmur":
        # far room tone: dark brown + slow AM
        m = one_pole_lp(brown(n, 13), 500)
        return am_smooth(m, 0.03, 0.25)

    if gen == "brown_bed":
        s = one_pole_lp(brown(n, 3), float(spec.get("lp", 900)))
        return soft_saturate(s, 0.15)

    if gen == "pink_bed":
        s = one_pole_lp(pink(n, 5), float(spec.get("lp", 1200)))
        return soft_saturate(s, 0.12)

    if gen == "air":
        s = one_pole_lp(pink(n, 19), float(spec.get("lp", 1400)))
        s = one_pole_hp(s, 180)
        return am_smooth(s, 0.05, 0.08)

    raise ValueError(gen)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    count = 0
    for pack_id, layers in STEM_SPECS.items():
        for layer_id, spec in layers:
            wav = OUT / pack_id / VERSION / f"{layer_id}.wav"
            opus = wav.with_suffix(".opus")
            samples = render(spec)
            write_wav(wav, samples, peak_target=0.42)
            subprocess.run(
                [
                    "ffmpeg", "-y", "-i", str(wav),
                    "-c:a", "libopus", "-b:a", "96k", "-application", "audio",
                    str(opus),
                ],
                check=True,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            wav.unlink(missing_ok=True)
            count += 1
            print(f"wrote {opus}")
    print(f"done: {count} stems -> {OUT}/{VERSION}")


if __name__ == "__main__":
    main()

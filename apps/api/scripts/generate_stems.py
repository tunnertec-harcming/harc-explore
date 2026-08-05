#!/usr/bin/env python3
"""Generate seamless preview stem WAVs for Harc packs (M1 pipeline)."""

from __future__ import annotations

import math
import os
import struct
import wave
from pathlib import Path

SR = 44100
OUT = Path(__file__).resolve().parents[1] / "assets" / "stems"


def write_wav(path: Path, samples: list[float]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    # fade splice for seamlessness
    fade = min(2048, len(samples) // 8)
    for i in range(fade):
        w = i / fade
        a = samples[i]
        b = samples[-(fade - i)]
        mixed = a * w + b * (1 - w)
        samples[i] = mixed
        samples[-(fade - i)] = mixed

    peak = max(1e-9, max(abs(x) for x in samples))
    norm = 0.7 / peak
    with wave.open(str(path), "w") as wf:
        wf.setnchannels(1)
        wf.setsampwidth(2)
        wf.setframerate(SR)
        frames = b"".join(
            struct.pack("<h", max(-32767, min(32767, int(x * norm * 32767))))
            for x in samples
        )
        wf.writeframes(frames)


def white(n: int) -> list[float]:
    # deterministic LCG noise for reproducibility
    x = 1234567
    out = []
    for _ in range(n):
        x = (1103515245 * x + 12345) & 0x7FFFFFFF
        out.append((x / 0x7FFFFFFF) * 2 - 1)
    return out


def pink(n: int) -> list[float]:
    w = white(n)
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


def brown(n: int) -> list[float]:
    w = white(n)
    last = 0.0
    out = []
    for x in w:
        last = (last + 0.02 * x) / 1.02
        out.append(last * 3.5)
    return out


def tone(n: int, freq: float, kind: str = "sine") -> list[float]:
    out = []
    for i in range(n):
        t = i / SR
        ph = 2 * math.pi * freq * t
        if kind == "sine":
            y = math.sin(ph)
        elif kind == "triangle":
            y = 2 * abs(2 * (freq * t % 1) - 1) - 1
        elif kind == "sawtooth":
            y = 2 * (freq * t % 1) - 1
        else:
            y = math.sin(ph)
        out.append(y)
    return out


def am(samples: list[float], lfo_hz: float, depth: float) -> list[float]:
    out = []
    for i, x in enumerate(samples):
        t = i / SR
        env = 1 - depth + depth * (0.5 + 0.5 * math.sin(2 * math.pi * lfo_hz * t))
        out.append(x * env)
    return out


def sparse_blips(n: int, freq: float, rate_hz: float) -> list[float]:
    out = [0.0] * n
    period = max(1, int(SR / max(rate_hz, 0.01)))
    blip = int(0.04 * SR)
    for start in range(period // 2, n - blip, period):
        for i in range(blip):
            t = i / SR
            env = math.sin(math.pi * i / blip) ** 2
            out[start + i] += math.sin(2 * math.pi * freq * t) * env * 0.8
    return out


def seconds_for_freq(freq: float, target: float = 12.0) -> float:
    # choose duration near target with integer cycles
    cycles = max(1, round(target * freq))
    return cycles / freq


STEM_SPECS: dict[str, list[tuple[str, dict]]] = {
    "sleep-deep-night": [
        ("bed_brown", {"gen": "brown", "sec": 12}),
        ("drone_low", {"gen": "tone", "freq": 55, "kind": "sine", "lfo": 0.04, "depth": 0.15}),
        ("drone_sub", {"gen": "tone", "freq": 82, "kind": "triangle", "lfo": 0.03, "depth": 0.1}),
        ("texture_soft", {"gen": "pink", "sec": 12}),
        ("event_drop", {"gen": "blip", "freq": 220, "rate": 0.012}),
    ],
    "sleep-soft-rain": [
        ("bed_pink", {"gen": "pink", "sec": 12}),
        ("drone_warm", {"gen": "tone", "freq": 98, "kind": "sine", "lfo": 0.05, "depth": 0.12}),
        ("texture_rain", {"gen": "white", "sec": 10, "lfo": 0.08, "depth": 0.25}),
        ("event_drip", {"gen": "blip", "freq": 480, "rate": 0.02}),
    ],
    "focus-clear": [
        ("bed_soft", {"gen": "pink", "sec": 10}),
        ("drone_mid", {"gen": "tone", "freq": 110, "kind": "triangle", "lfo": 0.06, "depth": 0.08}),
        ("pulse_soft", {"gen": "tone", "freq": 146, "kind": "sine", "lfo": 1.2, "depth": 0.55}),
        ("texture_air", {"gen": "white", "sec": 8}),
        ("event_chime", {"gen": "blip", "freq": 660, "rate": 0.03}),
    ],
    "focus-deep-work": [
        ("bed_brown", {"gen": "brown", "sec": 12}),
        ("drone_low", {"gen": "tone", "freq": 73, "kind": "sawtooth", "lfo": 0.04, "depth": 0.06}),
        ("pulse_deep", {"gen": "tone", "freq": 98, "kind": "triangle", "lfo": 1.06, "depth": 0.5}),
        ("texture_grain", {"gen": "pink", "sec": 10}),
    ],
    "atmosphere-rain-cafe": [
        ("bed_pink", {"gen": "pink", "sec": 12}),
        ("drone_warm", {"gen": "tone", "freq": 130, "kind": "sine", "lfo": 0.07, "depth": 0.1}),
        ("texture_rain", {"gen": "white", "sec": 10, "lfo": 0.15, "depth": 0.3}),
        ("event_cup", {"gen": "blip", "freq": 520, "rate": 0.04}),
        ("event_murmur", {"gen": "brown", "sec": 14, "lfo": 0.025, "depth": 0.7}),
    ],
    "atmosphere-forest-dusk": [
        ("bed_air", {"gen": "pink", "sec": 12}),
        ("drone_green", {"gen": "tone", "freq": 164, "kind": "triangle", "lfo": 0.05, "depth": 0.12}),
        ("texture_leaves", {"gen": "white", "sec": 10, "lfo": 0.2, "depth": 0.35}),
        ("event_bird", {"gen": "blip", "freq": 880, "rate": 0.035}),
    ],
    "relax-warm-tide": [
        ("bed_brown", {"gen": "brown", "sec": 14}),
        ("drone_wave", {"gen": "tone", "freq": 90, "kind": "sine", "lfo": 0.08, "depth": 0.35}),
        ("pulse_breath", {"gen": "tone", "freq": 120, "kind": "triangle", "lfo": 0.15, "depth": 0.6}),
        ("texture_soft", {"gen": "pink", "sec": 12}),
    ],
    "relax-still-air": [
        ("bed_air", {"gen": "pink", "sec": 12}),
        ("drone_soft", {"gen": "tone", "freq": 108, "kind": "sine", "lfo": 0.045, "depth": 0.18}),
        ("texture_haze", {"gen": "white", "sec": 10}),
        ("event_bell", {"gen": "blip", "freq": 740, "rate": 0.018}),
    ],
}


def render(spec: dict) -> list[float]:
    gen = spec["gen"]
    if gen == "tone":
        freq = float(spec["freq"])
        sec = seconds_for_freq(freq, 12.0)
        n = int(sec * SR)
        samples = tone(n, freq, spec.get("kind", "sine"))
        if "lfo" in spec:
            samples = am(samples, float(spec["lfo"]), float(spec.get("depth", 0.2)))
        return samples
    if gen == "blip":
        sec = 16.0
        n = int(sec * SR)
        return sparse_blips(n, float(spec["freq"]), float(spec["rate"]))
    sec = float(spec.get("sec", 12))
    n = int(sec * SR)
    if gen == "white":
        samples = white(n)
    elif gen == "pink":
        samples = pink(n)
    else:
        samples = brown(n)
    if "lfo" in spec:
        samples = am(samples, float(spec["lfo"]), float(spec.get("depth", 0.2)))
    return samples


def main() -> None:
    import subprocess

    OUT.mkdir(parents=True, exist_ok=True)
    count = 0
    for pack_id, layers in STEM_SPECS.items():
        for layer_id, spec in layers:
            path = OUT / pack_id / "v1" / f"{layer_id}.wav"
            samples = render(spec)
            write_wav(path, samples)
            opus = path.with_suffix(".opus")
            subprocess.run(
                ["ffmpeg", "-y", "-i", str(path), "-c:a", "libopus", "-b:a", "48k", str(opus)],
                check=True,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            path.unlink(missing_ok=True)
            count += 1
            print(f"wrote {opus}")
    print(f"done: {count} stems -> {OUT}")


if __name__ == "__main__":
    main()

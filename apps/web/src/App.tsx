import { useEffect, useMemo, useRef, useState } from "react";
import { fetchModes, fetchPacks, API_BASE } from "./api/client";
import { SoundscapeEngine } from "./audio/SoundscapeEngine";
import type { Mode, ModeId, Pack } from "./types";
import "./styles.css";

function formatDuration(min: number) {
  if (min < 0) return "∞";
  if (min >= 60) return `${min / 60}h`;
  return `${min}m`;
}

export default function App() {
  const engineRef = useRef(new SoundscapeEngine());
  const [modes, setModes] = useState<Mode[]>([]);
  const [packs, setPacks] = useState<Pack[]>([]);
  const [modeId, setModeId] = useState<ModeId>("sleep");
  const [packId, setPackId] = useState<string>("");
  const [duration, setDuration] = useState(60);
  const [playing, setPlaying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [live, setLive] = useState({
    energy: 0,
    density: 0,
    brightness: 0,
    phase: "—",
    source: "synth" as "stem" | "synth" | "mixed",
  });

  const mode = useMemo(
    () => modes.find((m) => m.id === modeId) ?? null,
    [modes, modeId],
  );
  const pack = useMemo(
    () => packs.find((p) => p.id === packId) ?? packs[0] ?? null,
    [packs, packId],
  );

  useEffect(() => {
    const engine = engineRef.current;
    return engine.subscribe(setLive);
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        setLoading(true);
        const m = await fetchModes();
        if (cancelled) return;
        setModes(m);
        const first = m[0];
        if (first) {
          setModeId(first.id);
          setDuration(first.default_duration_min);
        }
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const list = await fetchPacks(modeId);
        if (cancelled) return;
        setPacks(list);
        setPackId(list[0]?.id ?? "");
        const m = modes.find((x) => x.id === modeId);
        if (m) setDuration(m.default_duration_min);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [modeId, modes]);

  useEffect(() => {
    if (!mode) return;
    const root = document.documentElement;
    root.style.setProperty("--bg0", mode.palette.bg0);
    root.style.setProperty("--bg1", mode.palette.bg1);
    root.style.setProperty("--accent", mode.palette.accent);
    root.style.setProperty("--glow", mode.palette.glow);
  }, [mode]);

  async function togglePlay() {
    if (!pack) return;
    const engine = engineRef.current;
    try {
      setError(null);
      if (playing) {
        await engine.stop();
        setPlaying(false);
        return;
      }
      await engine.play(pack, duration);
      setPlaying(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function onSelectPack(id: string) {
    setPackId(id);
    const next = packs.find((p) => p.id === id);
    if (playing && next) {
      await engineRef.current.play(next, duration);
    }
  }

  async function onSelectMode(id: ModeId) {
    setModeId(id);
    if (playing) {
      await engineRef.current.stop(0.8);
      setPlaying(false);
    }
  }

  return (
    <div className="app">
      <div className="brand-row">
        <h1 className="brand">
          Harc<span>.</span>
        </h1>
        <div className="api-pill">Preview · {API_BASE}</div>
      </div>

      {loading && <p className="loading">正在连接云端配置…</p>}
      {error && <p className="error">{error}</p>}

      {mode && (
        <section className="hero" key={mode.id}>
          <h2 className="mode-name">{mode.name}</h2>
          <p className="tagline">{mode.tagline}</p>

          <div className="mode-tabs" role="tablist" aria-label="声景模式">
            {modes.map((m) => (
              <button
                key={m.id}
                role="tab"
                aria-selected={m.id === modeId}
                className={`mode-tab${m.id === modeId ? " active" : ""}`}
                onClick={() => void onSelectMode(m.id)}
              >
                {m.name}
              </button>
            ))}
          </div>

          <div className="controls">
            <button
              className={`play-btn${playing ? " playing" : ""}`}
              onClick={() => void togglePlay()}
              disabled={!pack}
            >
              {playing ? "暂停预览" : "开始预览"}
            </button>
            <div className="duration-group" aria-label="时长">
              {mode.duration_options_min.map((d) => (
                <button
                  key={d}
                  className={`chip${duration === d ? " active" : ""}`}
                  onClick={() => setDuration(d)}
                >
                  {formatDuration(d)}
                </button>
              ))}
            </div>
          </div>
        </section>
      )}

      <section className="packs">
        <div className="packs-label">场景包</div>
        <div className="pack-list">
          {packs.map((p) => (
            <button
              key={p.id}
              className={`pack-item${p.id === pack?.id ? " active" : ""}`}
              onClick={() => void onSelectPack(p.id)}
            >
              <strong>
                {p.name}
                {p.tier === "premium" ? " · Premium" : ""}
              </strong>
              <span>{p.description}</span>
            </button>
          ))}
        </div>
      </section>

      <section className="meter-panel" aria-label="引擎参数">
        <div className="meter">
          <label>Energy</label>
          <div className="bar">
            <i style={{ width: `${Math.round(live.energy * 100)}%` }} />
          </div>
        </div>
        <div className="meter">
          <label>Density</label>
          <div className="bar">
            <i style={{ width: `${Math.round(live.density * 100)}%` }} />
          </div>
        </div>
        <div className="meter">
          <label>Brightness</label>
          <div className="bar">
            <i
              style={{
                width: `${Math.round((pack?.engine_profile.brightness ?? live.brightness) * 100)}%`,
              }}
            />
          </div>
        </div>
        <div className="meter">
          <label>Phase / Source</label>
          <div className="phase">
            {live.phase} · {live.source}
          </div>
        </div>
      </section>
    </div>
  );
}

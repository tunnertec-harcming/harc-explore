import { useEffect, useMemo, useRef, useState } from "react";
import {
  API_BASE,
  applyProfileDelta,
  fetchInsights,
  fetchModes,
  fetchPacks,
  getDeviceId,
  getSessionId,
  personalizePack,
  postEvolutionEvents,
  recommendPack,
  type PersonalizeResult,
  type RecommendResult,
} from "./api/client";
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
  const deviceId = useMemo(() => getDeviceId(), []);
  const sessionId = useMemo(() => getSessionId(), []);
  const playStartedAt = useRef<number>(0);
  const lastPackRef = useRef<string>("");

  const [modes, setModes] = useState<Mode[]>([]);
  const [packs, setPacks] = useState<Pack[]>([]);
  const [modeId, setModeId] = useState<ModeId>("sleep");
  const [packId, setPackId] = useState<string>("");
  const [duration, setDuration] = useState(60);
  const [playing, setPlaying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [reco, setReco] = useState<RecommendResult | null>(null);
  const [persona, setPersona] = useState<PersonalizeResult | null>(null);
  const [insightLine, setInsightLine] = useState("进化未开始");
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
        const m = modes.find((x) => x.id === modeId);
        if (m) setDuration(m.default_duration_min);

        let nextId = list[0]?.id ?? "";
        try {
          const r = await recommendPack(deviceId, modeId);
          if (!cancelled) setReco(r);
          if (list.some((p) => p.id === r.pack_id)) nextId = r.pack_id;
        } catch {
          /* evolution optional */
        }
        if (!cancelled) setPackId(nextId);
      } catch (e) {
        if (!cancelled) setError(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [modeId, modes, deviceId]);

  useEffect(() => {
    if (!mode) return;
    const root = document.documentElement;
    root.style.setProperty("--bg0", mode.palette.bg0);
    root.style.setProperty("--bg1", mode.palette.bg1);
    root.style.setProperty("--accent", mode.palette.accent);
    root.style.setProperty("--glow", mode.palette.glow);
  }, [mode]);

  useEffect(() => {
    if (!playing || !pack) return;
    const timer = window.setInterval(() => {
      void postEvolutionEvents(deviceId, sessionId, [
        {
          type: "heartbeat",
          ts: Math.floor(Date.now() / 1000),
          mode: modeId,
          pack_id: pack.id,
          listened_sec: 30,
        },
      ]).catch(() => undefined);
      void refreshInsights();
    }, 30000);
    return () => clearInterval(timer);
  }, [playing, pack, deviceId, sessionId, modeId]);

  async function refreshInsights() {
    try {
      const data = await fetchInsights(deviceId);
      const entries = Object.entries(data.stats.listen_sec || {});
      if (!entries.length) {
        setInsightLine("进化未开始 · 播放后会学习偏好");
        return;
      }
      entries.sort((a, b) => b[1] - a[1]);
      const [topId, sec] = entries[0];
      const name = packs.find((p) => p.id === topId)?.name ?? topId;
      setInsightLine(`偏好学习中 · ${name} ${Math.round(sec)}s · ${reco?.source ?? "rules"}`);
    } catch {
      /* ignore */
    }
  }

  async function playPack(target: Pack, fromPackId?: string) {
    let playable = target;
    try {
      const p = await personalizePack(deviceId, target.id);
      setPersona(p);
      playable = applyProfileDelta(target, p.profile_delta);
    } catch {
      setPersona(null);
    }

    await engineRef.current.play(playable, duration);
    setPlaying(true);
    playStartedAt.current = Date.now();
    lastPackRef.current = target.id;

    const events = [];
    if (fromPackId && fromPackId !== target.id) {
      events.push({
        type: "pack_switch" as const,
        ts: Math.floor(Date.now() / 1000),
        mode: modeId,
        pack_id: target.id,
        from_pack_id: fromPackId,
        listened_sec: 0,
      });
    } else {
      events.push({
        type: "play_start" as const,
        ts: Math.floor(Date.now() / 1000),
        mode: modeId,
        pack_id: target.id,
        duration_min: duration,
      });
    }
    void postEvolutionEvents(deviceId, sessionId, events).then(() => refreshInsights());
  }

  async function togglePlay() {
    if (!pack) return;
    try {
      setError(null);
      if (playing) {
        const listened = Math.max(1, (Date.now() - playStartedAt.current) / 1000);
        await engineRef.current.stop();
        setPlaying(false);
        void postEvolutionEvents(deviceId, sessionId, [
          {
            type: "play_stop",
            ts: Math.floor(Date.now() / 1000),
            mode: modeId,
            pack_id: pack.id,
            listened_sec: listened,
          },
        ]).then(() => refreshInsights());
        return;
      }
      await playPack(pack);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  }

  async function onSelectPack(id: string) {
    const prev = packId;
    setPackId(id);
    const next = packs.find((p) => p.id === id);
    if (playing && next) {
      const listened = Math.max(1, (Date.now() - playStartedAt.current) / 1000);
      void postEvolutionEvents(deviceId, sessionId, [
        {
          type: "pack_switch",
          ts: Math.floor(Date.now() / 1000),
          mode: modeId,
          pack_id: id,
          from_pack_id: prev,
          listened_sec: listened,
        },
      ]);
      await playPack(next, prev);
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

          <p className="evo-line">
            {insightLine}
            {reco ? ` · 推荐 ${reco.reason}` : ""}
            {persona ? ` · ${persona.reason}` : ""}
          </p>
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
                {reco?.pack_id === p.id ? " · 推荐" : ""}
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

import type { Mode, ModeId, Pack } from "../types";

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API ${path} failed: ${res.status} ${body}`);
  }
  return res.json() as Promise<T>;
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API ${path} failed: ${res.status} ${text}`);
  }
  return res.json() as Promise<T>;
}

export async function fetchModes(): Promise<Mode[]> {
  const data = await getJSON<{ modes: Mode[] }>("/modes");
  return data.modes;
}

export async function fetchPacks(mode?: string): Promise<Pack[]> {
  const q = mode ? `?mode=${encodeURIComponent(mode)}` : "";
  const data = await getJSON<{ packs: Pack[] }>(`/packs${q}`);
  return data.packs;
}

export async function fetchPack(id: string): Promise<Pack> {
  return getJSON<Pack>(`/packs/${encodeURIComponent(id)}`);
}

export type EvolutionEvent = {
  type: "play_start" | "heartbeat" | "pack_switch" | "play_stop";
  ts: number;
  mode: ModeId;
  pack_id: string;
  from_pack_id?: string;
  duration_min?: number;
  listened_sec?: number;
};

export type RecommendResult = {
  pack_id: string;
  score: number;
  source: string;
  reason: string;
};

export type ProfileDelta = {
  energy: number;
  brightness: number;
  masking: number;
  space: number;
  event_density: number;
};

export type PersonalizeResult = {
  pack_id: string;
  profile_delta: ProfileDelta;
  source: string;
  reason: string;
};

export async function postEvolutionEvents(
  deviceId: string,
  sessionId: string,
  events: EvolutionEvent[],
) {
  return postJSON<{ accepted: number }>("/v1/evolution/events", {
    device_id: deviceId,
    session_id: sessionId,
    events,
  });
}

export async function recommendPack(
  deviceId: string,
  mode: ModeId,
  hourLocal = new Date().getHours(),
) {
  return postJSON<RecommendResult>("/v1/evolution/recommend", {
    device_id: deviceId,
    mode,
    hour_local: hourLocal,
  });
}

export async function personalizePack(
  deviceId: string,
  packId: string,
  hourLocal = new Date().getHours(),
) {
  return postJSON<PersonalizeResult>("/v1/evolution/personalize", {
    device_id: deviceId,
    pack_id: packId,
    hour_local: hourLocal,
  });
}

export async function fetchInsights(deviceId: string) {
  return getJSON<{
    device_id: string;
    stats: {
      listen_sec: Record<string, number>;
      play_count: Record<string, number>;
      early_switch: Record<string, number>;
    };
  }>(`/v1/evolution/insights?device_id=${encodeURIComponent(deviceId)}`);
}

export function getDeviceId() {
  const key = "harc_device_id";
  let id = localStorage.getItem(key);
  if (!id) {
    id = `web-${crypto.randomUUID()}`;
    localStorage.setItem(key, id);
  }
  return id;
}

export function getSessionId() {
  const key = "harc_session_id";
  let id = sessionStorage.getItem(key);
  if (!id) {
    id = crypto.randomUUID();
    sessionStorage.setItem(key, id);
  }
  return id;
}

export function applyProfileDelta(pack: Pack, delta: ProfileDelta): Pack {
  const clamp01 = (v: number) => Math.max(0, Math.min(1, v));
  const p = pack.engine_profile;
  return {
    ...pack,
    engine_profile: {
      ...p,
      energy: clamp01(p.energy + (delta.energy || 0)),
      brightness: clamp01(p.brightness + (delta.brightness || 0)),
      masking: clamp01(p.masking + (delta.masking || 0)),
      space: clamp01(p.space + (delta.space || 0)),
      event_density: clamp01(p.event_density + (delta.event_density || 0)),
    },
  };
}

export { API_BASE };

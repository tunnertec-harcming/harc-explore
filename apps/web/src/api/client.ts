import type { Mode, Pack } from "../types";

const API_BASE = import.meta.env.VITE_API_BASE ?? "/api";

async function getJSON<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) {
    const body = await res.text();
    throw new Error(`API ${path} failed: ${res.status} ${body}`);
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

export { API_BASE };

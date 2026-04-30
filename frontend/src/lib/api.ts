import type { AnomalyEvent, AnonymousReport, Building, SensorReading } from "@/hooks/useEnergyData";

export const API_BASE: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? "http://localhost:8080";

export const WS_URL: string = API_BASE.replace(/^http/, "ws") + "/ws";

export const READINGS_PER_HOUR = 12;

interface RawReading {
  id: number;
  building_id: string;
  timestamp: string;
  kwh: number;
  temperature: number;
  co2_ppm: number;
}

interface RawAnomaly {
  id: number;
  reading_id: number;
  detected_at: string;
  severity: "low" | "medium" | "high";
  tag?: string;
  building_id?: string;
  kwh?: number;
  temperature?: number;
  co2_ppm?: number;
}

interface RawForecastPoint {
  timestamp: string;
  predicted: number;
}

interface RawAnomalyAlert {
  type: "anomaly";
  anomaly: RawAnomaly;
  reading: RawReading;
}

export function mapReading(raw: RawReading): SensorReading {
  return {
    timestamp: new Date(raw.timestamp),
    buildingId: raw.building_id,
    kWh: raw.kwh,
    temperature: raw.temperature,
    co2: raw.co2_ppm,
  };
}

export function mapAnomaly(raw: RawAnomaly, buildings: Building[]): AnomalyEvent {
  const buildingId = raw.building_id ?? "";
  const building = buildings.find((b) => b.id === buildingId);
  const buildingName = building?.name ?? buildingId;
  const value = raw.kwh ?? 0;
  const tag = raw.tag && raw.tag !== "" ? raw.tag : undefined;
  return {
    id: String(raw.id),
    timestamp: new Date(raw.detected_at),
    buildingId,
    buildingName,
    type: "energy",
    severity: raw.severity,
    value: Math.round(value * 10) / 10,
    baseline: 0,
    zScore: 0,
    message: `${buildingName}: ${raw.severity} energy anomaly — ${Math.round(value)} kWh${tag ? ` (${tag})` : ""}`,
    tag,
    acknowledged: false,
  };
}

export function mapForecast(raw: RawForecastPoint): { timestamp: Date; predicted: number } {
  return { timestamp: new Date(raw.timestamp), predicted: raw.predicted };
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init);
  if (!res.ok) {
    throw new Error(`${init?.method ?? "GET"} ${path} → ${res.status}`);
  }
  return res.json() as Promise<T>;
}

export async function getReadings(buildingId?: string, limit = READINGS_PER_HOUR): Promise<SensorReading[]> {
  const params = new URLSearchParams();
  if (buildingId) params.set("building", buildingId);
  params.set("limit", String(limit));
  const raw = await fetchJSON<RawReading[]>(`/api/readings?${params.toString()}`);
  return raw.map(mapReading);
}

export async function getAnomalies(buildings: Building[], buildingId?: string, severity?: string): Promise<AnomalyEvent[]> {
  const params = new URLSearchParams();
  if (buildingId) params.set("building", buildingId);
  if (severity) params.set("severity", severity);
  const qs = params.toString();
  const raw = await fetchJSON<RawAnomaly[]>(`/api/anomalies${qs ? `?${qs}` : ""}`);
  return raw.map((a) => mapAnomaly(a, buildings));
}

// src/lib/api.ts update
export async function getForecast(buildingId: string, steps: number = 12): Promise<{ 
  kwh: { timestamp: Date; predicted: number }[], 
  temp: { timestamp: Date; predicted: number }[], 
  co2: { timestamp: Date; predicted: number }[] 
}> {
  // Add the steps parameter to the Go API request
  const raw = await fetchJSON<any>(`/api/forecast?building=${encodeURIComponent(buildingId)}&steps=${steps}`);
  
  // Map all three metric timelines
  return {
    kwh: (raw.kwh || []).map(mapForecast),
    temp: (raw.temp || []).map(mapForecast),
    co2: (raw.co2 || []).map(mapForecast),
  };
}

export async function postReport(buildingId: string, category: string, description: string): Promise<{ id: number; status: string }> {
  return fetchJSON<{ id: number; status: string }>(`/api/reports`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ building_id: buildingId, category, description }),
  });
}

export type { RawAnomalyAlert };

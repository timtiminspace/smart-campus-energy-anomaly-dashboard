import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import {
  getAnomalies,
  getForecast,
  getReadings,
  mapAnomaly,
  mapReading,
  postReport,
  READINGS_PER_HOUR,
  WS_URL,
  type RawAnomalyAlert,
} from "@/lib/api";
import useWebSocket from "@/hooks/useWebSocket";

export interface Building {
  id: string;
  name: string;
  area: number; // m²
  floors: number;
}

export interface SensorReading {
  timestamp: Date;
  buildingId: string;
  kWh: number;
  temperature: number;
  co2: number;
}

export interface KPIData {
  energyIntensity: number; // kWh/m²
  outOfHoursShare: number; // percentage
  avgTemperature: number;
  avgCO2: number;
  totalKWh: number;
  anomalyCount: number;
}

export interface AnomalyEvent {
  id: string;
  timestamp: Date;
  buildingId: string;
  buildingName: string;
  type: "energy" | "temperature" | "co2";
  severity: "low" | "medium" | "high";
  value: number;
  baseline: number;
  zScore: number;
  message: string;
  tag?: string;
  acknowledged: boolean;
}

export interface AnonymousReport {
  id: string;
  timestamp: Date;
  buildingId: string;
  message: string;
}

export const BUILDINGS: Building[] = [
  { id: "engineering", name: "Engineering Block", area: 4200, floors: 4 },
  { id: "library", name: "Main Library", area: 6800, floors: 3 },
  { id: "main-hall", name: "Main Hall", area: 3100, floors: 2 },
  { id: "admin", name: "Admin Building", area: 2400, floors: 3 },
  { id: "sports-hall", name: "Sports Centre", area: 5500, floors: 2 },
];

const POLL_INTERVAL_MS = 30000;
const MAX_READINGS = 2500;
const MAX_EVENTS = 200;

function dedupeReadings(arr: SensorReading[]): SensorReading[] {
  const seen = new Set<string>();
  const out: SensorReading[] = [];
  for (const r of arr) {
    const key = `${r.buildingId}-${r.timestamp.getTime()}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(r);
  }
  return out.sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime());
}

export function useEnergyData() {
  const [selectedBuilding, setSelectedBuilding] = useState<string>(BUILDINGS[0].id);
  const [timeFrame, setTimeFrame] = useState<"hour" | "day" | "week">("hour"); // New state
  const [readings, setReadings] = useState<SensorReading[]>([]);
  const [events, setEvents] = useState<AnomalyEvent[]>([]);
  const [reports, setReports] = useState<AnonymousReport[]>([]);
  const [isLive, setIsLive] = useState(true);
  
  // Updated forecast state to handle the multivariate object
  const [forecast, setForecast] = useState<{
    kwh: { timestamp: Date; predicted: number }[];
    temp: { timestamp: Date; predicted: number }[];
    co2: { timestamp: Date; predicted: number }[];
  }>({ kwh: [], temp: [], co2: [] });

  const readingsRef = useRef<SensorReading[]>([]);
  readingsRef.current = readings;

  const refreshReadings = useCallback(async () => {
    // Map timeFrame to the number of 5-minute intervals required
    const limits = { hour: 12, day: 288, week: 2016 };
    const limit = limits[timeFrame];

    try {
      const batches = await Promise.all(
        BUILDINGS.map((b) => getReadings(b.id, limit))
      );
      // Ensure we don't truncate the data we just fetched
      const flat = dedupeReadings(batches.flat()).slice(-MAX_READINGS);
      setReadings(flat);
    } catch (err) {
      console.warn("[useEnergyData] failed to refresh readings", err);
    }
  }, [timeFrame]); // Add timeFrame to dependencies

  const refreshAnomalies = useCallback(async () => {
    try {
      const evts = await getAnomalies(BUILDINGS);
      setEvents(evts.slice(0, MAX_EVENTS));
    } catch (err) {
      console.warn("[useEnergyData] failed to load anomalies", err);
    }
  }, []);

  // Track the most recent reading for the SELECTED building only
  const buildingReadings = readings.filter(r => r.buildingId === selectedBuilding);
  const latestReadingTime = buildingReadings.length > 0 
    ? buildingReadings[buildingReadings.length - 1].timestamp.getTime() 
    : 0;

  // Initial load
  useEffect(() => {
    refreshReadings();
    refreshAnomalies();
  }, [refreshReadings, refreshAnomalies]);

  // Forecast — refetch when selectedBuilding, timeFrame, OR data changes
  useEffect(() => {
    let cancelled = false;

    // 1. Map the current timeFrame to the correct number of prediction steps
    const stepsMap = { hour: 12, day: 288, week: 2016 };
    const steps = stepsMap[timeFrame];

    // 2. Pass the steps into your getForecast API call
    getForecast(selectedBuilding, steps)
      .then((fc: any) => {
        if (!cancelled) {
          // Store the full multivariate object { kwh, temp, co2 }
          setForecast({
            kwh: fc.kwh || [],
            temp: fc.temp || [],
            co2: fc.co2 || [],
          });
        }
      })
      .catch((err) => {
        if (!cancelled) {
          console.warn("[useEnergyData] failed to load forecast", err);
          setForecast({ kwh: [], temp: [], co2: [] });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [selectedBuilding, timeFrame, latestReadingTime]); // 3. Added timeFrame to dependencies!

  // Poll readings while live
  useEffect(() => {
    if (!isLive) return;
    const interval = window.setInterval(() => {
      refreshReadings();
    }, POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, [isLive, refreshReadings]);

  // WebSocket — append reading + prepend anomaly + toast on alert
  const handleWS = useCallback((payload: unknown) => {
    const msg = payload as Partial<RawAnomalyAlert>;
    if (!msg || msg.type !== "anomaly" || !msg.anomaly || !msg.reading) return;

    const reading = mapReading(msg.reading);
    const anomaly = mapAnomaly(msg.anomaly, BUILDINGS);

    setReadings((prev) => dedupeReadings([...prev, reading]).slice(-MAX_READINGS));
    setEvents((prev) => [anomaly, ...prev].slice(0, MAX_EVENTS));

    toast(`${anomaly.buildingName}: ${anomaly.severity} anomaly`, {
      description: `${Math.round(anomaly.value)} kWh${anomaly.tag ? ` · ${anomaly.tag}` : ""}`,
    });
  }, []);

  useWebSocket(WS_URL, { onMessage: handleWS, enabled: isLive });

  const getKPIs = useCallback((): KPIData => {
    const buildingReadings = readingsRef.current.filter((r) => r.buildingId === selectedBuilding);
    const building = BUILDINGS.find((b) => b.id === selectedBuilding)!;
    if (buildingReadings.length === 0) {
      return {
        energyIntensity: 0,
        outOfHoursShare: 0,
        avgTemperature: 0,
        avgCO2: 0,
        totalKWh: 0,
        anomalyCount: 0,
      };
    }

    const totalKWh = buildingReadings.reduce((a, r) => a + r.kWh, 0);
    const outOfHours = buildingReadings.filter((r) => {
      const h = r.timestamp.getHours();
      return h < 8 || h > 18;
    });
    const outOfHoursKWh = outOfHours.reduce((a, r) => a + r.kWh, 0);

    return {
      energyIntensity: Math.round((totalKWh / building.area) * 100) / 100,
      outOfHoursShare: totalKWh > 0 ? Math.round((outOfHoursKWh / totalKWh) * 1000) / 10 : 0,
      avgTemperature:
        Math.round((buildingReadings.reduce((a, r) => a + r.temperature, 0) / buildingReadings.length) * 10) / 10,
      avgCO2: Math.round(buildingReadings.reduce((a, r) => a + r.co2, 0) / buildingReadings.length),
      totalKWh: Math.round(totalKWh),
      anomalyCount: events.filter((e) => e.buildingId === selectedBuilding).length,
    };
  }, [selectedBuilding, events]);

  const tagEvent = useCallback((eventId: string, tag: string) => {
    setEvents((prev) => prev.map((e) => (e.id === eventId ? { ...e, tag } : e)));
  }, []);

  const acknowledgeEvent = useCallback((eventId: string) => {
    setEvents((prev) => prev.map((e) => (e.id === eventId ? { ...e, acknowledged: true } : e)));
  }, []);

  const submitReport = useCallback(async (buildingId: string, message: string) => {
    try {
      const res = await postReport(buildingId, "other", message);
      setReports((prev) => [
        { id: `rpt-${res.id}`, timestamp: new Date(), buildingId, message },
        ...prev,
      ]);
    } catch (err) {
      console.warn("[useEnergyData] failed to submit report", err);
      toast("Failed to submit report", { description: String(err) });
    }
  }, []);

  return {
    buildings: BUILDINGS,
    selectedBuilding,
    setSelectedBuilding,
    timeFrame,
    setTimeFrame,   
    readings,
    events,
    reports,
    isLive,
    setIsLive,
    forecast,
    getKPIs,
    tagEvent,
    acknowledgeEvent,
    submitReport,
  };
}

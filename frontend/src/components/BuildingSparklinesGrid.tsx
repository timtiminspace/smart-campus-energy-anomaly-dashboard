import { useMemo } from "react";
import { Area, AreaChart, ResponsiveContainer } from "recharts";
import type { Building, SensorReading } from "@/hooks/useEnergyData";

interface BuildingSparklinesGridProps {
  buildings: Building[];
  readings: SensorReading[];
  selectedBuilding: string;
  onSelect: (buildingId: string) => void;
}

const SPARKLINE_COLOR = "hsl(172, 66%, 50%)";
const POINTS = 12; // last hour at 5-min cadence

export default function BuildingSparklinesGrid({
  buildings,
  readings,
  selectedBuilding,
  onSelect,
}: BuildingSparklinesGridProps) {
  const seriesByBuilding = useMemo(() => {
    const map = new Map<string, { value: number }[]>();
    for (const b of buildings) {
      const series = readings
        .filter((r) => r.buildingId === b.id)
        .slice(-POINTS)
        .map((r) => ({ value: Math.round(r.kWh * 10) / 10 }));
      map.set(b.id, series);
    }
    return map;
  }, [buildings, readings]);

  return (
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-5 gap-3">
      {buildings.map((b) => {
        const series = seriesByBuilding.get(b.id) ?? [];
        const last = series.length > 0 ? series[series.length - 1].value : 0;
        const isActive = b.id === selectedBuilding;
        return (
          <button
            key={b.id}
            onClick={() => onSelect(b.id)}
            className={`text-left bg-card border rounded-lg p-3 transition-all active:scale-[0.98] ${
              isActive
                ? "border-primary/60 shadow-[0_0_0_1px_hsl(172,66%,50%,0.4)]"
                : "border-border hover:border-border/80"
            }`}
          >
            <div className="flex items-baseline justify-between mb-1">
              <span className="text-xs font-semibold truncate">{b.name}</span>
              <span className="font-mono-data text-[11px] text-muted-foreground shrink-0 ml-2">
                {last} kWh
              </span>
            </div>
            <div className="h-12">
              {series.length === 0 ? (
                <div className="h-full flex items-center justify-center text-[10px] text-muted-foreground">
                  no data
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={series} margin={{ top: 2, right: 0, bottom: 0, left: 0 }}>
                    <defs>
                      <linearGradient id={`spark-${b.id}`} x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor={SPARKLINE_COLOR} stopOpacity={0.4} />
                        <stop offset="100%" stopColor={SPARKLINE_COLOR} stopOpacity={0} />
                      </linearGradient>
                    </defs>
                    <Area
                      type="monotone"
                      dataKey="value"
                      stroke={SPARKLINE_COLOR}
                      strokeWidth={1.75}
                      fill={`url(#spark-${b.id})`}
                      isAnimationActive={false}
                      dot={false}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </div>
          </button>
        );
      })}
    </div>
  );
}

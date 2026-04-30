import { Zap, Thermometer, Wind, Clock, AlertTriangle, TrendingUp } from "lucide-react";
import type { KPIData } from "@/hooks/useEnergyData";

interface KPICardsProps {
  kpis: KPIData;
  buildingName: string;
}

const cards = [
  { key: "energyIntensity" as const, label: "Energy Intensity", unit: "kWh/m²", icon: Zap, color: "primary" },
  { key: "totalKWh" as const, label: "Total Energy", unit: "kWh", icon: TrendingUp, color: "primary" },
  { key: "avgTemperature" as const, label: "Avg Temperature", unit: "°C", icon: Thermometer, color: "warning" },
  { key: "avgCO2" as const, label: "Avg CO₂", unit: "ppm", icon: Wind, color: "muted" },
  { key: "outOfHoursShare" as const, label: "Out-of-Hours Usage", unit: "%", icon: Clock, color: "warning" },
  { key: "anomalyCount" as const, label: "Anomalies Detected", unit: "", icon: AlertTriangle, color: "destructive" },
];

const colorMap: Record<string, string> = {
  primary: "text-primary",
  warning: "text-warning",
  destructive: "text-destructive",
  muted: "text-muted-foreground",
};

const bgMap: Record<string, string> = {
  primary: "bg-primary/10",
  warning: "bg-warning/10",
  destructive: "bg-destructive/10",
  muted: "bg-muted",
};

export default function KPICards({ kpis, buildingName }: KPICardsProps) {
  return (
    <div className="space-y-3">
      <h2 className="text-sm font-medium text-muted-foreground uppercase tracking-wider">
        {buildingName} — Key Metrics
      </h2>
      <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3">
        {cards.map((card, i) => {
          const Icon = card.icon;
          const value = kpis[card.key];
          return (
            <div
              key={card.key}
              className="bg-card border border-border rounded-lg p-4 hover:border-primary/30 transition-colors duration-200 group"
              style={{ animation: `fade-up 0.5s cubic-bezier(0.16,1,0.3,1) ${i * 70}ms forwards`, opacity: 0 }}
            >
              <div className="flex items-center gap-2 mb-3">
                <div className={`p-1.5 rounded-md ${bgMap[card.color]}`}>
                  <Icon className={`w-3.5 h-3.5 ${colorMap[card.color]}`} />
                </div>
                <span className="text-xs text-muted-foreground truncate">{card.label}</span>
              </div>
              <div className="font-mono-data text-xl font-semibold text-foreground group-hover:text-primary transition-colors">
                {typeof value === "number" ? value.toLocaleString() : value}
                <span className="text-xs text-muted-foreground ml-1 font-normal">{card.unit}</span>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

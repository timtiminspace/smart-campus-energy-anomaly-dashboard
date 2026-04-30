import { useState } from "react";
import { Pause, Play } from "lucide-react";
import { useEnergyData } from "@/hooks/useEnergyData";
import KPICards from "./KPICards";
import EnergyChart from "./EnergyChart";
import EventLog from "./EventLog";
import BuildingSelector from "./BuildingSelector";
import BuildingSparklinesGrid from "./BuildingSparklinesGrid";
import AnonymousReportForm from "./AnonymousReportForm";
import Layout from "./Layout";

export default function Dashboard() {
  const {
    buildings,
    selectedBuilding,
    setSelectedBuilding,
    timeFrame,      
    setTimeFrame,   
    readings,
    events,
    isLive,
    setIsLive,
    forecast,
    getKPIs,
    tagEvent,
    acknowledgeEvent,
    submitReport,
  } = useEnergyData();

  const [chartMetric, setChartMetric] = useState<"kWh" | "temperature" | "co2">("kWh");
  const kpis = getKPIs();
  const building = buildings.find((b) => b.id === selectedBuilding)!;

  // Actual liveControls JSX
  const liveControls = (
    <>
      <div className="flex items-center gap-1.5">
        <div className={`w-2 h-2 rounded-full ${isLive ? "bg-success animate-pulse-dot" : "bg-muted-foreground"}`} />
        <span className="font-mono-data text-xs text-muted-foreground">{isLive ? "LIVE" : "PAUSED"}</span>
      </div>
      <button
        onClick={() => setIsLive(!isLive)}
        className="flex items-center gap-1.5 text-xs px-3 py-1.5 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors active:scale-[0.97]"
      >
        {isLive ? <Pause className="w-3 h-3" /> : <Play className="w-3 h-3" />}
        {isLive ? "Pause" : "Resume"}
      </button>
    </>
  );

  return (
    <Layout headerControls={liveControls}>
      <BuildingSelector buildings={buildings} selected={selectedBuilding} onSelect={setSelectedBuilding} />
      
      {/* Time Frame Toggles */}
      <div className="flex items-center gap-2 mb-4 bg-card border border-border p-1 rounded-lg w-fit">
        {(["hour", "day", "week"] as const).map((tf) => (
          <button
            key={tf}
            onClick={() => setTimeFrame(tf)}
            className={`text-[10px] uppercase px-3 py-1 rounded-md transition-all ${
              timeFrame === tf 
                ? "bg-primary/15 text-primary font-bold" 
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {tf}
          </button>
        ))}
      </div>

      <BuildingSparklinesGrid
        buildings={buildings}
        readings={readings}
        selectedBuilding={selectedBuilding}
        onSelect={setSelectedBuilding}
      />

      <KPICards kpis={kpis} buildingName={building.name} />

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-2 space-y-4">
          <div className="flex items-center gap-1 bg-card border border-border rounded-lg p-1 w-fit">
            {(["kWh", "temperature", "co2"] as const).map((m) => (
              <button
                key={m}
                onClick={() => setChartMetric(m)}
                className={`text-xs px-3 py-1.5 rounded-md transition-all active:scale-[0.97] ${
                  chartMetric === m
                    ? "bg-primary/15 text-primary font-medium"
                    : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {m === "kWh" ? "Energy" : m === "temperature" ? "Temperature" : "CO₂"}
              </button>
            ))}
          </div>

          <EnergyChart 
            readings={readings} 
            forecast={forecast} 
            selectedBuilding={selectedBuilding} 
            metric={chartMetric} 
            timeFrame={timeFrame}
          />

          <AnonymousReportForm buildings={buildings} onSubmit={submitReport} />
        </div>

        <div className="xl:col-span-1 min-h-[500px]">
          <EventLog 
            events={events} 
            onTag={tagEvent} 
            onAcknowledge={acknowledgeEvent} 
            selectedBuilding={selectedBuilding} 
          />
        </div>
      </div>
    </Layout>
  );
}
import { useState } from "react";
import Layout from "@/components/Layout";
import EventLog from "@/components/EventLog";
import BuildingSelector from "@/components/BuildingSelector";
import { useEnergyData } from "@/hooks/useEnergyData";

export default function Log() {
  const { buildings, events, tagEvent, acknowledgeEvent } = useEnergyData();
  const [selectedBuilding, setSelectedBuilding] = useState<string>(buildings[0]?.id ?? "");

  return (
    <Layout>
      <BuildingSelector buildings={buildings} selected={selectedBuilding} onSelect={setSelectedBuilding} />
      <div className="min-h-[600px]">
        <EventLog
          events={events}
          onTag={tagEvent}
          onAcknowledge={acknowledgeEvent}
          selectedBuilding={selectedBuilding}
        />
      </div>
    </Layout>
  );
}

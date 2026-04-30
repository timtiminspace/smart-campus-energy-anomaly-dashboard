import { useState, useEffect } from "react";
import { AlertTriangle, Check, Tag, Filter, Download, Search } from "lucide-react";
import type { AnomalyEvent } from "@/hooks/useEnergyData";

interface EventLogProps {
  events: AnomalyEvent[];
  onTag: (eventId: string, tag: string) => void;
  onAcknowledge: (eventId: string) => void;
  selectedBuilding: string;
}

const TAGS = ["HVAC Fault", "Lighting Left On", "Equipment Malfunction", "Weather-Related", "Holiday Spike", "False Positive"];

const severityStyles: Record<string, string> = {
  high: "bg-destructive/15 text-destructive border-destructive/30",
  medium: "bg-warning/10 text-warning border-warning/20",
  low: "bg-primary/10 text-primary border-primary/20",
};

const severityDot: Record<string, string> = {
  high: "bg-destructive animate-pulse-dot",
  medium: "bg-warning",
  low: "bg-primary",
};

export default function EventLog({ events, onTag, onAcknowledge, selectedBuilding }: EventLogProps) {
  const [filter, setFilter] = useState<"all" | "building">("building");
  const [severityFilter, setSeverityFilter] = useState<"all" | "low" | "medium" | "high">("all"); // TASK 7: Severity state
  const [searchQuery, setSearchQuery] = useState("");
  const [tagDropdown, setTagDropdown] = useState<string | null>(null);
  
  // Local state to merge prop events with live WebSocket events
  const [liveEvents, setLiveEvents] = useState<AnomalyEvent[]>(events);

  // Sync with parent state if it changes massively
  useEffect(() => {
    setLiveEvents(events);
  }, [events]);

  // TASK 9: WebSocket Connection
  useEffect(() => {
    const ws = new WebSocket("ws://localhost:8080/ws");
    
    ws.onmessage = (event) => {
      try {
        const newAnomaly = JSON.parse(event.data);
        // Format timestamp since it comes as a string from JSON
        newAnomaly.timestamp = new Date(newAnomaly.timestamp);
        
        // Add new anomaly to the top of the list
        setLiveEvents((prev) => [newAnomaly, ...prev]);
      } catch (err) {
        console.error("Failed to parse WebSocket message", err);
      }
    };

    return () => ws.close(); // Cleanup on unmount
  }, []);

  // TASK 8: PATCH Tag Edit
  const handleTagSelect = async (eventId: string, tag: string) => {
    try {
      await fetch(`http://localhost:8080/api/anomalies/${eventId}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tag }),
      });
      onTag(eventId, tag); // Update parent hook
      
      // Update local live state optimistically
      setLiveEvents(prev => prev.map(e => e.id === eventId ? { ...e, tag } : e));
      setTagDropdown(null);
    } catch (err) {
      console.error("Failed to update tag:", err);
    }
  };

  const filteredEvents = liveEvents
    .filter((e) => (filter === "building" ? e.buildingId === selectedBuilding : true))
    .filter((e) => severityFilter === "all" || e.severity === severityFilter) // TASK 7: Apply filter
    .filter((e) => searchQuery === "" || e.message.toLowerCase().includes(searchQuery.toLowerCase()) || (e.tag && e.tag.toLowerCase().includes(searchQuery.toLowerCase())));

  const exportCSV = () => {
    const header = "Timestamp,Building,Type,Severity,Value,Baseline,Z-Score,Tag,Message\n";
    const rows = filteredEvents
      .map((e) => `"${e.timestamp.toISOString()}","${e.buildingName}","${e.type}","${e.severity}",${e.value},${e.baseline},${e.zScore},"${e.tag || ""}","${e.message}"`)
      .join("\n");
    const blob = new Blob([header + rows], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `anomaly-log-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="bg-card border border-border rounded-lg flex flex-col h-full">
      <div className="flex items-center justify-between p-4 border-b border-border">
        <div className="flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 text-warning" />
          <h3 className="text-sm font-semibold">Event Log</h3>
          <span className="font-mono-data text-xs text-muted-foreground bg-muted px-2 py-0.5 rounded-full">
            {filteredEvents.length}
          </span>
        </div>
        <div className="flex items-center gap-1.5">
          {/* TASK 7: Severity Dropdown UI */}
          <select
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value as any)}
            className="text-xs px-2 py-1.5 rounded-md bg-secondary text-secondary-foreground border-none focus:ring-1 focus:ring-primary/50 cursor-pointer"
          >
            <option value="all">All Severities</option>
            <option value="high">High</option>
            <option value="medium">Medium</option>
            <option value="low">Low</option>
          </select>
          
          <button
            onClick={() => setFilter(filter === "all" ? "building" : "all")}
            className="flex items-center gap-1 text-xs px-2.5 py-1.5 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors active:scale-[0.97]"
          >
            <Filter className="w-3 h-3" />
            {filter === "all" ? "All" : "Building"}
          </button>
          <button
            onClick={exportCSV}
            className="flex items-center gap-1 text-xs px-2.5 py-1.5 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors active:scale-[0.97]"
          >
            <Download className="w-3 h-3" />
            CSV
          </button>
        </div>
      </div>

      <div className="px-4 py-2 border-b border-border">
        <div className="relative">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Search events..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-muted border-none rounded-md pl-8 pr-3 py-1.5 text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary/50"
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto min-h-0 divide-y divide-border">
        {filteredEvents.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <AlertTriangle className="w-8 h-8 mb-2 opacity-30" />
            <p className="text-sm">No anomalies detected</p>
            <p className="text-xs mt-1">Events will appear here in real-time</p>
          </div>
        ) : (
          filteredEvents.map((event) => (
            <div
              key={event.id}
              className={`px-4 py-3 hover:bg-muted/50 transition-colors ${event.acknowledged ? "opacity-50" : ""}`}
              style={{ animation: "slide-in-right 0.3s cubic-bezier(0.16,1,0.3,1) forwards" }}
            >
              <div className="flex items-start gap-2.5">
                <div className={`w-2 h-2 rounded-full mt-1.5 shrink-0 ${severityDot[event.severity]}`} />
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className={`text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded border ${severityStyles[event.severity]}`}>
                      {event.severity}
                    </span>
                    <span className="text-[10px] uppercase tracking-wider text-muted-foreground font-medium">{event.type}</span>
                    <span className="font-mono-data text-[10px] text-muted-foreground ml-auto shrink-0">
                      {event.timestamp.toLocaleTimeString()}
                    </span>
                  </div>
                  <p className="text-xs text-foreground/80 leading-relaxed break-words">{event.message}</p>
                  <div className="flex items-center gap-2 mt-2">
                    {event.tag && (
                      <span className="text-[10px] bg-primary/10 text-primary px-2 py-0.5 rounded-full">{event.tag}</span>
                    )}
                    <div className="relative ml-auto flex items-center gap-1">
                      <button
                        onClick={() => setTagDropdown(tagDropdown === event.id ? null : event.id)}
                        className="p-1 rounded hover:bg-muted transition-colors active:scale-95"
                        title="Tag event"
                      >
                        <Tag className="w-3 h-3 text-muted-foreground" />
                      </button>
                      {!event.acknowledged && (
                        <button
                          onClick={() => onAcknowledge(event.id)}
                          className="p-1 rounded hover:bg-muted transition-colors active:scale-95"
                          title="Acknowledge"
                        >
                          <Check className="w-3 h-3 text-muted-foreground" />
                        </button>
                      )}
                      {tagDropdown === event.id && (
                        <div className="absolute right-0 top-6 z-10 bg-popover border border-border rounded-md shadow-lg py-1 min-w-[160px]">
                          {TAGS.map((tag) => (
                            <button
                              key={tag}
                              onClick={() => handleTagSelect(event.id, tag)} // Replaced with fetch call
                              className="w-full text-left px-3 py-1.5 text-xs hover:bg-muted transition-colors text-foreground"
                            >
                              {tag}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
import { Building2 } from "lucide-react";
import type { Building } from "@/hooks/useEnergyData";

interface BuildingSelectorProps {
  buildings: Building[];
  selected: string;
  onSelect: (id: string) => void;
}

export default function BuildingSelector({ buildings, selected, onSelect }: BuildingSelectorProps) {
  return (
    <div className="flex items-center gap-1.5 flex-wrap">
      {buildings.map((b) => (
        <button
          key={b.id}
          onClick={() => onSelect(b.id)}
          className={`flex items-center gap-1.5 text-xs px-3 py-2 rounded-lg border transition-all duration-200 active:scale-[0.97] ${
            selected === b.id
              ? "bg-primary/10 border-primary/40 text-primary glow-teal"
              : "bg-card border-border text-muted-foreground hover:text-foreground hover:border-border"
          }`}
        >
          <Building2 className="w-3 h-3" />
          {b.name}
        </button>
      ))}
    </div>
  );
}

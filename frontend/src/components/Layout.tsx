import type { ReactNode } from "react";
import { NavLink } from "react-router-dom";
import { Activity } from "lucide-react";

interface LayoutProps {
  children: ReactNode;
  headerControls?: ReactNode;
}

const navLinkClass = ({ isActive }: { isActive: boolean }) =>
  `text-xs px-3 py-1.5 rounded-md transition-colors ${
    isActive
      ? "bg-primary/15 text-primary font-medium"
      : "text-muted-foreground hover:text-foreground hover:bg-secondary/60"
  }`;

export default function Layout({ children, headerControls }: LayoutProps) {
  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur-md">
        <div className="max-w-[1600px] mx-auto px-4 sm:px-6 py-3 flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="p-1.5 rounded-lg bg-primary/10">
              <Activity className="w-5 h-5 text-primary" />
            </div>
            <div>
              <h1 className="text-base font-bold leading-none">EnergyPulse</h1>
              <p className="text-[10px] text-muted-foreground mt-0.5">Campus Energy Intelligence</p>
            </div>
          </div>
          <nav className="hidden sm:flex items-center gap-1">
            <NavLink to="/" end className={navLinkClass}>
              Dashboard
            </NavLink>
            <NavLink to="/log" className={navLinkClass}>
              Event Log
            </NavLink>
            <NavLink to="/report" className={navLinkClass}>
              Report
            </NavLink>
          </nav>
          <div className="flex items-center gap-3">{headerControls}</div>
        </div>
      </header>
      <main className="max-w-[1600px] mx-auto px-4 sm:px-6 py-6 space-y-6">{children}</main>
    </div>
  );
}

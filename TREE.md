# Project File Tree

```
smart-campus-energy-anomaly/
├── cmd/
│   └── server/
│       └── main.go                  # Server entrypoint — wires DB, cron, routes
├── internal/
│   ├── database/
│   │   ├── database.go              # InitDB — open SQLite + run embedded schema
│   │   ├── migrate.go               # RunMigrations helper (golang-migrate)
│   │   └── schema.sql               # Embedded DDL (readings, anomalies, reports)
│   ├── models/
│   │   └── models.go                # Reading, Anomaly, Report, AnomalyAlert structs
│   ├── handlers/
│   │   └── handlers.go              # REST endpoints + linear-regression forecast
│   ├── workers/
│   │   ├── ingestor.go              # Mock IoT data generator (cron every 5 min)
│   │   └── detector.go              # Z-score anomaly detection + WS broadcast
│   ├── integrations/
│   │   ├── weather.go               # Open-Meteo API client (daily cache)
│   │   └── holidays.go              # UK public holidays API client (yearly cache)
│   └── ws/
│       └── hub.go                   # WebSocket hub — register, unregister, broadcast
├── database/
│   └── migrations/
│       ├── 000001_init_schema.up.sql
│       ├── 000001_init_schema.down.sql
│       └── schema.sql
├── frontend/
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── index.css
│   │   ├── App.css
│   │   ├── components/
│   │   │   ├── Dashboard.tsx
│   │   │   ├── EventLog.tsx
│   │   │   ├── KPICards.tsx
│   │   │   ├── EnergyChart.tsx
│   │   │   ├── BuildingSparklinesGrid.tsx
│   │   │   ├── BuildingSelector.tsx
│   │   │   ├── AnonymousReportForm.tsx
│   │   │   ├── Layout.tsx
│   │   │   ├── NavLink.tsx
│   │   │   └── ui/                  # shadcn/ui component library
│   │   ├── hooks/
│   │   │   ├── useEnergyData.ts
│   │   │   ├── useWebSocket.js
│   │   │   ├── use-toast.ts
│   │   │   └── use-mobile.tsx
│   │   ├── pages/
│   │   │   ├── index.tsx            # Dashboard page
│   │   │   ├── Log.tsx              # Event log page
│   │   │   ├── Report.tsx           # Anonymous report page
│   │   │   └── NotFound.tsx
│   │   ├── lib/
│   │   │   ├── api.ts
│   │   │   └── utils.ts
│   │   └── test/
│   │       ├── setup.ts
│   │       └── example.test.ts
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   └── index.html
├── data/                            # Runtime — SQLite DB (gitignored)
├── go.mod
├── go.sum
├── run.sh                           # Dev launcher — starts backend + frontend
├── CLAUDE.md                        # Project specification
├── Project Checklist.md
├── TREE.md
├── README.md
└── .gitignore
```

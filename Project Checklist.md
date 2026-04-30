# Smart Campus Anomaly Dashboard — Project Checklist

## Current Status Summary

> **Backend: ✅ Fully implemented and compiling**
> All Go backend code lives under `cmd/server/main.go` + `internal/` (module: `smart-campus-dashboard`).
> The `backend/` directory contains stale stub code from an earlier layout and should be removed.

---

## Remaining Work — In Order

### Phase 1 — Wire frontend to real backend (biggest unlock)
- [x] 1. Build `frontend/src/hooks/useWebSocket.js` — connect, exponential-backoff reconnect, JSON parse, unmount cleanup
- [x] 2. Replace mock simulation in `frontend/src/hooks/useEnergyData.ts` with real `GET /api/readings` fetch
- [x] 3. Wire one live sparkline per building to WebSocket-driven state (kWh, last hour)
- [x] 4. Add anomaly toast/banner that fires on WebSocket payload arrival
- [x] 5. Wire `components/ForecastChart.jsx` to `GET /api/forecast`
- [x] 6. Confirm all three routes render: `/` (Dashboard), `/log` (Event Log), `/report` (Report)

### Phase 2 — Close remaining frontend gaps
- [x] 7. Event Log: severity filter (low / medium / high)
- [x] 8. Event Log: inline tag editing → `PATCH /api/anomalies/:id`
- [x] 9. Event Log: new anomalies appear live via WebSocket (no refresh)
- [x] 10. Report form: category dropdown (lighting, HVAC, other)
- [x] 11. Report form: submit button posts to `POST /api/reports`

### Phase 3 — Backend tests
- [x] 12. Unit tests: z-score, rolling average, threshold logic (incl. out-of-hours + holiday), linear regression
- [x] 13. `go vet` + `gofmt` clean across the repo
- [x] 14. End-to-end test: ingestor run → anomaly detected → DB record present
- [x] 15. API response tests for every REST endpoint
- [x] 16. Verify zero PII stored after a report submission

### Phase 4 — Critical performance gate (success criterion)
- [x] 17. Write WebSocket latency load test (spike injection → client receipt)
- [x] 18. Confirm sub-2-second alert delivery under load — **must pass before Final Audit**

### Phase 5 — Dockerise
- [x] 19. `Dockerfile` (root) — multi-stage Go builder → Alpine runtime (Go code lives at root, not in `backend/`)
- [x] 20. `frontend/Dockerfile` — Vite build served via Nginx (with SPA fallback)
- [x] 21. `docker-compose.yml` — backend + frontend + ML wired, named SQLite volume `campus-db`
- [x] 22. `docker compose up` brings up the full stack with no extra steps

### Phase 6 — Surrey VM deploy
> Target: Ubuntu 24.04 VM at `user@10.2.8.118` (SSH from Heron/Otter only).
> Public URL: `https://com2042-hendrixx.csee.surrey.ac.uk` — Surrey gateway
> terminates TLS and forwards a single HTTP port (3000) to the VM.
> Code lives in **Surrey GitLab** (no backups on the VM).
> See `VM Connection Details.md` for the full brief.
- [ ] 23. Push repo to Surrey GitLab and configure a Personal Access Token for VM clone access
- [ ] 24. SSH in (`ssh user@10.2.8.118`) and install Docker Engine + Compose plugin
- [ ] 25. `git clone` the repo onto the VM and run `docker compose up -d --build`
- [ ] 26. Confirm `https://com2042-hendrixx.csee.surrey.ac.uk` reaches the dashboard, REST endpoints (`/api/*`) and the WebSocket (`/ws`) end-to-end (only port 3000 is exposed; nginx in the frontend container reverse-proxies to backend + ML)
- [ ] 27. Verify SQLite persists across `docker compose down && up` (named volume `campus-db`)
- [ ] 28. Configure VM-side process to bring the stack back up on reboot (the per-service `restart: unless-stopped` covers container crashes; Docker's own systemd unit covers reboots)
- [ ] 29. Local `docker compose up` fallback tested as Final Audit backup

### Phase 7 — Repo hygiene
- [ ] 30. Remove stale `backend/` directory (code lives under root `cmd/` + `internal/`)
- [ ] 31. Branch protection enabled on `main` (Surrey GitLab)
- [ ] 32. MR-only workflow with at least one peer review required
- [ ] 33. No committed secrets — confirm all API keys via env vars
- [ ] 34. All commits reference a task ID from the project schedule

### Phase 8 — Final QA sign-off
- [ ] 35. UX review by QA Lead (Mubeen)
- [ ] 36. Marking rubric criteria verified end-to-end
- [ ] 37. Virtual sensor stream validated end-to-end
- [ ] 38. External API integrations (weather + holidays) verified live
- [ ] 39. Scope confirmed — no out-of-scope features included

---

## Backend (Go) — Implementation Details

> **All items below are ✅ implemented, compiling, and working.**
> Source: `cmd/server/main.go`, `internal/{database,models,handlers,workers,ws,integrations}/`

### Database
- [x] `internal/database/schema.sql` — embedded schema with three tables (`readings`, `anomalies`, `reports`)
- [x] `internal/database/database.go` — `InitDB()` opens SQLite, runs embedded schema on startup via `//go:embed`
- [x] `internal/database/migrate.go` — additional `RunMigrations()` helper using `golang-migrate`
- [x] No PII fields exist anywhere in the schema (no name, email, IP)
- [x] Indexes on `readings(building_id, timestamp)`, `reports(building_id, created_at)`, `anomalies(reading_id)`

### Models (`internal/models/models.go`)
- [x] `Reading` struct — ID, BuildingID, Timestamp, KWh, Temperature, CO2PPM
- [x] `Anomaly` struct — ID, ReadingID, DetectedAt, Severity, Tag + JOIN fields (BuildingID, KWh, etc.)
- [x] `Report` struct — ID, BuildingID, CreatedAt, Category, Description
- [x] `AnomalyAlert` struct — Type, Anomaly, Reading (used for WebSocket broadcast payloads)

### Mock IoT Ingestor (`internal/workers/ingestor.go`)
- [x] Cron job runs every 5 minutes (via `robfig/cron/v3`)
- [x] Generates readings for 5 buildings: `library`, `sports-hall`, `main-hall`, `engineering`, `admin`
- [x] Base ranges: kWh 10–200, temperature 18–28°C, CO2 400–1200 ppm
- [x] Occupancy factor of 0.25× applied during out-of-hours and holidays (reduces baselines)
- [x] Injects a kWh spike ~15% of the time (1.5–2.0× multiplier) to trigger anomaly detection
- [x] Writes each reading to the `readings` table
- [x] Seeds an initial batch on startup (goroutine in `main.go`)

### Anomaly Detector (`internal/workers/detector.go`)
- [x] Runs inline after every reading insertion (called from `Ingestor.Run()`)
- [x] Queries last 12 kWh readings per building
- [x] Computes rolling average (mean) for kWh
- [x] Computes z-score: `|value − mean| / stdDev`
- [x] Flags anomaly if z-score > threshold OR value > 150% of rolling average
- [x] Default threshold = 2.5; lowered to 2.0 outside 08:00–18:00 or on public holidays
- [x] Severity classification: z ≥ 4 → "high", z ≥ 3 → "medium", else → "low"
- [x] Tags anomalies: `holiday-spike` if holiday, `out-of-hours` if outside business hours
- [x] Writes anomaly record to `anomalies` table
- [x] Broadcasts `AnomalyAlert` to WebSocket hub immediately after writing

### WebSocket Hub (`internal/ws/hub.go`)
- [x] Maintains registry of all connected `Client` instances
- [x] Upgrades HTTP connection to WebSocket on `GET /ws` via gorilla/websocket
- [x] Channel-based architecture: register, unregister, broadcast channels
- [x] Broadcasts JSON anomaly payloads to all connected clients
- [x] Handles client disconnect gracefully (removes from registry, closes send channel)
- [x] `readPump` / `writePump` goroutines per client for concurrent I/O
- [ ] Alert reaches client within **2 seconds** of detection (needs load test — Phase 4)

### REST Endpoints (`internal/handlers/handlers.go`)
- [x] `GET /api/readings?building=X&limit=N` — returns last N readings (default 60, max 1000), filterable by building
- [x] `GET /api/anomalies?building=X&severity=Y` — returns anomaly log with JOIN on readings, filterable
- [x] `PATCH /api/anomalies?id=X` — updates tag on an anomaly (body: `{"tag": "..."}`)
- [x] `POST /api/reports` — accepts anonymous staff report, validates category + description, **no IP or headers logged**
- [x] `GET /api/forecast?building=X` — linear regression predictions (default building: library)
- [x] `GET /ws` — WebSocket upgrade endpoint
- [x] CORS middleware allowing `*` origin with `GET, POST, PATCH, OPTIONS` methods

### ML Forecasting (in `handlers.go` → `GetForecast`)
- [x] Pulls last 144 readings per building (24 hrs at 5-min intervals)
- [x] Implements linear regression: `x = Unix timestamp`, `y = kWh`
- [x] Computes slope and intercept using least-squares formula
- [x] Returns predictions for next 12 steps (1 hour ahead, 5-min intervals)
- [x] Clamps negative predictions to zero
- [x] Returns RFC3339 timestamps for each prediction
- [x] Implemented in pure Go — no external ML library

### External API Integrations
- [x] `internal/integrations/weather.go` — `WeatherClient` fetches current temperature from Open-Meteo (Guildford coordinates)
- [x] `internal/integrations/holidays.go` — `HolidayClient` fetches UK public holidays from `date.nager.at`
- [x] Both APIs cached server-side per day (in-memory `map[string]` keyed by date string)
- [x] Holiday client caches all holidays for the year in one call
- [x] Thread-safe with `sync.RWMutex`

### Privacy
- [x] `POST /api/reports` does not log `r.RemoteAddr`
- [x] No client headers are stored or logged for any endpoint
- [x] `reports` table has no PII fields at schema level (no name, email, IP)

### Server Entrypoint (`cmd/server/main.go`)
- [x] Initialises SQLite database with schema
- [x] Creates WebSocket hub and starts it in a goroutine
- [x] Creates weather + holiday clients
- [x] Creates ingestor with all dependencies injected
- [x] Seeds initial readings on startup
- [x] Schedules cron job for every 5 minutes
- [x] Registers all HTTP routes on `http.ServeMux`
- [x] CORS middleware wrapping all routes
- [x] Configurable port via `PORT` env var (default: 8080)
- [x] Configurable DB path via `DB_PATH` env var (default: `./data/campus.db`)

---

## Frontend (React)

### App Shell
- [x] Vite + React project scaffolded (TypeScript)
- [x] `react-router-dom` installed and routing configured
- [x] `recharts` installed for charts
- [x] Three routes: `/` (Dashboard), `/log` (Event Log), `/report` (Report)

### WebSocket Hook (`hooks/useWebSocket.js`)
- [x] Connects to backend WebSocket on component mount
- [x] Reconnects automatically with exponential back-off on disconnect
- [x] Parses incoming JSON and dispatches to state/context
- [x] Cleans up connection on unmount

### Dashboard Page (`pages/Dashboard.jsx`)
- [x] KPI card: total kWh today
- [x] KPI card: average CO2 across buildings
- [x] KPI card: out-of-hours usage share (%)
- [x] KPI card: energy use intensity (kWh/m²)
- [x] One live sparkline chart per building (kWh over last hour)
- [x] Sparklines update in real time via WebSocket
- [x] Anomaly toast/banner appears within 2 seconds of a WebSocket alert arriving

### Event Log Page (`pages/EventLog.jsx`)
- [x] Table showing all anomalies: timestamp, building, severity, tag
- [x] Filter by building
- [ ] Filter by severity (low / medium / high)
- [ ] Inline tag editing — sends `PATCH /api/anomalies/:id` to backend
- [ ] New anomalies appear live via WebSocket without page refresh

### Report Page (`pages/Report.jsx`)
- [x] Building selector dropdown
- [ ] Category dropdown (e.g. lighting, HVAC, other)
- [x] Free-text description field
- [ ] Submit button posts to `POST /api/reports`
- [x] No user identifier, login, or session is sent with the request
- [x] Success/error feedback shown after submission

### Forecast Chart (`components/ForecastChart.jsx`)
- [x] Fetches data from `GET /api/forecast`
- [x] Renders historical kWh alongside predicted values on one chart
- [x] Visually distinguishes historical vs. forecast data (e.g. dashed line)

---

## DevOps & Deployment

### Docker
- [x] `Dockerfile` for backend — multi-stage build (Go builder → Alpine runtime)
- [x] `Dockerfile` for frontend — builds React app, serves via Nginx
- [x] `Dockerfile` for ML sidecar — FastAPI + CPU-only PyTorch
- [x] `docker-compose.yml` — wires backend + frontend + ML, mounts SQLite volume
- [x] `docker compose up` runs the full stack locally without extra steps

### Surrey VM deploy
- [ ] Repo pushed to **Surrey GitLab** with a Personal Access Token configured for VM clone access
- [ ] Docker Engine + Compose plugin installed on the Ubuntu 24.04 VM (`user@10.2.8.118`)
- [ ] `docker compose up -d --build` running on the VM with `restart: unless-stopped` on every service
- [ ] Only host port **3000** is published — frontend nginx reverse-proxies `/api/*` and `/ws` to the backend service over the docker network (single port matches the Surrey gateway's forwarding rule)
- [ ] `https://com2042-hendrixx.csee.surrey.ac.uk` reaches the dashboard from a Heron/Otter machine; REST + WebSocket both work end-to-end
- [ ] Named volume `campus-db` keeps SQLite data across container restarts
- [ ] Local `docker compose up` fallback verified as Final Audit backup

### Surrey GitLab
- [ ] Remove stale `backend/` directory (duplicate code from old structure)
- [ ] Branch protection enabled on `main`
- [ ] All changes go through Merge Requests
- [ ] At least one peer review required before merge
- [ ] No secrets committed — API keys via environment variables only
- [ ] All commits reference a task ID from the project schedule

---

## Testing & QA

### Backend Unit Tests
- [ ] Unit tests for z-score calculation (`internal/workers/detector.go`)
- [ ] Unit tests for rolling average calculation
- [ ] Unit tests for anomaly threshold logic (including out-of-hours and holiday adjustments)
- [ ] Unit tests for linear regression slope + intercept (`handlers.go → GetForecast`)
- [ ] All tests pass `go vet` and `gofmt`

### Integration & System Tests
- [ ] End-to-end test: ingestor runs → anomaly detected → written to DB
- [ ] API response tests for all REST endpoints (GET /api/readings, GET /api/anomalies, POST /api/reports, GET /api/forecast, PATCH /api/anomalies)
- [ ] Verified no PII is stored after submitting a report

### WebSocket Latency Load Test
- [ ] Load test script written (validates the 2-second alert window)
- [ ] Test simulates a spike injection and measures time to client receipt
- [ ] **Must pass before Final Audit** — sub-2-second latency confirmed under load

### QA Sign-off
- [ ] User experience reviewed by QA Lead (Mubeen)
- [ ] All marking rubric criteria verified as met
- [ ] Virtual sensor data stream validated end-to-end
- [ ] External API integrations verified (weather + holidays)
- [ ] Scope confirmed — no out-of-scope features included

---

### Machine Learning Microservice (Python) — NEW 🚀

### Model Architecture (brain.py)
- [x] Implemented a deep learning Transformer architecture (replacing basic linear regression).
- [x] Uses Multi-Head Self-Attention and Positional Encoding to understand cyclical daily rhythms.
- [x] Multivariate input: 5 dimensions (Energy, Temperature, CO₂, Sin(time), Cos(time)).
- [x] Autoregressive inference: predicts the next step, then feeds it back in to predict the future.

### Training Pipeline (train.py)
- [x] Extracts ~20,000 simulated historical database rows.
- [x] Applies Min-Max scaling to compress raw data into 0-1 range for neural network stability.
- [x] Saves scaling parameters to scaler.json for inference translation.
- [x] Trained over 100 epochs, reaching a final loss of ~0.007.
- [x] Weights saved locally as best_model.pth.

### Inference API (brain.py)
- [x] FastAPI server running on port 8000.
- [x] Strict 24-step (2-hour) context window extraction to prevent context collapse on large UI requests.
- [x] Loads scaler.json to safely scale incoming React data down, and un-scale AI predictions back to real-world numbers.

## File Map (Active Code)

| Component | File | Status |
|-----------|------|--------|
| Entrypoint | `cmd/server/main.go` | ✅ |
| Database | `internal/database/database.go` | ✅ |
| Migrations | `internal/database/migrate.go` | ✅ |
| Schema | `internal/database/schema.sql` | ✅ |
| Models | `internal/models/models.go` | ✅ |
| REST Handlers | `internal/handlers/handlers.go` | ✅ |
| WebSocket Hub | `internal/ws/hub.go` | ✅ |
| Ingestor Worker | `internal/workers/ingestor.go` | ✅ |
| Anomaly Detector | `internal/workers/detector.go` | ✅ |
| Weather API | `internal/integrations/weather.go` | ✅ |
| Holiday API | `internal/integrations/holidays.go` | ✅ |
| Run Script | `run.sh` | ✅ |

> ⚠️ The `backend/` directory is a stale duplicate — all active code uses `cmd/` + `internal/` at the project root.

---
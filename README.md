# Smart Campus Energy Anomaly Dashboard

A full-stack smart-campus monitoring platform for detecting, reviewing, and reporting unusual energy-consumption behaviour.

The project combines a **Go backend**, **React/Vite frontend**, **FastAPI machine-learning service**, and **SQLite database** to provide a live dashboard for energy anomalies, forecasts, reports, and operational review.

> Developed as a collaborative university project. Contributions were shared across the team; see the Git history for individual authorship.

## Features

- Live monitoring of campus energy anomalies
- Interactive React dashboard
- Energy anomaly reporting and validation
- Machine-learning forecast endpoint
- WebSocket-based live updates
- Pause / resume controls for live monitoring
- CSV export for analysis and reporting
- Categorisation and tagging of events, including:
  - HVAC
  - Lighting
  - Equipment
  - Weather
  - Holiday
  - False Positive
- Pop-up notifications for detected events
- SQLite-backed persistent storage
- Automated backend and detector tests

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | React, Vite, TypeScript |
| Backend | Go |
| ML Service | Python, FastAPI |
| Database | SQLite |
| Real-time updates | WebSockets |
| Testing | Go testing framework |
| Version control | Git / GitLab |

## Architecture

```text
┌─────────────────────┐
│   React / Vite UI   │
│      :5173          │
└──────────┬──────────┘
           │ HTTP / WebSocket
           ▼
┌─────────────────────┐
│      Go Backend     │
│       :8080         │
├─────────────────────┤
│ API handlers        │
│ validation          │
│ WebSocket hub       │
│ database access     │
└───────┬─────────────┘
        │
        ├──────────────► SQLite
        │
        ▼
┌─────────────────────┐
│ FastAPI ML Service  │
│       :8000         │
├─────────────────────┤
│ forecasting         │
│ anomaly support     │
└─────────────────────┘
```

## Project Structure

The exact layout may vary slightly, but the application is organised around the following components:

```text
.
├── backend/            # Go backend, API handlers and WebSocket logic
├── frontend/           # React / Vite client
├── ml/                 # FastAPI machine-learning service
├── data/               # Local data / generated artefacts
├── tests/              # Automated and supporting tests
├── run.sh              # Convenience script for starting services
└── README.md
```

## Getting Started

### Prerequisites

Install:

- Go
- Python 3
- Node.js and npm
- SQLite

### Run the project

The repository includes a convenience script for starting the application:

```bash
chmod +x run.sh
./run.sh
```

The services are expected to run locally on:

```text
Frontend:    http://localhost:5173
Go backend:  http://localhost:8080
ML service:  http://localhost:8000
```

If running services individually, start the backend, frontend, and FastAPI service from their respective directories using the project-specific dependency files.

## Machine-Learning Forecasting

The FastAPI service exposes forecasting functionality used by the dashboard.

Example endpoint:

```text
/api/forecast
```

The frontend consumes this service alongside live and historical energy data to display predicted behaviour and support anomaly investigation.

## Testing

The Go backend includes automated tests covering important application behaviour such as:

- request validation
- report creation
- anomaly detection logic
- WebSocket hub behaviour
- database initialisation

Run the Go test suite with:

```bash
go test ./...
```

The project was also tested manually through end-to-end dashboard workflows, including:

- live dashboard monitoring
- anomaly logging
- report submission
- CSV export
- tagging
- notifications
- pause / resume functionality
- ML forecasting

## Validation and Reliability

Input validation is performed by the backend before data is persisted or processed. Invalid report data is rejected with an appropriate HTTP error response.

Automated testing is used alongside manual QA to reduce regressions across API, WebSocket, database, and anomaly-detection behaviour.

## My Contributions

My work on the project focused primarily on **backend reliability, testing, validation, and QA**, including:

- Implementing and extending Go backend tests
- Adding validation for report creation and invalid categories
- Testing anomaly-detection behaviour
- Testing WebSocket hub functionality
- Testing database initialisation
- Creating and maintaining the project `run.sh` workflow
- Performing end-to-end QA across the dashboard, reporting, CSV export, tagging, notifications, pause/resume behaviour, and ML forecasting

## Development Workflow

The project was developed collaboratively using Git branches and merge requests.

Changes were reviewed and merged through the shared repository.

## Notes

This repository was created as part of a university collaborative project. It is intended primarily as an educational and portfolio project rather than a production energy-management system.

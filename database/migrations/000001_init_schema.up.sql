CREATE TABLE IF NOT EXISTS readings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    building_id TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    kwh REAL NOT NULL,
    temperature REAL NOT NULL,
    co2_ppm REAL NOT NULL
);

CREATE TABLE IF NOT EXISTS reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    building_id TEXT,
    created_at DATETIME NOT NULL,
    category TEXT NOT NULL CHECK(category IN ('lighting', 'HVAC', 'security', 'other')),
    description TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS anomalies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    reading_id INTEGER NOT NULL,
    detected_at DATETIME NOT NULL,
    severity TEXT NOT NULL CHECK(severity IN ('low', 'medium', 'high')),
    tag TEXT,
    FOREIGN KEY (reading_id) REFERENCES readings(id) ON DELETE CASCADE
);

-- Optional: Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_readings_building_timestamp ON readings(building_id, timestamp);
CREATE INDEX IF NOT EXISTS idx_reports_building_created ON reports(building_id, created_at);
CREATE INDEX IF NOT EXISTS idx_anomalies_reading_id ON anomalies(reading_id);

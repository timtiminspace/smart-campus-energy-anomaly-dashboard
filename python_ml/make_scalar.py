# make_scaler.py
import sqlite3
import json
import numpy as np

conn = sqlite3.connect('../data/campus.db')
rows = conn.cursor().execute("SELECT kwh, temperature, co2_ppm FROM readings").fetchall()
conn.close()

metrics = np.array(rows, dtype=float)
d_min, d_max = metrics.min(axis=0), metrics.max(axis=0)

with open("scaler.json", "w") as f:
    json.dump({"min": d_min.tolist(), "max": d_max.tolist()}, f)

print("Success! scaler.json created instantly.")
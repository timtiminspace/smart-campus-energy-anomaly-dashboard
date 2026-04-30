import sqlite3
from datetime import datetime, timedelta
import random
import math

def seed_database():
    conn = sqlite3.connect('../data/campus.db')
    c = conn.cursor()
    
    # 1. Clear out the tiny dataset
    print("Clearing old readings...")
    c.execute("DELETE FROM readings")
    
    buildings = ['engineering', 'library', 'main-hall', 'admin', 'sports-hall']
    
    # Start exactly 14 days ago
    end_time = datetime.now()
    start_time = end_time - timedelta(days=14)
    
    print(f"Generating 14 days of cyclical data for {len(buildings)} buildings...")
    
    # 2. Generate cyclical data
    for b in buildings:
        curr = start_time
        while curr < end_time:
            # Create a 24-hour cycle peaking around 2 PM (14:00)
            hour_float = curr.hour + (curr.minute / 60.0)
            
            # Math to create a realistic daytime swell
            cycle = math.sin((hour_float - 6) * (math.pi / 12))
            cycle = max(0, cycle) # Buildings don't use negative energy
            
            # Apply base load + cycle + random noise
            kwh = 30 + (cycle * 180) + random.uniform(-10, 15)
            temp = 16 + (cycle * 6) + random.uniform(-0.5, 0.5)
            co2 = 400 + (cycle * 500) + random.uniform(-15, 15)
            
            ts_str = curr.strftime("%Y-%m-%dT%H:%M:%SZ")
            
            c.execute(
                "INSERT INTO readings (building_id, timestamp, kwh, temperature, co2_ppm) VALUES (?, ?, ?, ?, ?)",
                (b, ts_str, kwh, temp, co2)
            )
            curr += timedelta(minutes=5)
            
    conn.commit()
    conn.close()
    print("Success! Your database now has ~20,000 realistic readings.")

if __name__ == "__main__":
    seed_database()
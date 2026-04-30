import sqlite3
import torch
import torch.nn as nn
import numpy as np
from datetime import datetime
from brain import EnergyTransformer 

def train_transformer():
    # 1. Device Selection: FIXED the attribute error here
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print(f"Training on device: {device}")

    try:
        conn = sqlite3.connect('../data/campus.db')
        cursor = conn.cursor()
        cursor.execute("SELECT kwh, temperature, co2_ppm, timestamp FROM readings ORDER BY timestamp ASC")
        rows = cursor.fetchall()
        conn.close()
    except Exception as e:
        print(f"Database error: {e}")
        return

    if len(rows) < 50:
        print("Need more data in SQLite to train.")
        return

    # 2. Extract and Scale core metrics
    metrics = np.array([r[:3] for r in rows], dtype=float)
    d_min, d_max = metrics.min(axis=0), metrics.max(axis=0)
    scaled_metrics = (metrics - d_min) / (d_max - d_min + 1e-7)

    # 3. Build 5D Features (Metrics + Time Encodings)
    full_features = []
    for i, row in enumerate(rows):
        ts_str = row[3]
        try:
            ts = datetime.fromisoformat(ts_str.replace('Z', '+00:00'))
        except:
            ts = datetime.strptime(ts_str.split('.')[0], "%Y-%m-%d %H:%M:%S")

        minutes = ts.hour * 60 + ts.minute
        rad = (minutes / 1440.0) * (2 * np.pi)
        
        feat = np.append(scaled_metrics[i], [np.sin(rad), np.cos(rad)])
        full_features.append(feat)
    
    full_features = np.array(full_features)

    # 4. Sequence Generation
    X, y = [], []
    for i in range(len(full_features) - 24):
        X.append(full_features[i:i+24])
        y.append(scaled_metrics[i+24])
    
    # 5. Move Tensors to Device
    X_tensor = torch.tensor(np.array(X), dtype=torch.float32).to(device)
    y_tensor = torch.tensor(np.array(y), dtype=torch.float32).to(device)

    # 6. Initialize Transformer and move to Device
    model = EnergyTransformer(input_dim=5, output_dim=3).to(device) 
    optimizer = torch.optim.Adam(model.parameters(), lr=0.001)
    criterion = nn.MSELoss()

    print(f"Training Transformer on {len(X_tensor)} sequences...")
    
    model.train()
    start_time = datetime.now()
    
    for epoch in range(101):
        optimizer.zero_grad()
        preds = model(X_tensor) 
        loss = criterion(preds, y_tensor)
        loss.backward()
        optimizer.step()
        
        if epoch % 20 == 0:
            print(f"Epoch {epoch} | Loss: {loss.item():.6f}")

    end_time = datetime.now()
    print(f"Training complete in: {end_time - start_time}")

    # 7. Save weights (Move to CPU for universal loading)
    torch.save(model.to("cpu").state_dict(), "best_model.pth")
    print("Transformer weights saved to best_model.pth!")

if __name__ == "__main__":
    train_transformer()
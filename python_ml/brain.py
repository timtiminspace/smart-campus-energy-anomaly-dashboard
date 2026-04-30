import torch
import torch.nn as nn
import numpy as np
import math
import json
import os
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
from datetime import datetime, timedelta

app = FastAPI()

class PositionalEncoding(nn.Module):
    def __init__(self, d_model, max_len=500):
        super().__init__()
        pe = torch.zeros(max_len, d_model)
        position = torch.arange(0, max_len, dtype=torch.float).unsqueeze(1)
        div_term = torch.exp(torch.arange(0, d_model, 2).float() * (-math.log(10000.0) / d_model))
        pe[:, 0::2] = torch.sin(position * div_term)
        pe[:, 1::2] = torch.cos(position * div_term)
        self.register_buffer('pe', pe.unsqueeze(0))

    def forward(self, x):
        return x + self.pe[:, :x.size(1), :]

class EnergyTransformer(nn.Module):
    def __init__(self, input_dim=5, d_model=64, nhead=4, num_layers=2, output_dim=3):
        super().__init__()
        self.input_proj = nn.Linear(input_dim, d_model)
        self.pos_encoder = PositionalEncoding(d_model)
        encoder_layer = nn.TransformerEncoderLayer(d_model=d_model, nhead=nhead, batch_first=True, dropout=0.1)
        self.transformer = nn.TransformerEncoder(encoder_layer, num_layers=num_layers)
        self.output_proj = nn.Linear(d_model, output_dim)

    def forward(self, x):
        x = self.input_proj(x)
        x = self.pos_encoder(x)
        x = self.transformer(x)
        return self.output_proj(x[:, -1, :])

model = EnergyTransformer()
if os.path.exists("best_model.pth"):
    model.load_state_dict(torch.load("best_model.pth", weights_only=True))
    print("Loaded best_model.pth successfully.")
model.eval()

# Load the scaler
if os.path.exists("scaler.json"):
    with open("scaler.json", "r") as f:
        scaler = json.load(f)
        d_min = np.array(scaler["min"])
        d_max = np.array(scaler["max"])
        d_range = d_max - d_min + 1e-7
else:
    print("WARNING: scaler.json not found! Run make_scaler.py")

class MultiForecastRequest(BaseModel):
    history_kwh: list[float]
    history_temp: list[float]
    history_co2: list[float]
    steps: int = 12
    start_timestamp: str

@app.post("/api/ml/forecast")
def get_forecast(req: MultiForecastRequest):
    try:
        ts = req.start_timestamp.replace('Z', '+00:00')
        start_time = datetime.fromisoformat(ts)
        
        # 1. Enforce strict 24-step context (ignore older data for the prediction)
        kwh_24 = req.history_kwh[-24:]
        temp_24 = req.history_temp[-24:]
        co2_24 = req.history_co2[-24:]
        offset_idx = len(req.history_kwh) - len(kwh_24)

        # 2. Scale the input data down to 0-1
        features = []
        for i in range(len(kwh_24)):
            t_curr = start_time + timedelta(minutes=5 * (offset_idx + i))
            rad = ((t_curr.hour * 60 + t_curr.minute) / 1440.0) * (2 * np.pi)
            
            s_kwh = (kwh_24[i] - d_min[0]) / d_range[0]
            s_temp = (temp_24[i] - d_min[1]) / d_range[1]
            s_co2 = (co2_24[i] - d_min[2]) / d_range[2]
            
            features.append([s_kwh, s_temp, s_co2, np.sin(rad), np.cos(rad)])
        
        curr_input = torch.tensor([features], dtype=torch.float32)
        preds_kwh, preds_temp, preds_co2 = [], [], []

        # 3. Forecast loop
        with torch.no_grad():
            for step in range(req.steps):
                out = model(curr_input)
                p = out[0].tolist() 
                
                # 4. Un-scale the AI's prediction back to real values
                real_kwh = p[0] * d_range[0] + d_min[0]
                real_temp = p[1] * d_range[1] + d_min[1]
                real_co2 = p[2] * d_range[2] + d_min[2]
                
                preds_kwh.append(max(real_kwh, 0.0))
                preds_temp.append(real_temp)
                preds_co2.append(max(real_co2, 0.0))
                
                # Roll forward with SCALED predictions for the AI's internal state
                future_time = start_time + timedelta(minutes=5 * (len(req.history_kwh) + step))
                rad_future = ((future_time.hour * 60 + future_time.minute) / 1440.0) * (2 * np.pi)
                
                new_row = torch.tensor([[[
                    p[0], p[1], p[2], np.sin(rad_future), np.cos(rad_future)
                ]]], dtype=torch.float32)
                
                curr_input = torch.cat((curr_input[:, 1:, :], new_row), dim=1)

        return {"kwh": preds_kwh, "temp": preds_temp, "co2": preds_co2}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
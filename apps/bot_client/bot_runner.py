
import argparse
import socket
import struct
import time
import os
import subprocess
import torch
import torch.nn as nn
from torch.distributions import Categorical
import numpy as np
import sys

PROJECT_ROOT = "/content/drive/MyDrive/skateboard"
MODEL_PATH = os.path.join(PROJECT_ROOT, "shank_ai.pth")

# --- PROTOCOL v3 (10 Players) ---
SERVER_STATE_FMT = f"i {10 * 'i 3f 3f f f i i i 5i i i i i i'}" 
INPUT_FMT = "f f f f i i i i"

class ShankBrain(nn.Module):
    def __init__(self):
        super(ShankBrain, self).__init__()
        self.fc1 = nn.Linear(8, 128)
        self.fc2 = nn.Linear(128, 64)
        self.actor_move = nn.Linear(64, 4)   
        self.actor_aim = nn.Linear(64, 2)    
        self.actor_shoot = nn.Linear(64, 2)  
        self.critic = nn.Linear(64, 1)       

    def forward(self, x):
        x = torch.relu(self.fc1(x))
        x = torch.relu(self.fc2(x))
        move_probs = torch.softmax(self.actor_move(x), dim=-1)
        shoot_probs = torch.softmax(self.actor_shoot(x), dim=-1)
        aim_mean = torch.tanh(self.actor_aim(x))
        return move_probs, aim_mean, shoot_probs, None

class BotClient:
    def __init__(self, host, port, bot_id, brain):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.sock.setblocking(False)
        self.addr = (host, port)
        self.id = bot_id
        self.brain = brain
        self.my_state = None
        self.yaw = 0.0
        self.pitch = 0.0
        self.last_packet_time = 0
        self.connected = False

    def think(self):
        try:
            data, _ = self.sock.recvfrom(4096)
            self.last_packet_time = time.time()
            if not self.connected:
                print(f"[Bot {self.id}] 🟢 CONNECTED (10-Player Lobby)")
                self.connected = True
            obs = self.parse_state(data)
        except BlockingIOError:
            obs = None
            if self.connected and (time.time() - self.last_packet_time > 3.0):
                 self.connected = False

        if obs is not None:
            with torch.no_grad():
                move_probs, aim_mean, shoot_probs, _ = self.brain(obs)
            m = Categorical(move_probs).sample().item()
            s = Categorical(shoot_probs).sample().item()
            self.send_input((m, aim_mean[0].item(), aim_mean[1].item(), s))
        else:
            self.send_input((0,0,0,0))

    def parse_state(self, data):
        try:
            sz = struct.calcsize(SERVER_STATE_FMT)
            if len(data) < sz: return None
            unpacked = struct.unpack(SERVER_STATE_FMT, data[:sz])
            
            p_size = 23
            target_slot = (self.id + 2) % 10 # Update modulo for 10
            
            enemies = []

            for i in range(10): # Loop 10
                base = 1 + (i * p_size)
                p_data = unpacked[base : base + p_size]
                if p_data[0] == 0: continue 
                
                p_dict = {'pos': np.array(p_data[1:4]), 'hp': p_data[9], 'kills': p_data[10], 'ammo': p_data[13]}
                
                if i == target_slot: self.my_state = p_dict
                else: enemies.append(p_dict)

            if self.my_state is None: return None

            closest_dist = 9999.0
            rel_pos = np.array([0.0, 0.0, 0.0])
            enemy_hp = 0
            
            for e in enemies:
                dist = np.linalg.norm(e['pos'] - self.my_state['pos'])
                if dist < closest_dist:
                    closest_dist = dist
                    rel_pos = e['pos'] - self.my_state['pos']
                    enemy_hp = e['hp']
            
            return torch.tensor([
                self.my_state['hp'] / 100.0,
                self.my_state['ammo'] / 6.0,
                rel_pos[0] / 50.0,
                rel_pos[1] / 10.0,
                rel_pos[2] / 50.0,
                closest_dist / 50.0,
                enemy_hp / 100.0,
                1.0
            ], dtype=torch.float32)
        except Exception: return None

    def send_input(self, action):
        m, yd, pd, s = action
        fwd=0.0; strf=0.0
        if m==0: fwd=1.0
        elif m==1: fwd=-1.0
        elif m==2: strf=-1.0
        elif m==3: strf=1.0
        self.yaw += yd * 5.0
        self.pitch += pd * 5.0
        reload = 1 if self.my_state and self.my_state['ammo'] == 0 else 0
        header = struct.pack("i", 1)
        payload = struct.pack(INPUT_FMT, fwd, strf, self.yaw, self.pitch, 0, 0, s, reload)
        self.sock.sendto(header + payload, self.addr)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--host', default="127.0.0.1")
    parser.add_argument('--port', type=int, default=5000)
    parser.add_argument('--bots', type=int, default=1)
    args = parser.parse_known_args()[0]

    brain = ShankBrain()
    if os.path.exists(MODEL_PATH): brain.load_state_dict(torch.load(MODEL_PATH))
    
    target_host = socket.gethostbyname(args.host)
    bots = [BotClient(target_host, args.port, i, brain) for i in range(args.bots)]

    try:
        while True:
            for bot in bots: bot.think()
            time.sleep(0.016)
    except KeyboardInterrupt: pass

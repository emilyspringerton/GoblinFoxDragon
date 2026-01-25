import socket
import struct
import time
import random
import threading

# --- CONFIGURATION ---
SERVER_IP = "s.farthq.com"
SERVER_PORT = 5314
BOT_COUNT = 2 
PACKET_INPUT = 1

# Packet Format (Phase 95 MTU Fix + Phase 98 Header)
# Header: int type + int owner_id (Server ignores owner_id from client, but we match size)
# Body: ClientInput struct (fwd, strafe, yaw, pitch, jump, crouch, shoot, reload, wpn, zoom)
PACKET_FMT = "ii" + "ffffiiiiii" 

class Bot:
    def __init__(self, id):
        self.id = id
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.sock.setblocking(False)
        self.addr = (SERVER_IP, SERVER_PORT)
        
        self.yaw = 0.0
        self.pitch = 0.0
        self.move_timer = 0
        self.fwd = 0.0
        self.strafe = 0.0
        self.jump = 0
        self.shoot = 0
        self.weapon = 2
        
    def update(self):
        # Slower, more deliberate movement
        self.yaw += 1.5 
        
        self.move_timer += 1
        if self.move_timer > 60: # Change every 3 seconds
            self.move_timer = 0
            self.fwd = 1.0 # Just walk forward in a circle
            self.jump = 1 if random.random() < 0.2 else 0
            self.shoot = 1 if random.random() < 0.3 else 0
            
            # Swap between Rifle (2) and Shotgun (3)
            self.weapon = 3 if self.weapon == 2 else 2

        # Data construction
        # Send Type (1) + OwnerID (0, ignored) + Input Data
        data = struct.pack(PACKET_FMT, 
                           PACKET_INPUT, 0,
                           self.fwd, self.strafe, self.yaw, self.pitch,
                           self.jump, 0, self.shoot, 0, self.weapon, 0)
        
        try:
            self.sock.sendto(data, self.addr)
        except:
            pass

    def listen(self):
        try:
            while True:
                self.sock.recvfrom(4096)
        except BlockingIOError:
            pass

def run_patrol():
    print(f"👮 LAUNCHING PATROL: {BOT_COUNT} BOTS -> {SERVER_IP}:{SERVER_PORT}")
    bots = [Bot(i) for i in range(BOT_COUNT)]
    
    try:
        frame = 0
        while True:
            for bot in bots:
                bot.listen()
                bot.update()
            
            time.sleep(0.05) # 20 ticks/sec
            frame += 1
            if frame % 40 == 0:
                print(f"   [Patrol Active] Bots are walking circles...")
                
    except KeyboardInterrupt:
        print("\n🛑 PATROL STOPPED.")

if __name__ == "__main__":
    run_patrol()


import subprocess
import time
import os
import sys
import select

PROJECT_ROOT = "/content/drive/MyDrive/skateboard"
BOT_RUNNER = os.path.join(PROJECT_ROOT, "apps", "bot_client", "bot_runner.py")

DEFAULT_HOST = "s.farthq.com"
DEFAULT_PORT = 5000

def launch_swarm(host, port, instances=6):
    print(f"🚀 LAUNCHING VERBOSE SWARM ({instances} Clients)...")
    print(f"   Target: {host}:{port}")
    
    procs = []
    
    for i in range(instances):
        # We start each process and PIPE stdout so we can read it
        cmd = ["python", "-u", BOT_RUNNER, "--host", host, "--port", str(port), "--bots", "1"]
        p = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, bufsize=1)
        procs.append(p)
    
    print("✅ SWARM ACTIVE. LISTENING TO COMMS (Ctrl+C to Stop)...\n")

    try:
        while True:
            # Poll stdout of all bots
            reads = [p.stdout.fileno() for p in procs]
            ret = select.select(reads, [], [], 0.1)

            for fd in ret[0]:
                for i, p in enumerate(procs):
                    if p.stdout.fileno() == fd:
                        line = p.stdout.readline()
                        if line:
                            # Filter boring lines if needed
                            clean = line.strip()
                            if "HP:" in clean or "CONNECTION" in clean or "SILENCE" in clean:
                                print(f"[Inst {i}] {clean}")
    except KeyboardInterrupt:
        print("\n🛑 STOPPING SWARM...")
    finally:
        for p in procs: p.terminate()

if __name__ == "__main__":
    launch_swarm(DEFAULT_HOST, DEFAULT_PORT)

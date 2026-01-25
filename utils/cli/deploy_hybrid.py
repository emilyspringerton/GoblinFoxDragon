
import os
import subprocess
import sys
import time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../../"))
SERVER_SRC = os.path.join(PROJECT_ROOT, "apps", "server", "src", "main.c")
BIN_DIR = os.path.join(PROJECT_ROOT, "bin")
LOCAL_BINARY = os.path.join(BIN_DIR, "ShankServer_Linux")

HOST = "s.farthq.com"
USER = "root"
KEY_PATH = os.path.join(PROJECT_ROOT, ".ssh", "id_rsa")
REMOTE_DIR = "/root/shankpit"

GREEN = "\033[92m"
CYAN = "\033[96m"
RESET = "\033[0m"

def run_ssh(cmd):
    temp_key = "/tmp/shank_deploy_key"
    if not os.path.exists(temp_key):
        subprocess.run(f"cp '{KEY_PATH}' {temp_key} && chmod 600 {temp_key}", shell=True)
    
    ssh_opts = f"-i {temp_key} -o StrictHostKeyChecking=no -o ConnectTimeout=10"
    full_cmd = f"ssh {ssh_opts} {USER}@{HOST} '{cmd}'"
    subprocess.run(full_cmd, shell=True)

def deploy():
    print(f"{CYAN}🚀 DEPLOYING HYBRID SERVER...{RESET}")
    
    # 1. Compile
    print("🔨 Compiling Linux Binary...")
    inc_proto = os.path.join(PROJECT_ROOT, "packages", "protocol")
    inc_sim = os.path.join(PROJECT_ROOT, "packages", "simulation")
    inc_map = os.path.join(PROJECT_ROOT, "packages", "map")
    
    cmd = f"gcc {SERVER_SRC} -o {LOCAL_BINARY} -O2 -static -static-libgcc -I{inc_proto} -I{inc_sim} -I{inc_map} -lm"
    subprocess.run(cmd, shell=True, check=True)
    
    # 2. Upload
    print("☁️  Uploading...")
    temp_key = "/tmp/shank_deploy_key"
    subprocess.run(f"scp -i {temp_key} -o StrictHostKeyChecking=no {LOCAL_BINARY} {USER}@{HOST}:{REMOTE_DIR}/ShankServer", shell=True)
    
    # 3. Restart
    print("🔄 Restarting Service...")
    run_ssh(f"pkill -9 -f ShankServer || true")
    run_ssh(f"chmod +x {REMOTE_DIR}/ShankServer")
    run_ssh(f"nohup {REMOTE_DIR}/ShankServer > {REMOTE_DIR}/server.log 2>&1 &")
    
    print(f"{GREEN}✅ DEPLOY COMPLETE.{RESET}")
    print("   You can now connect with [A] (Online) or [D] (Offline)!")

if __name__ == "__main__":
    deploy()

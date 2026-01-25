
import os
import subprocess
import sys
import time

# --- PATHS ---
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../../"))
SERVER_SRC = os.path.join(PROJECT_ROOT, "apps", "server", "src", "main.c")
BIN_DIR = os.path.join(PROJECT_ROOT, "bin")
LOCAL_BINARY = os.path.join(BIN_DIR, "ShankServer_Linux")

# --- REMOTE CONFIG ---
HOST = "s.farthq.com"
USER = "root"
KEY_PATH = os.path.join(PROJECT_ROOT, ".ssh", "id_rsa")
REMOTE_DIR = "/root/shankpit"

GREEN = "\033[92m"
CYAN = "\033[96m"
RED = "\033[91m"
RESET = "\033[0m"

def run_local(cmd):
    try:
        subprocess.run(cmd, shell=True, check=True)
    except subprocess.CalledProcessError as e:
        print(f"{RED}❌ LOCAL FAIL: {e.cmd}{RESET}")
        sys.exit(1)

def run_ssh(cmd, verbose=True):
    # Setup Key
    temp_key = "/tmp/shank_nuclear_key"
    if not os.path.exists(temp_key):
        subprocess.run(f"cp '{KEY_PATH}' {temp_key} && chmod 600 {temp_key}", shell=True)
    
    ssh_opts = f"-i {temp_key} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=10"
    full_cmd = f"ssh {ssh_opts} {USER}@{HOST} '{cmd}'"
    
    try:
        res = subprocess.run(full_cmd, shell=True, capture_output=True, text=True)
        if verbose and res.stdout: print(f"   [REMOTE]: {res.stdout.strip()}")
        if res.stderr: print(f"   [STDERR]: {res.stderr.strip()}")
        return res.returncode == 0
    except Exception as e:
        print(f"{RED}❌ SSH FAIL: {e}{RESET}")
        return False

def nuke_and_pave():
    print(f"{CYAN}☢️  INITIATING NUCLEAR DEPLOYMENT SEQUENCE...{RESET}")
    
    # 1. COMPILE LOCALLY
    print(f"\n{CYAN}1. COMPILING FRESH BINARY...{RESET}")
    # Includes
    inc_proto = os.path.join(PROJECT_ROOT, "packages", "protocol")
    inc_sim = os.path.join(PROJECT_ROOT, "packages", "simulation")
    inc_map = os.path.join(PROJECT_ROOT, "packages", "map")
    
    compile_cmd = f"gcc {SERVER_SRC} -o {LOCAL_BINARY} -O2 -static -static-libgcc -I{inc_proto} -I{inc_sim} -I{inc_map} -lm"
    run_local(compile_cmd)
    print(f"{GREEN}   ✅ Compilation Successful.{RESET}")

    # 2. PREPARE KEY
    if not os.path.exists(KEY_PATH):
        print(f"{RED}❌ MISSING KEY: {KEY_PATH}{RESET}")
        return

    # 3. WIPE REMOTE
    print(f"\n{CYAN}2. WIPING REMOTE DIRECTORY...{RESET}")
    run_ssh(f"pkill -9 -f ShankServer") # Kill old
    run_ssh(f"rm -rf {REMOTE_DIR}")     # Delete folder
    run_ssh(f"mkdir -p {REMOTE_DIR}")   # Recreate
    print(f"{GREEN}   ✅ Remote Cleaned.{RESET}")

    # 4. UPLOAD (Force Simple Name)
    print(f"\n{CYAN}3. UPLOADING ARTIFACT...{RESET}")
    temp_key = "/tmp/shank_nuclear_key"
    scp_cmd = f"scp -i {temp_key} -o StrictHostKeyChecking=no {LOCAL_BINARY} {USER}@{HOST}:{REMOTE_DIR}/ShankServer"
    run_local(scp_cmd)
    print(f"{GREEN}   ✅ Upload Complete.{RESET}")

    # 5. EXECUTE
    print(f"\n{CYAN}4. LAUNCHING SERVER...{RESET}")
    # Chmod + Run
    start_cmd = f"chmod +x {REMOTE_DIR}/ShankServer && nohup {REMOTE_DIR}/ShankServer > {REMOTE_DIR}/server.log 2>&1 &"
    run_ssh(start_cmd)
    
    time.sleep(2)

    # 6. VERIFY
    print(f"\n{CYAN}5. VERIFICATION:{RESET}")
    print("   Checking Process...")
    run_ssh("ps aux | grep ShankServer | grep -v grep")
    
    print("   Checking Port 5000...")
    run_ssh("netstat -ulpn | grep :5000")
    
    print("   Checking Logs...")
    run_ssh(f"tail -n 5 {REMOTE_DIR}/server.log")

if __name__ == "__main__":
    nuke_and_pave()

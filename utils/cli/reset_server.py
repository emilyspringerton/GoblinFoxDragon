
import os
import subprocess
import sys
import argparse
import time

# --- DEFAULTS ---
DEFAULT_HOST = "s.farthq.com"
DEFAULT_USER = "root"
DEFAULT_KEY = os.path.join(os.path.dirname(os.path.abspath(__file__)), "../../.ssh/id_rsa")
REMOTE_DIR = "/root/shankpit"

GREEN = "\033[92m"
RED = "\033[91m"
CYAN = "\033[96m"
RESET = "\033[0m"

def run_ssh(host, user, key_path, cmd):
    # Prepare Key
    temp_key = "/tmp/shank_rescue_key"
    if not os.path.exists(temp_key):
        subprocess.run(f"cp '{key_path}' {temp_key} && chmod 600 {temp_key}", shell=True)

    ssh_opts = f"-i {temp_key} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=5"
    full_cmd = f"ssh {ssh_opts} {user}@{host} '{cmd}'"
    
    try:
        res = subprocess.run(full_cmd, shell=True, capture_output=True, text=True)
        return res.stdout.strip(), res.stderr.strip(), res.returncode
    except Exception as e:
        return "", str(e), 1

def rescue(host, user, key_path):
    print(f"{CYAN}1. CHECKING PROCESSES...{RESET}")
    out, err, _ = run_ssh(host, user, key_path, "ps aux | grep ShankServer | grep -v grep")
    if out:
        print(f"   Found running process:\n{out}")
    else:
        print("   No running server found (Server is DOWN).")

    print(f"\n{CYAN}2. FORCE KILLING...{RESET}")
    run_ssh(host, user, key_path, "pkill -9 -f ShankServer")
    time.sleep(1)
    
    # Verify Kill
    out, _, _ = run_ssh(host, user, key_path, "ps aux | grep ShankServer | grep -v grep")
    if out:
        print(f"{RED}   ❌ FAILED TO KILL PROCESS. IT IS A ZOMBIE.{RESET}")
        return False
    else:
        print(f"{GREEN}   ✅ Process Eliminated.{RESET}")

    print(f"\n{CYAN}3. STARTING FRESH (Latest Binary)...{RESET}")
    # Find the latest binary deployed
    start_cmd = f"nohup {REMOTE_DIR}/ShankServer > {REMOTE_DIR}/server.log 2>&1 &"
    out, err, code = run_ssh(host, user, key_path, start_cmd)
    
    time.sleep(2) # Give it a moment to bind

    print(f"\n{CYAN}4. VERIFYING PORT 5000...{RESET}")
    # Check if port 5000 is listening (using netstat or ss)
    out, _, _ = run_ssh(host, user, key_path, "netstat -ulpn | grep :5000")
    if "5000" in out:
        print(f"{GREEN}   ✅ SERVER IS LISTENING ON PORT 5000{RESET}")
        print(f"   {out}")
        
        print(f"\n{CYAN}5. RECENT LOGS:{RESET}")
        out, _, _ = run_ssh(host, user, key_path, f"tail -n 5 {REMOTE_DIR}/server.log")
        print(out)
        return True
    else:
        print(f"{RED}   ❌ SERVER STARTED BUT NOT LISTENING. CHECK LOGS:{RESET}")
        out, _, _ = run_ssh(host, user, key_path, f"cat {REMOTE_DIR}/server.log")
        print(out)
        return False

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--user", default=DEFAULT_USER)
    # Fix key path resolution
    script_root = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.abspath(os.path.join(script_root, "../../"))
    key_default = os.path.join(project_root, ".ssh", "id_rsa")
    
    parser.add_argument("--key", default=key_default)
    args = parser.parse_args()

    rescue(args.host, args.user, args.key)

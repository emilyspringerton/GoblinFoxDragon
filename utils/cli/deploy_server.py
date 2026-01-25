
import os
import subprocess
import sys
import argparse
import time

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../../"))
BIN_DIR = os.path.join(PROJECT_ROOT, "bin")

DEFAULT_HOST = "s.farthq.com"
DEFAULT_USER = "root"
DEFAULT_KEY = os.path.join(PROJECT_ROOT, ".ssh", "id_rsa")
REMOTE_DIR = "/root/shankpit"

GREEN = "\033[92m"
CYAN = "\033[96m"
RESET = "\033[0m"

def run_cmd(cmd, shell=False, check=True):
    try:
        return subprocess.run(cmd, shell=shell, check=check, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    except subprocess.CalledProcessError as e:
        print(f"❌ COMMAND FAILED: {e.cmd}")
        print(e.stderr)
        sys.exit(1)

def deploy(host, user, key_path):
    print(f"⚔️  DEPLOYING TO {host}...")

    # 1. BUILD FIRST (Get Version)
    print("🔨 Triggering Build...")
    res = run_cmd(f"python {os.path.join(SCRIPT_DIR, 'build_server.py')}", shell=True)
    
    # Parse output for VERSION_TAG=...
    version_tag = "ShankServer_Latest" # Fallback
    for line in res.stdout.splitlines():
        if line.startswith("VERSION_TAG="):
            version_tag = line.split("=")[1].strip()
            print(f"   -> Detected Build Version: {CYAN}{version_tag}{RESET}")

    local_binary = os.path.join(BIN_DIR, version_tag)

    # 2. PREPARE KEY
    temp_key = "/tmp/shank_deploy_key"
    run_cmd(f"cp '{key_path}' {temp_key} && chmod 600 {temp_key}", shell=True)
    ssh_opts = f"-i {temp_key} -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null"

    # 3. UPLOAD
    print(f"🚀 Uploading {version_tag}...")
    run_cmd(f"ssh {ssh_opts} {user}@{host} 'mkdir -p {REMOTE_DIR}'", shell=True)
    run_cmd(f"scp {ssh_opts} {local_binary} {user}@{host}:{REMOTE_DIR}/{version_tag}", shell=True)

    # 4. SWAP & RESTART
    print(f"🔄 Switching Version & Restarting...")
    remote_cmds = [
        f"chmod +x {REMOTE_DIR}/{version_tag}",
        "pkill -f ShankServer || true", 
        f"ln -sf {REMOTE_DIR}/{version_tag} {REMOTE_DIR}/ShankServer_Current",
        f"nohup {REMOTE_DIR}/ShankServer_Current > {REMOTE_DIR}/server.log 2>&1 &"
    ]
    run_cmd(f"ssh {ssh_opts} {user}@{host} '{' && '.join(remote_cmds)}'", shell=True)

    print(f"{GREEN}✅ DEPLOY SUCCESSFUL.{RESET}")
    print(f"   Active Version: {version_tag}")
    os.remove(temp_key)
    return True

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default=DEFAULT_HOST)
    parser.add_argument("--user", default=DEFAULT_USER)
    parser.add_argument("--key", default=DEFAULT_KEY)
    args = parser.parse_args()
    deploy(args.host, args.user, args.key)


import os
import subprocess
import sys
import datetime

# --- CONFIG ---
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../../"))
KEY_PATH = os.path.join(PROJECT_ROOT, ".ssh", "id_rsa")
REPO_URL = "git@gitlab.com:danowski/skateboard.git"

GREEN = "\033[92m"
RED = "\033[91m"
CYAN = "\033[96m"
YELLOW = "\033[93m"
RESET = "\033[0m"

def run_cmd(cmd, env=None, check=True):
    try:
        res = subprocess.run(cmd, shell=True, env=env, check=check, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        return res.stdout.strip(), res.stderr.strip(), res.returncode
    except subprocess.CalledProcessError as e:
        return e.stdout.strip() if e.stdout else "", e.stderr.strip() if e.stderr else str(e), e.returncode

def sync():
    print(f"{CYAN}📂 GIT SYNC v2 (ANTI-HANG){RESET}")
    
    # 1. SETUP SSH
    temp_key = "/tmp/gitlab_key"
    if not os.path.exists(KEY_PATH):
        print(f"{RED}❌ SSH Key missing at {KEY_PATH}{RESET}")
        return

    subprocess.run(f"cp '{KEY_PATH}' {temp_key} && chmod 600 {temp_key}", shell=True)
    
    # BatchMode=yes prevents password prompts (which cause hangs)
    git_env = os.environ.copy()
    git_env["GIT_SSH_COMMAND"] = f"ssh -i {temp_key} -o StrictHostKeyChecking=no -o BatchMode=yes"

    # 2. INIT
    if not os.path.exists(os.path.join(PROJECT_ROOT, ".git")):
        print(f"   Initializing Repo...")
        run_cmd(f"git -C {PROJECT_ROOT} init", env=git_env)
        run_cmd(f"git -C {PROJECT_ROOT} remote add origin {REPO_URL}", env=git_env)
    
    run_cmd(f"git -C {PROJECT_ROOT} remote set-url origin {REPO_URL}", env=git_env)
    run_cmd(f"git -C {PROJECT_ROOT} config user.email 'skatebot@farthq.com'", env=git_env)
    run_cmd(f"git -C {PROJECT_ROOT} config user.name 'SkateChain Bot'", env=git_env)

    # 3. COMMIT
    run_cmd(f"git -C {PROJECT_ROOT} add .", env=git_env)
    status, _, _ = run_cmd(f"git -C {PROJECT_ROOT} status --porcelain", env=git_env)
    if status:
        ts = datetime.datetime.now().strftime("%Y-%m-%d %H:%M")
        print(f"   Committing changes...")
        run_cmd(f"git -C {PROJECT_ROOT} commit -m 'Auto-save {ts}'", env=git_env)

    # 4. PUSH
    print(f"🚀 Pushing to GitLab...")
    out, err, code = run_cmd(f"git -C {PROJECT_ROOT} push -u origin master", env=git_env, check=False)

    if code == 0:
        print(f"{GREEN}✅ SYNC COMPLETE.{RESET}")
    else:
        print(f"{RED}❌ PUSH FAILED (Auth Error?){RESET}")
        if "Permission denied" in err:
            print(f"👉 YOUR PUBLIC KEY (Add to GitLab):")
            subprocess.run(f"ssh-keygen -y -f {temp_key}", shell=True)
        else:
            print(f"Error: {err}")

if __name__ == "__main__":
    sync()

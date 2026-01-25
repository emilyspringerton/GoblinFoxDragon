
import os
import subprocess
import shutil
import sys
import datetime

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
PROJECT_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "../../"))

SERVER_SRC = os.path.join(PROJECT_ROOT, "apps", "server", "src", "main.c")
BIN_DIR = os.path.join(PROJECT_ROOT, "bin")

# Generate Version Name: ShankServer_vYYYYMMDD_HHMM
VERSION = datetime.datetime.now().strftime("v%Y%m%d_%H%M")
BINARY_NAME = f"ShankServer_{VERSION}"
OUTPUT_BINARY = os.path.join(BIN_DIR, BINARY_NAME)

INC_PROTO = os.path.join(PROJECT_ROOT, "packages", "protocol")
INC_SIM = os.path.join(PROJECT_ROOT, "packages", "simulation")
INC_MAP = os.path.join(PROJECT_ROOT, "packages", "map")

def build_server():
    print(f"🔨 BUILDING {BINARY_NAME}...")
    
    if not os.path.exists(SERVER_SRC):
        print(f"❌ Source missing: {SERVER_SRC}")
        return None

    if not os.path.exists(BIN_DIR):
        os.makedirs(BIN_DIR)

    cmd = [
        "gcc", SERVER_SRC,
        "-o", OUTPUT_BINARY,
        "-O2", "-static", "-static-libgcc",
        f"-I{INC_PROTO}", f"-I{INC_SIM}", f"-I{INC_MAP}",
        "-lm"
    ]
    
    try:
        subprocess.run(cmd, check=True)
        print(f"✅ COMPILE SUCCESS: {OUTPUT_BINARY}")
        
        # Also create a 'latest' symlink/copy for ease of use
        latest = os.path.join(BIN_DIR, "ShankServer_Latest")
        shutil.copy(OUTPUT_BINARY, latest)
        
        return BINARY_NAME
    except subprocess.CalledProcessError as e:
        print(f"❌ COMPILE FAILED: {e}")
        return None

if __name__ == "__main__":
    res = build_server()
    if res:
        # Print the version name to stdout so deploy script can read it
        print(f"VERSION_TAG={res}")
        sys.exit(0)
    else:
        sys.exit(1)

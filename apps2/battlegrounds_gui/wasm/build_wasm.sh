#!/usr/bin/env bash
# build_wasm.sh — real WebAssembly build of the DragonsNShit MUD GUI client
# (apps2/battlegrounds_gui, the real IDUNA email+password login client, per
# REDGARDEN_GUI_NORTHSTAR.md and the Windows cross-compile step in
# .github/workflows/build.yml).
#
# 2026-08-20, founder real-time, urgent: "also we need the GFD gui client
# with login to have a web WASM or whateer however you wanna do it we need
# a web client for that product yesterday."
#
# Real, verified finding from getting this working: apps2/battlegrounds_gui's
# 3D world rendering already uses modern GL (VS_SRC/FS_SRC shaders, dynamic
# function loading) and ports to WebGL cleanly with no changes. The 2D HUD
# pass is legacy immediate-mode GL (glBegin/glEnd/glVertex2f/glColor3f/
# glOrtho -- ~400 glVertex2f call sites) -- Emscripten's own
# LEGACY_GL_EMULATION=1 flag handles all of that automatically, EXCEPT
# glRectf, which that emulation layer doesn't implement (it's a rarely-used
# convenience wrapper, not core immediate-mode). glrectf_shim.c in this
# directory is an 8-line trivial reimplementation in terms of the calls
# LEGACY_GL_EMULATION does cover -- not a rewrite of any real game code, a
# WASM-build-only addition.
#
# Prereq: Emscripten (emcc). No sudo needed -- emsdk installs entirely in
# userspace:
#   git clone https://github.com/emscripten-core/emsdk.git
#   cd emsdk && ./emsdk install latest && ./emsdk activate latest
#   source ./emsdk_env.sh
#
# Run from this directory:
#   source /path/to/emsdk/emsdk_env.sh
#   ./build_wasm.sh
# Output: battlegrounds.html / .js / .wasm in this directory. Serve with any
# static file server (emrun, python3 -m http.server, etc.) -- must be served
# over HTTP(S), not opened as a file:// URL (WASM fetch restrictions).

set -euo pipefail
cd "$(dirname "$0")"

if ! command -v emcc >/dev/null 2>&1; then
    echo "emcc not found -- source emsdk_env.sh first (see this script's own header comment)." >&2
    exit 1
fi

GUI_ROOT="$(cd .. && pwd)"

emcc -std=c99 -D_DEFAULT_SOURCE -O2 \
    -s USE_SDL=2 \
    -s LEGACY_GL_EMULATION=1 \
    -s ALLOW_MEMORY_GROWTH=1 \
    -o battlegrounds.html \
    "$GUI_ROOT/src/main.c" \
    "$GUI_ROOT/packages/simulation/arena_game.c" \
    "$GUI_ROOT/packages/simulation/arena_replay.c" \
    "$GUI_ROOT/packages/simulation/arena_ai_bridge.c" \
    "$GUI_ROOT/packages/common/mlp_infer.c" \
    "$GUI_ROOT/packages/goldenband/gband.c" \
    "$GUI_ROOT/packages/goldenband/gband_rig.c" \
    "$GUI_ROOT/packages/goldenband/gskel.c" \
    "$GUI_ROOT/packages/goldenband/gmesh.c" \
    "$GUI_ROOT/packages/goldenband/gband_mesh_rig.c" \
    glrectf_shim.c \
    -I"$GUI_ROOT/packages" \
    -lm

echo "Built: $(pwd)/battlegrounds.html (+ .js/.wasm). Serve over HTTP, e.g.:"
echo "  python3 -m http.server 8090 --directory $(pwd)"

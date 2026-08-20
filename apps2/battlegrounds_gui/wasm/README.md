# WASM build — DragonsNShit MUD GUI client

Real, working WebAssembly build of `apps2/battlegrounds_gui` (the real IDUNA email+password
login client — see `docs2/REDGARDEN_GUI_NORTHSTAR.md` and the Windows cross-compile step in
`.github/workflows/build.yml`). Founder, real-time, urgent (2026-08-20): "we need a web client
for that product yesterday."

## Status: compiles and links clean, serves correctly. Not yet visually verified in a real browser.

**Verified:**
- `emcc` (Emscripten 6.0.7, installed via `emsdk` — no sudo needed, entirely userspace) compiles
  and links the real client source (`src/main.c` + `packages/simulation/arena_game.c`/
  `arena_replay.c`/`arena_ai_bridge.c` + `packages/common/mlp_infer.c` + all 5 GOLDENBAND
  `packages/goldenband/*.c` files) with **zero source changes** to any of them.
- The 3D world rendering is already modern GL (dynamically-loaded via `SDL_GL_GetProcAddress`,
  shader-based) and needed nothing special.
- The 2D HUD pass is legacy immediate-mode GL (`glBegin`/`glEnd`/`glVertex2f`/`glColor3f`/
  `glOrtho` — roughly 400 `glVertex2f` call sites alone). Emscripten's own
  `-s LEGACY_GL_EMULATION=1` flag emulates all of that on top of WebGL automatically — this is
  the real, standard path for porting legacy-GL C code to the web, not something built for this
  session. The one gap: `glRectf`, a rarely-used convenience wrapper that emulation layer doesn't
  implement. `glrectf_shim.c` in this directory is an 8-line reimplementation in terms of the
  calls the emulation layer *does* cover — a WASM-build-only addition, not a change to any real
  game source.
- The three build artifacts (`battlegrounds.html`/`.js`/`.wasm`) serve correctly over plain HTTP
  (`python3 -m http.server`, confirmed via `curl` — all three return 200).

**Not verified — real, stated limitation, not glossed over:** this environment has no headless
browser (`chromium`/`google-chrome`) and no `puppeteer`/`playwright` install, so the page has not
actually been loaded and exercised in a real browser — whether SDL2's Emscripten port correctly
opens a WebGL context, whether input (mouse/keyboard) reaches the game, whether the login screen
actually renders and can authenticate against IDUNA over the browser's own fetch/XHR path (the
client's `IdunaClient`-equivalent networking code has not been checked for browser compatibility
at all — raw BSD sockets, which this client may use for its UDP arena protocol, do **not** work
in a browser at all; that's flagged as a real open question, not assumed to just work). A real
next step, not done here: open this in an actual browser (or install a headless one) and confirm
it boots to the login screen without a blank canvas or console errors.

## Build

```bash
git clone https://github.com/emscripten-core/emsdk.git   # if not already installed
cd emsdk && ./emsdk install latest && ./emsdk activate latest
source ./emsdk_env.sh

cd apps2/battlegrounds_gui/wasm
./build_wasm.sh
python3 -m http.server 8090   # serve the output dir; WASM needs real HTTP, not file://
```

## Real open question this doesn't answer

Confirmed directly (`grep`, not assumed): `src/main.c` uses real raw UDP sockets —
`socket(AF_INET, SOCK_DGRAM, 0)` + `sendto`/`recvfrom` for both the arena server connection and
matchmaker discovery, matching SHANKPIT/REDGARDEN's own server-authoritative UDP model. **Browsers
have no UDP socket API at all** — this build linked successfully only because Emscripten's own
POSIX socket emulation layer (`-lsockets`) stubs the calls at compile time; that says nothing
about whether they actually work at runtime without a real WebSocket-to-UDP proxy server in front
of the arena/matchmaker backends (Emscripten's socket emulation expects one, doesn't provide one).
This is the real, still-unsolved hard part of a *playable* web client — the rendering port done
here was the easy half. Real next step, not started: either stand up a WebSocket↔UDP relay in
front of `apps2/server-go`, or (bigger, cleaner) give the server a native WebSocket listener
alongside its existing UDP one.

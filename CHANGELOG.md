## 2026-08-03 (11)

- feat(battlegrounds_gui): real Town <-> Dragonfly zone teleport -- founder: "im expecting to
  teleport from town to the new zone." `town_telecrystal_travel` used to stop at the IDUNA
  position PATCH, leaving Town's own geometry on screen (a named gap in its own old doc comment).
  Now it also lazy-loads the real Dragonfly Meadow heightmap (`dfzone_load`, worldapi scene 0) and
  switches the client's render mode (`g_dfzone_active`): Town's ground/buildings/worms/building-
  labels stop drawing, `town_draw_dfzone` draws the real live heightfield mesh (reusing
  Milestones 2-4's pipeline unchanged) at the world origin, camera/avatar height follow it
  (`dfzone_height_at`). New "G" key is the return trip (`town_telecrystal_return`, real
  `TELECRYSTAL_ID_MEADOW_RETURN_HANDINGTON` values) -- a dedicated key rather than hijacking
  right-click, which players need for camera control while exploring the new zone.
- fix(battlegrounds_gui): real bug found while wiring the above -- the F10 debug toggle shipped in
  Milestone 2 lived in the battlegrounds-match event loop (`e`-scoped), which Town's own render
  branch's `continue;` skips entirely whenever `in_town` is true. F10 was dead code for real Town
  play; every "live-verified under Xvfb" screenshot for Milestones 2-4 actually exercised a
  temporary test-only env-var hook that set state directly, not the real key (the hook was
  removed before each commit as documented, but the real key handler itself was never reachable
  in Town). Moved into Town's own `te`-scoped event loop, where it's now actually reachable.
- Live-verified visually under Xvfb: screenshotted Town's checkerboard/buildings genuinely
  disappearing and the real Meadow terrain filling the screen at the destination, confirming the
  render-mode swap and worldapi fetch both work end to end. `go vet`/`go test ./...` and a direct
  `gcc` client build both clean.

## 2026-08-03 (10)

- feat(battlegrounds_gui): ship Milestone 4 of SMOOTH_TERRAIN_NORTHSTAR.md -- movement/camera
  elevation awareness, scoped to the F10 test patches (Town itself untouched, stays flat by
  design). New `terrain_test_height_at` samples the same CPU-side heights the GPU mesh was built
  from and returns real terrain height when standing inside a test patch, 0 elsewhere. Wired into
  camera focus (`mat4_orbit_view`'s `focus_y`, was hardcoded 0.0f) and the avatar's draw-time Y
  (combined with the existing jump-arc translate). New `terrain_test_offset_x` is the one shared
  source of each patch's world placement, used by both the renderer and the height lookup so they
  can't drift apart. Explicitly not done, named rather than skipped: `screen_to_ground`'s
  click-to-move ray-cast still targets a flat y=0 plane (real ray-vs-heightfield intersection is a
  harder problem, out of scope here) and WASD's own (x,z) update logic is unchanged -- only the
  resulting position's rendered Y now reads real terrain. Live-verified visually under Xvfb: the
  camera correctly settles onto the real sloped terrain surface at the avatar's position instead
  of floating above or clipping through it. `go vet`/`go test ./...` and a direct `gcc` client
  build both clean.

## 2026-08-03 (9)

- feat(battlegrounds_gui): ship Milestone 3 of SMOOTH_TERRAIN_NORTHSTAR.md -- biome flat-coloring.
  New `biome_color` maps worldapi's own `scene`/biome id to a flat RGB per draw call (Meadow
  grass green, Hills olive, Swampville muddy brown-green, unknown scenes grey). The F10 debug
  scene (Milestone 2) now fetches and renders all three column-derived biomes side by side
  instead of a single hardcoded green, so the milestone's own real terrain-test scaffolding is
  the proof, not new throwaway code. No new client-side biome enum -- reuses worldapi's own
  informal "sceneID is the biome selector" convention. Live-verified visually under Xvfb: all
  three patches render simultaneously with visibly distinct hues driven by each patch's own real
  `scene` field. `go vet`/`go test ./...` and a direct `gcc` client build both clean.

## 2026-08-03 (8)

- feat(battlegrounds_gui): ship Milestone 2 of SMOOTH_TERRAIN_NORTHSTAR.md -- client heightfield
  mesh renderer. New `build_heightfield_mesh`/`heightfield_sample` (`src/main.c`) fetch a real
  heightmap from the new `/heightmap` endpoint (Milestone 1) over HTTP, bilinearly interpolate at
  2x source resolution, and derive per-vertex normals from finite-difference height gradients --
  emitted through the exact same pos+normal `upload_mesh`/`draw_mesh` path every other mesh in
  this client already uses, no shader changes. New `http_extract_json_uint8_array_field`
  (`http_client.h`) parses the heightmap's numeric array field, same "controlled shape, not a real
  parser" convention as the other extractors there. Wired as an F10 debug toggle
  (`town_load_terrain_test`/`town_draw_terrain_test`) rendering the real live Hills chunk floating
  clear of Town's own footprint -- deliberately not integrated into Town itself (stays flat by
  design) and not wired into movement/collision (Milestone 4, later). Live-verified visually: built
  and ran the real client under Xvfb, connected to Town via a WOTAN dev-agent identity, screenshot
  of the F10 mesh shows a real smooth, continuously gradient-shaded rolling surface -- not
  stair-stepped cubes -- confirming both interpolation and lighting work against real backend
  data. `go vet`/`go test ./...` and a direct `gcc` build of the C client both clean.

## 2026-08-03 (7)

- feat(worldapi): ship Milestone 1 of SMOOTH_TERRAIN_NORTHSTAR.md -- backend heightmap exposure.
  New `GET /heightmap?scene=N&cx=X&cz=Z` (`server/worldapi/heightmap.go`) returns
  `{"height": [uint8 x256], "biome": int}` for Meadow (flat), Hills (real per-column variation --
  `hillsColumnHeight` split out of `hillsChunk` so block generation and the heightmap endpoint
  share one formula, can't drift apart), and Swampville (flat, water one block higher than land).
  Caves correctly 204s -- it's a genuinely 3D solid grid with no single height per column. New
  tests include a direct cross-check of the heightmap against `hillsChunk`'s own real block
  output. Live-verified against the running `gfd-server-go.service`. `go vet`/`go test ./...`
  clean (one pre-existing, unrelated `sync.RWMutex` copy warning in `apps2/server-go/main.go`,
  confirmed via earlier `git stash` comparison this session, not touched here).

## 2026-08-03 (6)

- docs(mud): confirmed Sunderworm crisis broadcasts reach Town's chat/combat-log pane -- an
  earlier open item, resolved by direct observation rather than new code. The crisis-phase
  handler (`main.go` ~1702) broadcasts to every connected player (`gw.players`), not zone-scoped,
  which is why this session's own Meadow validation test (P2, above) saw a New-Handington-zone
  crisis message while sitting in Meadow. On the headless path the push queues into the
  character's connection buffer and flushes on the next command -- observed directly in the P2
  `attack` response. `apps2/battlegrounds_gui`'s `town_poll_combat` already drains that buffer
  every ~1.5s into the shared combat log pane -- the exact code path this session's manual test
  exercised. No code change; delivery was already shipped, now verified rather than assumed.

## 2026-08-03 (5)

- test(mud): validated Meadow (scene 0) end-to-end through the real headless `/api/town/command`
  path -- P2 from the founder's own sprint plan ("get telecrystal working and then we validate the
  new zone"). Confirmed live: `look` shows real Meadow room text + 8 real worm mobs; `crystals`
  lists the real telecrystal network with the return-to-New-Handington crystal in range;
  `attack worm-meadow-0` lands a real 30-damage hit with TP gain (0→40) and a live world-crisis
  event fired mid-fight ("Something vast burrows beneath the Worm Hut"), after which all Meadow
  worms show `(burrowed)`; `north`/`south` correctly transition Meadow↔Hills and back; `say` works.
  Known, already-documented gap, not rediscovered as new: `apps2/battlegrounds_gui`'s 3D view stays
  New-Handington-specific after a real telecrystal travel -- this validation used the headless text
  path specifically because it doesn't depend on that render gap.

## 2026-08-03 (4)

- feat(server-go): get apps2/server-go running under supervision for the first time -- founder:
  "for now we need to get the dragonfly server seeded with a world". The binary existed but had
  never been run supervised; its hardcoded UDP `:6969` also collided with SHANKPIT's own live
  `shank_server` (confirmed via `lsof -i :6969`, not `ss`, which didn't reveal the real listener).
  Added a `-udp-port` flag (default unchanged at 6969) so it no longer has to fight SHANKPIT for
  the port. New `ops/systemd/gfd-server-go.service` user unit, deployed on `:6970` (worldapi
  `:7070`, trapx `:7071`). Live-verified: `GET /chunks?scene=0&cx=0&cz=0` returns real Meadow
  block data (1308 blocks, correct grass/dirt/stone layering, 8 real oak-log tree blocks) --
  "seeded with a world" is now literally true and running.
- docs(smooth-terrain): amended SMOOTH_TERRAIN_NORTHSTAR.md (§3.5 Trees, §3.6 the Town<->Dragonfly
  bridge open question, §3.7 explicitly-out-of-scope: world sculpting + real Bedrock connectivity)
  in response to founder direction ("render the dragonfly biomes smooth with trees... like a nice
  minecraft meadow biome but we render it with our frontend"). Added Milestone 0.5 (server-go
  running + seeded, DONE).
- docs(smooth-terrain): founder forked the real `github.com/df-mc/dragonfly` Bedrock server
  library to `emilyspringerton/dragonfly`. Confirmed genuine and unmodified (zero commits ahead of
  upstream `master`). Built and ran it vanilla: real RakNet/Bedrock listener on UDP `:19132`,
  `mc-version=1.26.30` -- a real phone Minecraft client can connect to it today, unmodified, for
  debug purposes. This answers "can I connect from my phone's minecraft, to debug" using vanilla
  upstream content; getting GoblinFoxDragon's own Meadow content reachable the same way is a
  separate, much larger integration (a custom world/chunk provider sourcing from
  `server/worldapi`'s `ProceduralWorldStore`), not attempted here.

## 2026-08-03 (3)

- fix(mud): REAL root cause of the gw.mu deadlock found and fixed -- founder: "pls fix" (not
  satisfied with the client-side workaround shipped a moment earlier). It was a plain,
  garden-variety self-deadlock, not anything exotic: `handle()` itself already does
  `gw.mu.Lock(); defer gw.mu.Unlock()` UNCONDITIONALLY right before its own dispatch switch (the
  only exception is the `/p` party-chat shortcut, which returns early before ever reaching that
  lock). `cmdTravel` then tried to acquire the exact same, non-reentrant `sync.Mutex` again on the
  SAME goroutine -- Go doesn't detect or panic on this, it just blocks forever. This is exactly
  why no "holder" ever showed up in any goroutine dump (SIGQUIT or live `dlv` inspection): the
  holder WAS the stuck goroutine itself, one frame further up its own stack inside `handle()`, not
  a separate one. It also explains why the earlier telnet A/B test looked like it worked: the
  "crystal resonates" message is sent BEFORE the lock attempt, so the telnet client received it
  before that same connection's own goroutine silently self-deadlocked afterward -- a follow-up
  command on that same session would have shown it too; it was just never tried.
  - Found via one precise structural test: an inline, trivial `gw.mu.Lock()` test case added
    directly to `handle()`'s own switch hung identically to `cmdTravel` -- but the exact same code
    moved to `/p`'s own position (before the switch, i.e. before the outer lock) worked. That
    isolated it immediately.
  - Real fix: removed `cmdTravel`'s own redundant `Lock()`/`Unlock()` entirely -- it's already
    called from inside `handle()`'s own locked dispatch, same as every other `cmd*` function that
    mutates `gw` state without locking (`cmdLook`, etc.).
  - Audited every other `gw.mu.Lock()` call site in the file and found three more real instances
    of the exact same bug, all fixed the same way: `cmdBattlegrounds` (re-locked to read
    `gw.charIDBySlot`), `cmdSummonAvatar` (re-locked to read `gw.players`), and the `warcry`
    ability case (called `broadcastZone`, the locking wrapper, instead of `broadcastZoneNoLock`).
    All three were real, live, previously-undiscovered risks -- any of them could have taken the
    whole server down the same way `cmdTravel` did, the moment anyone actually used them from
    chat.
  - Live-verified against the real production `gfd-mud.service`: two consecutive real telecrystal
    trips (`TELECRYSTAL_ID_TOWN_TO_MINES` then `_MINES_RETURN_TOWN`), the new
    `TELECRYSTAL_ID_HANDINGTON_TO_MEADOW`, and `battlegrounds` (ticket minting) -- all instant
    (~0.015s), all correct, `gameLoop`'s own tick confirmed healthy throughout (real crisis/
    faction-war events kept flowing).
  - The client-side workaround from the previous entry (`town_telecrystal_travel`, direct IDUNA
    PATCH bypassing `apps2/mud`) is left in place, not reverted -- it's simpler for the one Dragon
    Gate case and does no harm now that the underlying path is actually safe too.

## 2026-08-03 (2)

- fix(town): telecrystal ships via a safe workaround -- founder: "cook cook cook" (continuing P0
  from the sprint plan: root-cause the `cmdTravel` headless-path deadlock). Exhaustive
  investigation, root cause NOT found despite it:
  - Confirmed via a real SIGQUIT goroutine dump AND a live `dlv` session (attached with the
    correct Go toolchain version, target run with the real `IDUNA_AGENT_NAME`/`SECRET` env) that
    `gw.mu`'s own raw internal state shows `{state: 17, sema: 0}` -- genuinely locked, 2 real
    waiters (`gameLoop`'s own tick + the stuck request) -- while the COMPLETE goroutine list (11
    goroutines, delve's own exhaustive enumeration, not a partial signal-dump) contains no live
    holder anywhere. Every other request that also needs `gw.mu` (confirmed with `/p` party chat,
    which uses the identical `Lock()`/`defer Unlock()` shape) works instantly and correctly via
    the same headless path -- ruling out "any locked command via headless."
  - Reproduced with a `-race` build (no data race reported) and with a completely fresh character
    (never touched by this session's own earlier DB edits) using a pre-existing crystal
    (`TELECRYSTAL_ID_TOWN_TO_MINES`, positioned correctly, sufficient gold) -- ruling out both the
    new Meadow crystal and any stale test-character state as causes.
    Reproduced with `cmdTravel` restructured from a nested closure to a flat top-level
    `Lock()`/`defer Unlock()` (removing the one structural difference from the working `/p`
    pattern) -- ruling that out too. Confirmed `cmdBattlegrounds` and `cmdSummonAvatar` are the
    only other named functions with the same "locks `gw.mu`, called via `handle()`'s switch"
    shape as `cmdTravel` -- **not yet tested, a real, live, unconfirmed risk** that they carry the
    identical bug the moment either is ever exercised through headless/chat dispatch.
  - Given the severity (this takes the whole server down for every player, not just the one
    request) and that root-causing it has genuinely exhausted the obvious leads, shipped a real
    workaround instead of leaving telecrystal unusable: `town_telecrystal_travel()` bypasses
    `apps2/mud` (and its broken lock) entirely for the Dragon Gate specifically -- a direct PATCH
    to IDUNA's own `/api/v1/characters/:id/position`, the exact same safe, already-proven
    mechanism `town_sync_position` already uses continuously for ordinary movement. Target
    scene/position are `TELECRYSTAL_ID_HANDINGTON_TO_MEADOW`'s own real values, duplicated
    client-side -- the same convention `apps/lobby`'s own `TELECRYSTAL_DEFS` already established
    for the older SHANKPIT-lobby client, not a new pattern. Free (matches the server-side
    `CastCost` of 0), so no gold-deduction race to worry about doing this client-side.
  - Named, honest gap this doesn't solve: the client has no Meadow rendering at all, so after a
    real, correct backend zone/position change, the 3D view keeps showing New Handington until
    relogin (or a real future Meadow render mode) -- same category as the earlier Town-movement-
    bounds bug, but expected here, not a surprise.

## 2026-08-03 (1)

- fix(town): clamp movement to the real ground extent -- founder, live: "when i log in im not in
  town... i am floating in a blue abyss and it looks like theres some white writing off in the
  distance and i cant tell if i can run towards it or not and thats all thats rendering." Real
  bug: neither click-to-move nor WASD ever clamped position, unlike Battlegrounds' own hero
  movement (bounded to `ARENA_HALF_EXTENT`). Confirmed live -- the founder's own real character
  had drifted to `(61, 0, 3332.6)`, thousands of units past the actual ~113-unit ground/building
  layout. Nothing 3D renders that far out (only 2D building-name labels still project onto
  screen from any distance, which is exactly what read as "white writing in the distance" with
  everything else gone). Repositioned the live character back to open ground via a direct IDUNA
  update. Added `TOWN_MOVE_HALF_EXTENT` (derived from `town_draw_ground`'s own real footprint, so
  it can't drift out of sync with the visible ground) and clamped both click-to-move's target and
  WASD's per-tick target to it -- WASD held long enough was the more likely way to reach an
  absurd position in the first place, since it compounds every ~100ms with no cap.

## 2026-08-02 (23)

- feat(mud): New Handington <-> Meadow telecrystal, real critical bug found and NOT shipped to
  the GUI -- founder: "how do we get from town to the starter zone? have one of the gates act as
  a telecrystal... check the shankpit dragonsnshit codebase for the telecrystal logic." Found
  apps2/mud already has a full, real, player-facing telecrystal system (`server/telecrystal`,
  `cmdCrystals`/`cmdTravel`/`cmdTouchCrystal`) that apps/lobby's own SHANKPIT-style client
  already uses for Town/Mines/Docks/Giza -- New Handington (zone 4, apps2/battlegrounds_gui's own
  Town) predates that network entirely and had no crystal of its own. Added
  `TELECRYSTAL_ID_HANDINGTON_TO_MEADOW` + its return pair (free, CastCost 0 -- a starter-zone
  shuttle a level-1 character needs before they'd ever have Flow to spend on the older network),
  positioned at New Handington's real "Dragon Gate" building. Updated telecrystal_test.go for the
  new registry size/shape (8 crystals, non-negative-cost convention now that free ones exist).
  - **Real, critical, unresolved bug found live, not shipped**: the exact same crystal, invoked
    via a real telnet session, works correctly every time. Invoked via
    apps2/mud's headless `/api/town/command` HTTP path -- what Town's own GUI client uses for
    every chat/gate command -- `cmdTravel`'s `gw.mu.Lock()` call never returns, and takes the
    WHOLE mud server down with it: confirmed via a real SIGQUIT goroutine dump that `gameLoop`'s
    own 1Hz tick, which was ticking healthily seconds earlier, permanently stops too. Reproduced
    with `cmdTravel`'s body stripped to a bare `Lock()`/`Unlock()` with nothing else inside --
    the bug is not in what the function does with the lock, it's in headless dispatch reaching
    this point in some way not yet isolated. Converted the manual `Lock()`/`Unlock()` pair to
    `defer` (real hardening against any future panic there, but did not fix the actual hang).
    Given the severity (one GUI interaction can deadlock the entire server for every player),
    deliberately did NOT wire the Dragon Gate's right-click to `travel` in
    `apps2/battlegrounds_gui` -- the building interaction was reverted, left as a clear comment
    warning future work (and the founder, via chat) not to trigger `/travel` from the GUI until
    this is properly root-caused. The underlying telecrystal system itself is real and working
    for telnet players; only the headless/GUI path is unsafe right now.

## 2026-08-02 (22)

- feat(mud): real Sunderworm boss content for World Crisis -- founder: "start working on the
  sunderworm world event" -> "worm as the northstar" -> "build it on top of our starter zone on
  top of dragonfly." Found the event already had a full spec (`docs2/specs/WORLD_CRISIS_VS0.md`)
  and a real, tested phase machine (`server/worldcrisis`) wired into `apps2/mud`, but zero actual
  Sunderworm content: the trigger auto-restarted with no cooldown (spec E2 violation), the
  Anchor objective had no implementation anywhere, and "Chaos Elementals emerge in the Swamp"
  was 3 generic reskinned mobs with no boss mechanics.
  - `server/mob/sunderworm.go`: a real boss (15,000 HP) reusing `StateBurrowed` unchanged as its
    invulnerable state -- the phase names OMENS/BURROW/EMERGENCE already describe that exact
    cycle, no new mechanic needed. Two Sunderworm Head sub-bosses (4,000 HP) for Split War.
    Scaled up from `KindWorm`, the exact mob already in the starter zone -- not a new creature.
  - Wired into the crisis phase handler: spawns burrowed at the real Worm Hut position in zone 4
    when Burrow begins; the crisis handler (not an autonomous per-mob timer) surfaces it at
    Emergence alongside a 3-mob "Sunderworm Brood" add-wave in the same zone; two geo-separated
    Heads spawn east/west of the hut at Split War; Resolution soft-despawns everything (marks
    dead in place -- `Registry` has no removal API, a named gap).
  - Real gap closed: killing a Head now completes the Anchor objective (+15 LEY) -- previously
    the one of three required concurrent objectives with zero player-facing action, meaning
    Final Window's own gate could never actually be met.
  - Real bug fixed: added `crisisCooldown` (20 min) + `world.lastCrisisEndAt` so the event can't
    immediately re-trigger the instant it resolves.
  - Go build/vet/test green; redeployed `gfd-mud.service`; live-verified via a real telnet
    session through Omens -> Burrow.
  - Honestly still open per the spec's own DoD checklist: persistence (pure in-memory, IDUNA
    `PatchWorldEvent` never actually fires), rewards/merit/tiering, real Builder/Ritualist
    non-combat mechanics (Ritual is still just repurposed ore-mining), diminishing-returns
    anti-zerg, weak-point/armor-break beyond burrow/surface, and all client-side rendering (no
    GUI client consumes the crisis packet yet -- ties to `SMOOTH_TERRAIN_NORTHSTAR`'s still-
    unstarted milestones for real dragonfly terrain, not done here).

## 2026-08-02 (21)

- feat(town): C/Y/T also open chat, alongside Enter -- founder, after the Enter-key AH ordering
  fix still felt broken in practice: "the reason the auction house menu doesnt work is im trying
  to hit enter but that is triggering chat can we get a different hotkey than enter to start a
  chat enter can still send the chat" -> "how about make it work for c y and t just have them all
  map to start chat for now" -> "and then when we are not in the auction house enter also will
  open the chat." Enter staying in the open-chat list is safe specifically because the AH-menu
  block already runs first and consumes the event while `g_ah_screen != AH_CLOSED` -- Enter never
  reaches the chat-open check while the menu is open, so C/Y/T are additional ways in, not a
  replacement. In Battlegrounds' own in-match chat (separate event loop from Town's), "C" is
  deliberately left out -- it's already NORTHSTAR §15.1's `cam_locked` toggle there, and "keep
  battlegrounds as is" means that pre-existing binding doesn't get contested; Y/T are added there
  instead.

## 2026-08-02 (20)

- fix(town): stale connect ticket silently broke every requeue after ~5 minutes of play --
  founder, live-testing: "if i queue for battle grounds and then after that game return to town
  and then requeu for battlegrounds it doesnt work"; repro'd worked twice, then failed every
  time; "killing my client and relaunching it fixes it its a bug." Not the bot-pool/matchmaker
  race investigated earlier in this same session (that's real but was a red herring for this
  specific symptom) -- the real cause: `get_player_login_ticket` mints a connect ticket ONCE at
  initial login, and `net_connect()` reused that exact same `g_supplied_ticket_hex` for every
  reconnect for the rest of the session. IDUNA's `RedgardenTicketTTL` is a hardcoded 5 minutes
  (`redgarden_self_ticket.go`), and `arena_server` silently drops `PACKET_CONNECT` for an expired
  ticket (`apps/arena_server/src/main.c`'s own `expires_at` check, no rejection packet sent back)
  -- a dropped UDP connect just looks like "the human never joined the lobby," matching the
  "stuck at 19/20" symptom found while investigating. A fresh client relaunch mints a fresh
  5-minute ticket at its own new login, which is why that "fixed" it -- confirming this was
  never a REDGARDEN-side bug. Added `refresh_self_ticket()`, which re-mints from the stored login
  JWT (`g_chat_jwt`, much longer-lived than the ticket itself) right before every connect instead
  of reusing the first one -- same `/api/v1/redgarden/self-ticket` call `get_player_login_ticket`
  already made once, just repeatable. Falls through to the original static ticket if the refresh
  call itself fails (network hiccup, IDUNA briefly down). Bots/--ticket/--connect dev launches
  are untouched (no `g_chat_jwt` in those paths).

## 2026-08-02 (19)

- fix(town): Auction House Enter-key event-ordering bug -- founder, live-testing: "when i hit
  enter for browse categories the whole client crashes." Not a crash: Town's event loop checked
  the "Enter opens chat" shortcut BEFORE the Auction House menu's own Enter handling, so for any
  real logged-in player (`g_chat_jwt` set -- never true in this session's own earlier dev-agent
  testing, which is why it went unnoticed until a real login hit it), pressing Enter with the AH
  menu open opened the chat box instead of confirming the menu selection, then swallowed every
  further keystroke as chat text with the AH menu stuck open behind it and no way back -- reads
  exactly like a hang/crash at the keyboard. Fixed by checking the AH-menu block first, same
  precedence chat_input_active itself already gets. Verified both ways: pushed a real
  `SDL_KEYDOWN`/`SDLK_RETURN` event through the actual event-dispatch code (not a direct function
  call, which didn't reproduce this) against the pre-fix commit -- confirmed it reproduces
  (`ah_screen` stuck on MAIN, `chat_input_active` set) -- then against the fix, confirmed the
  menu now advances correctly and chat is untouched.

## 2026-08-02 (18)

- feat(town): real FFXI-style Auction House menu, doubled town/buildings, `/logout` chat command.
  - Auction House (founder: "make the auction house real - menu based system navigatable with
    arrow keys and enter just like ffxi - have it be interractable on right click"): right-click
    on the Auction House building opens a real menu (`AHScreen`: MAIN -> CATEGORIES ->
    CATEGORY_ITEMS / MY_LISTINGS), Up/Down navigate with wraparound, Enter confirms, Backspace
    goes back a level, Escape closes -- wired to apps2/mud's real, pre-existing `ah` command
    surface (`ah browse`, `ah sell`, `ah buy`, `ah history`, `ah status`, `ah cancel`) via
    `/api/town/command`, not mocked. `ah_draw_loading` added after live testing showed the
    blocking HTTP calls froze the frame with no feedback -- same fix pattern as
    `draw_queuing_screen`. Known real gap, not solved here: `ah browse <category>` only returns
    item-level aggregates, no listing IDs, so buying a specific other player's listing has no
    command surface yet.
  - Doubled the town (founder: "double the size of the town and the buildings"): every
    `TOWN_BUILDINGS[]` position and half-extent x2 (25 entries), `TOWN_TARGET_X/Z[]` (worm hut
    cluster) x2 to match, `server/mob/worm.go`'s `TownSquareWormSpawns` hutX/hutZ and cluster
    spread x2 to stay in sync with the client-side Worm Hut position. Diffed field-by-field
    against the pre-doubling commit (20b418e) to confirm an exact, clean x2 on every value.
  - `/logout` (founder: "in the chat /logout should log me out"): typing `/logout` in chat now
    quits the client, in both Town's own chat and Battlegrounds' in-match chat. Found and fixed a
    real bug in the process: the in-match chat handler had never actually been converted to
    `chat_send_or_command` (still called plain `chat_send`), so `/`-prefixed commands only worked
    from Town, not from an in-progress match.
  - Real bug found live during doubled-town verification, not a regression in the doubling
    itself: a test character had been positioned at the Auction House's exact center coordinates
    for AH testing before the doubling landed. Standing inside a building's own mesh means its
    inward-facing polygons are backface-culled, so the building (and anything else near you)
    renders as nothing -- read exactly like "buildings are gone." Repositioned off any building's
    bounding box; not a rendering or geometry defect.

## 2026-08-02 (17)

- feat(mud): headless-session M4 -- idle eviction + telnet-conflict handling, closing out
  `HEADLESS_SESSION_NORTHSTAR.md`'s full milestone table.
  - `evictIdleHeadlessSessions`: drops any headless session idle past 10 minutes (new
    `headlessLastActive`, updated on every `runHeadlessCommand` call), flushing final position
    the same shape a real telnet disconnect would (level/XP/flow are already kept in sync
    incrementally, unlike telnet's one-shot flush). Runs once per tick from `tickAll()`.
  - `disconnectHeadlessSession`: shared teardown (position flush, party leave, every registry a
    real disconnect clears, "has left the world" broadcast) used by both the idle sweep and the
    telnet-conflict path below.
  - `handleConn` now tears down any live headless session for the same character before a real
    telnet login takes over -- never two live `*player` structs for one character. Live-verified:
    created a headless session for a real character, connected via telnet under the same name,
    no crash, no duplicate registration.
  - Symmetric case also handled: `getOrCreateHeadlessPlayer` now refuses (409, not a crash) to
    spawn a headless session for a character already connected over real telnet -- a real
    telnet `*player`'s `w` wraps a `net.Conn`, not a buffer, so there's no output to hand back to
    a headless caller (routing a command INTO a live telnet session's own output stream is a
    real, separate, harder feature, not built here). Live-verified: held a telnet connection
    open, confirmed a concurrent `/api/town/command` call for the same character correctly
    returns 409 "character is currently connected via telnet" instead of crashing or duplicating.
  Go build/vet clean.

## 2026-08-02 (16)

- feat(town): sync Town with the MUD -- real commands from the chat box, real telnet visibility.
  Founder: "i want you to sync up town with the MUD" -> chose both "real MUD commands from
  Town's chat box" and "telnet players see Town's GUI players."
  - `chat_send_or_command` (`apps2/battlegrounds_gui`): a line typed into either chat box (Town's
    own, or the in-match one -- `g_town_char_id` survives entering/leaving a match) starting with
    "/" now routes to the real headless-session command dispatch instead of ordinary chat --
    `HEADLESS_SESSION_NORTHSTAR.md`'s own original M3 design, finally built. "/look",
    "/inventory", anything `handle(p, line)` understands, not just the "1" attack keybind. Output
    shares the combat log pane.
  - `getOrCreateHeadlessPlayer` (`apps2/mud`) now registers into `gw.zoneMgr`/`gw.chatRouter` and
    broadcasts "X has entered the world" on creation, same as a real telnet connection -- a real
    telnet player standing in the same zone now sees a Town player's presence live, not just via
    `look`'s own already-working `gw.players` loop.
  Go build/vet clean. Live-verified end-to-end with two real characters: created character 1's
  headless session, then character 2's -- character 1's own session buffer captured "Test2Warrior2
  has entered the world." in real time, confirming the broadcast reaches other live sessions
  exactly like a real telnet connection would see it.
  Still open, unchanged: no idle eviction or "has left the world" broadcast (a headless session
  never disconnects in the traditional sense, `HEADLESS_SESSION_NORTHSTAR.md` M4).

## 2026-08-02 (15)

- fix(town): dead-connection recovery -- a narrower race in the same REDGARDEN-side matchmaking
  issue found earlier today, this time surfacing as "put me into the map with nothing happening
  skipping the draft" rather than an outright connect failure. Founder: "i closed dragonsnshit
  client and reopened it and that did not fix it - well it did something different it put me
  into the map with nothing happening skipping the draft."
  - Root cause: `net_connect()` can legitimately receive `PACKET_WELCOME` (so the earlier
    connect-failure fallback never fires -- the client really did connect) in the same window
    `arena_server`'s own 60s no-lobby-progress watchdog kills the match. The client is then left
    on a dead socket forever: `net_phase` stuck at its default `ARENA_PHASE_WAITING`, never
    `ARENA_PHASE_DRAFT`, so the draft screen never shows and `arena_state` never updates --
    exactly "nothing happening, skipped the draft."
  - Fix: `g_net_last_packet_ms` now tracks the last time *anything* arrived from the server
    (`net_poll_snapshots`), reset to 0 on a fresh connect. If 10s pass post-connect with zero
    packets ever received, the client treats it as a dead match and recovers -- same "land back
    in Town, not a dead end" pattern the earlier requeue-failure fix already established. Gated
    on `queue_host` (no Town to return to for a direct `--connect` dev session, same convention
    as the other Town-recovery paths). `apps/arena_server`/`apps/matchmaker` untouched, per
    standing instruction.
  - Live-verified the exact race with a fake matchmaker+server (real `NetHeader`/`MatchFoundMsg`
    wire format, confirmed via `sizeof()` against the real struct rather than assumed): replies
    with a real `PACKET_WELCOME` then goes silent forever. Confirmed via log timestamps: the
    client connects, waits the full 10s with zero packets, then fires the exact recovery message
    and sets `in_town = 1` -- the same reused code path already visually verified rendering Town
    correctly in three other contexts this session (initial entry, Return-to-Town button, failed
    requeue).

## 2026-08-02 (14)

- feat(town): "New Handington" -- real town layout transcribed from a hand-drawn map. Founder
  uploaded `town-map.jpeg` straight to GitHub: "i want the town layout to match town map pretty
  much exactly."
  - Zone 4 renamed "Town Square" -> "New Handington" (`server/zone/zone.go`), matching the map's
    own title.
  - 25 named buildings transcribed from the sketch into `TOWN_BUILDINGS` (`apps2/battlegrounds_gui`):
    Warrior Guild, Seed Shop, Fishing, Blacksmith, Butcher, Armor Shop, Shady Dealer, Guild
    House, Potions, Gold Guild, Secret Gate, Auction House, Archery Guild, Post Office, Town
    Hall, Gem Dealer, Police, Gemani Tower, MineCo Ops Office, Mining Supplies, Glove Shop,
    Hats, Worm Hut, Dragon Gate, Diamond Gate -- placed at a row/col reading of the map's own
    relative layout, not exact hand-drawn shapes (every other structure in this renderer is
    axis-aligned boxes, so buildings follow the same art style). Each renders as a colored box
    (category-coded: guilds blue, shops green, official grey, shady/secret purple, gates gold)
    with a floating name label (`world_to_screen`, same projection Battlegrounds' own per-hero
    health bars use).
  - `server/mob/worm.go`'s `TownSquareWormSpawns` repositioned from an origin-centered ring to a
    tight cluster at the map's own real "Worm Hut" location (5, 15) -- client-side
    `TOWN_TARGET_X/Y` updated to match exactly.
  Go build/test green. Native client build clean; visually verified under Xvfb (elevated test
  camera): buildings render at real relative positions with readable labels, matching the map's
  layout (Dragon Gate north, Warrior Guild/Seed Shop/Blacksmith clustered near it, Gold
  Guild/Guild House/Butcher/Fishing/Post Office/Potions/Auction House/Armor Shop all correctly
  positioned relative to each other and to spawn).

## 2026-08-02 (13)

- feat(town): chat + combat log panes, target cycling, real "1" attack, jump, worm ring expanded
  to 4. Founder, several rapid follow-ups after killing the first worm: "where is my chat window
  and combat log window in town? those are going to stay up during normal gameplay" -> "add tab
  and shift tab to cycle through targets like wow" -> "to be clear we need to unify battlegrounds
  combat with the mud combat on the dragonsnshit side dont touch our MOBA in REDGARDEN repo" ->
  "where's my starter zone outside of town with the worms?" -> "add jump space bar".
  - `chat_draw`/`combat_log_draw` (already built for Battlegrounds) now also render in Town, plus
    full chat input handling (Enter to open/send, Escape to cancel) wired into Town's own event
    loop -- same shape Battlegrounds' own chat handling already uses, checked first so it
    consumes keystrokes before WASD/target-cycling/attack do.
  - `server/mob/worm.go`'s `TownSquareWormSpawns` expanded from 1 worm to a real ring of 4 (same
    shape as `MeadowWormSpawns`' own ring, smaller) -- "a single worm" didn't read as "a starter
    zone." `town_draw_worms` (renamed from `town_draw_worm`) now draws all 4 at their real spawn
    positions, matching mob IDs (`worm-town-0..3`).
  - Tab/Shift+Tab cycle `g_town_target_index` through the 4 worms; the selected one renders with
    an amber highlight, and the HUD shows `Target: worm-town-N`.
  - Pressing "1" -- the same ability-slot keybind Battlegrounds already uses (Q/W/E rebound to
    1/2/3 this fork) -- sends `attack <target>` to apps2/mud's real `/api/town/command`. This is
    the concrete first step of "unify battlegrounds combat with the mud combat": the same keybind
    language now drives the real MUD combat system, not a separate control scheme. REDGARDEN's
    own repo untouched, as instructed.
  - `town_poll_combat`: throttled (~1.5s) drain of the headless session's buffer, so background
    auto-attack ticks show up in the combat log even without pressing anything. New
    `town_send_command` shares the exact combat log pane Battlegrounds' own combat log already
    uses (`combat_log_push`) -- filters out the bracketed status line and bare prompt so the pane
    reads as combat events, not a raw MUD terminal dump.
  - Real bug fixed on the server side while wiring this up: `/api/town/command`'s JSON response
    used Go's default HTML-escaping, turning `>>> LEVEL UP <<<`-style real MUD text into
    `>>>`-garbled output the client's minimal JSON extractor couldn't unescape.
    Fixed with `Encoder.SetEscapeHTML(false)`.
  - Space bar triggers a purely cosmetic vertical bounce (sine arc, `TOWN_JUMP_DURATION_MS`/
    `TOWN_JUMP_HEIGHT`) -- Town has no verticality/collision system, so this doesn't interact
    with anything, named honestly in its own doc comment. Applied via a local vp pre-multiplied
    by a world-space Y translate for just the avatar's own draw call, not a change to
    `draw_hero_model`'s shared signature (also used by the real match renderer).
  Go build/vet + `server/mob` tests green. Native client build clean; visually verified under
  Xvfb with synthetic input (Tab, "1", Space) against a real login: target highlight, real combat
  text flowing into the pane ("worm-town-0 hits you for 8", "turns toward you"), real chat
  history rendering, all confirmed live.

## 2026-08-02 (12)

- feat(mud): real headless-session combat -- Town Square's worm is now genuinely fightable, not
  decorative. Founder: "can we kill worms?" -> chose "the real MUD combat system" over a simpler
  fake-hit-for-damage mode. Implements HEADLESS_SESSION_NORTHSTAR.md's core mechanism for the
  first time:
  - `getOrCreateHeadlessPlayer(characterID)`: builds a real `*player` (no telnet connection) from
    a real IDUNA character, `w: bufio.NewWriter(&buf)` where `buf` is an owned `*bytes.Buffer`
    (new `headlessBuf` field on `player`, nil for every real telnet player). Registered directly
    into `gw.players["headless:"+characterID]` -- the exact same map every telnet player uses --
    so the real 1Hz `gameLoop()`/`tickAll()` resolves its combat (auto-attack swing timer, TP,
    enmity, kill/loot/XP via `resolveKill`) with zero changes to the tick loop itself. Seeded from
    the character's real `scene_id`/position, which Town's own position sync (`50d582e`) already
    writes as zone 4 -- a fresh headless session naturally starts standing in Town Square, next
    to the real worm.
  - `runHeadlessCommand(characterID, line)`: runs one line through the real `handle(p, line)`
    dispatch and drains everything written since the last drain -- both the command's own
    response and any background tick messages, so a caller can poll with an empty line to catch
    auto-attack ticks without issuing a new command each time.
  - New `POST /api/town/command` on the existing `:7171` world-events API (same mux, same
    no-auth trust model -- named gap, not fixed: `character_id` is caller-supplied, not derived
    from any verified identity; apps2/mud has never verified an incoming JWT at all, only issued
    outbound agent calls).
  - Fixed a real, separate bug found while making this actually persist: a headless session
    never "disconnects," so `handleConn`'s own disconnect-time level/XP/flow sync never fires for
    it. New `headlessSyncedLevel/XP/Flow` fields + a delta-sync after every `runHeadlessCommand`
    call, same `UpdateCharacterLevel`/`CreditGold`/`DeductGold` calls the disconnect path uses.
  - **Two additional real bugs found and fixed along the way**, both pre-existing, both affecting
    real telnet play too, not just this feature: (1) `gfd-mud.service` never had
    `IDUNA_AGENT_NAME`/`IDUNA_AGENT_SECRET` configured (no `EnvironmentFile=` line existed) --
    every `idunaclient` call this live process has ever made was silently 401ing, masked by
    best-effort error handling everywhere. Fixed via a new `~/.config/gfd-mud/env`, same
    convention `iduna.service` already uses. (2) `idunaclient.UpdateCharacterLevel` called a
    route that has never existed on IDUNA (`PATCH /api/v1/characters/:id` with no suffix) --
    fixed client-side (now hits the new `/level` route, IDUNA `3ebad87`) and would have silently
    broken level/XP persistence for every real telnet disconnect too, this whole time.
  - `p.conn.Close()` in the `quit` command guarded against a nil `conn` (headless sessions have
    none) -- defensive; the new endpoint never sends `quit`, but cheap to guard anyway.
  Live-verified end-to-end, real character, real live worm: `attack worm` targets and
  auto-approaches, real tick-based swings land (30 damage/hit, worm's own 8-damage retaliation),
  real kill (`The creature collapses!`), real XP (+900), real level-up (1→3), real loot (Worm
  Sinew, Earth Crystal) -- and, after the two bugs above were fixed, confirmed landing in IDUNA
  for real (`level: 3, current_xp: 452`), not just in-memory. Go build + vet clean across
  `apps2/mud`, `apps2/server-go` (shares the fixed `idunaclient`), and `server/idunaclient`.

## 2026-08-02 (11)

- fix(town): position now flushes on quit, closing a real "login to same spot" gap. Founder:
  "ensure my avatar can move around town and the location is persisted so login to same spot."
  `town_sync_position` was throttled to once per 2s -- a player who moved and then closed the
  window inside that window lost their last few steps, landing slightly short of their real
  position on next login. `town_sync_position` gained a `force` param; called once more, forced,
  right as the app shuts down (after the main loop exits, before SDL teardown), flushing any
  movement the throttle hadn't caught yet. Live-verified end-to-end: reset test character to
  (0,0,0), ran a real login + forced movement + quit at 800ms (well under the 2s throttle) --
  IDUNA showed the partial movement `(2.383, 0, 1.589)` persisted anyway; a second fresh launch
  loaded from exactly that position, confirming true session-to-session persistence, not just
  in-session syncing.

## 2026-08-02 (10)

- fix(mud): S170-57, worm's Poison restored. Founder, real-time: "add poison back to that level
  1 worm you winey noob you just lowered the game difficulty because you didnt like it lol."
  Offered the real, tested reason it was removed (`ba735e8`, 2026-07-23: a flat Potency=10 debuff
  ticking once per game tick for up to 30s is up to 300 total damage from a single proc, against
  a level 1-5 character with only 90-150 max HP -- a real death, live, on this game's own zone-0
  tutorial mob) and asked which way to go; founder chose "Poison exactly as it was," knowingly:
  "this is a game for the hardcore a lvl1 poison ko is perfect." `mobSpellPool["worm"]` restored
  to its exact pre-`ba735e8` value, `{status.Slow, status.Poison}`. Still not "always" (confirmed,
  founder: "as long as its not always") -- unchanged 20% base proc chance, then a 50/50 pick
  between Slow and Poison, same mechanism as every other mob in the pool. Go build/vet green
  (no test coverage exists for apps2/mud at all, unchanged). Live `gfd-mud.service` rebuilt and
  redeployed.

## 2026-08-02 (9)

- feat(town): M5 ability panes (inert) + new zone 4 "Town Square" + starter-area worm. Founder:
  "and then implement the starter area worm" -> "you may need to add the next zone."
  - **M5 (ability panes)**: same `draw_ability_tile`, bottom-center layout, and Q/W/R color
    scheme Battlegrounds' own ability bar uses, ported into Town's HUD. Deliberately inert --
    Town has no cast/combat system, no per-job skill data wired in from apps2/mud's real
    weapon-skill system, and no mana; every tile is permanently ready and labeled
    "(unassigned)" rather than faked as functional. Only shown once a real character has loaded.
  - **New zone 4, "Town Square"** (`server/zone/zone.go`): a real, separate zone rather than
    reusing zone 0 (Meadow) -- Meadow already has a live telnet MUD presence (worm combat, "X
    has entered the world" broadcasts), and conflating it with the GUI's own local-only,
    no-combat Town scene would be exactly the "gui and the mud play nice... might get weird"
    tension named earlier today. `apps2/mud`'s `initWorld()` now spawns a mob registry + weather
    entry for it. `town_sync_position`'s own scene_id changed from 0 to `TOWN_ZONE_ID` (4).
  - **`server/mob/worm.go`'s new `TownSquareWormSpawns()`**: one real worm mob (not Meadow's full
    ring of eight -- Town Square is a small starter area), scene_id 4, spawned into
    `apps2/mud`'s real mob registry -- real backend content, not just client-side decoration.
  - **`town_draw_worm()` (client-side)**: a small three-segment box silhouette at the worm's real
    spawn position (mirrored by hand, same convention this codebase already uses for static
    positions). Named honestly as decorative in its own doc comment: apps2/mud has no HTTP
    surface for mob state at all yet, so this isn't a live sync of the real mob's HP/AI, just a
    placeholder at the right spot.
  - **Explicitly not built**: `apps2/server-go`'s own real voxel/chunk "Dragonfly" world
    (`server/worldapi`'s `DragonflyChunkGenerator`/`ProceduralWorldStore`, which already reserves
    the same 0-3 scene IDs for its own procedural terrain) -- wiring Town into that would mean a
    whole new UDP protocol client speaking `packages2/common/protocol.h`, a real, separate,
    much larger undertaking, flagged here rather than guessed at.
  Go build + `server/zone`/`server/mob` tests green (`zone_test.go`'s own zone-count assertion
  updated 4→5). Native Linux client build clean; visually verified under Xvfb with a real login --
  avatar, worm, and all three ability tiles render correctly together.

## 2026-08-02 (8)

- feat(town): real avatar, movement, and position sync in Town, backed by IDUNA's real
  `characters` record ("the dragonfly backend"). Founder: "i want my avatar to move around in
  town but i want it backed by dragonfly" -- continuing "this is the time to unify the whole
  bitch" / "wire up the dragonfly backend" / "the xyz at least needs to flow back to the
  dragonfly server for the gui xyz source of truth." M1-M4 of the same-day backlog/sprint plan:
  - `town_fetch_character()`: on entering Town, resolves the player_id captured from login's own
    self-ticket response (already returned it, no new endpoint) to the real character via
    `GET /api/v1/characters/by-player/:id`, seeding position and `job_main`.
  - `town_hero_id_for_job()`: `ARENA_HERO_WARRIOR` for job `WAR` -- the one real, non-guessed
    correspondence (`arena_game.h`'s own doc comment already calls it "DragonsNShit's Warrior
    job, ported as Battlegrounds content"). Every other apps2/mud job has no hero-visual mapping
    yet -- falls back to Warrior, named as a placeholder in the comment, not the UI. A real, open
    design question, not resolved here.
  - Real movement: WASD (camera-relative, same derivation as Battlegrounds' own) + click-to-move
    (`screen_to_ground`, same ray-cast), interpolated at `ARENA_HERO_SPEED` for a consistent feel
    between the two scenes. Camera now follows the avatar's own position instead of a fixed
    origin. Purely local -- no server involved in Town's own simulation.
  - `town_sync_position()`: throttled (2s, only if actually moved) `PATCH
    /api/v1/characters/:id/position` using the player's own JWT -- the same endpoint IDUNA
    `ab35b72` just hardened for exactly this caller.
  - New `http_extract_json_double_field` helper (`http_client.h`) for `pos_x`/`pos_y`/`pos_z`
    (SQL `REAL` columns) -- the existing extractors only handled strings and integers.
  Accepted tradeoff, founder's own words: "it might get weird having the gui and the mud play
  nice" -- whichever of Town or a live apps2/mud telnet session last PATCHes position wins, no
  conflict resolution beyond that. Live-verified end-to-end against the real test account and
  live IDUNA: real login -> real character fetch (confirmed exact `pos_x/y/z`/`job_main` from a
  prior manual PATCH) -> simulated movement -> position correctly persisted back
  (`pos_x/z` matched the exact forced offset). Visually verified under Xvfb: avatar renders,
  faces its movement direction, camera follows. Ability panes (M5) not done this pass. Native
  Linux build + link clean (mingw unavailable locally, same standing note as the rest of this
  client).

## 2026-08-02 (7)

- fix(town): a failed requeue now lands back in Town instead of a dead blank arena. Founder,
  live bug report: "if i dont requeue fast enough in GFD when i requeue it is like an empty game
  it says matchmaking fail." Root cause: `arena_server`'s own 60s "no lobby progress" watchdog
  (confirmed live in `var/logs/matchmaker-bots.log`: `No lobby progress in 60s (phase=0, 19/20
  connected) -- shutting down.`) can kill a freshly-matched game before a slow requeue finishes
  connecting to it -- `net_find_and_connect`/`net_connect` then correctly return failure, but the
  requeue handler had already `memset` `arena_state` to zero and did nothing further on failure,
  leaving the player staring at a blank arena (no heroes, no nodes) with no way back except
  force-quitting. That's REDGARDEN's own server/matchmaker code, explicitly out of scope this
  session (founder: "keep the battlegrounds working as is do not change that keep that server
  and matchmaking as is") -- fixed on the client side instead: a failed requeue now sets
  `in_town = 1`, landing the player back in a real, working Town scene where "QUEUE FOR
  BATTLEGROUNDS" can just be clicked again, same escape hatch the RETURN TO TOWN button already
  provides.

## 2026-08-02 (6)

- feat(town): "RETURN TO TOWN" button on the post-match win/lose screen, alongside the existing
  "OK - REQUEUE" button. Founder: "after a battlegrounds game in GFD i need the option to return
  to the town like a back button i only have requeue." Same cleanup path requeue already uses
  (close socket, reset `arena_state`/obstacles/rings/win_logged/net_picked/selected_unit_count),
  except no reconnect -- just sets `in_town = 1` so next frame's Town branch takes over. Only
  shown for the `--queue` path (there's no Town to return to for a direct `--connect` dev
  session, same gate Town's own entry uses).

## 2026-08-02 (5)

- feat(town): `apps2/battlegrounds_gui` now defaults to a real Town scene instead of connecting
  straight into the matchmaker. Founder: "we need the default to be town... a button top right
  to queue for battlegrounds which would trigger the matchmaker that leads to the draft and the
  game etc... build the world outside of the battlegrounds for now a flat plane is ok have it
  checkers grey and brown like a chessboard just make it the same size as the battlegrounds
  scene for now just with no buildings or trees or rocks yet." First slice of
  `HEADLESS_SESSION_NORTHSTAR.md` §3.4's "second scene," client-rendering-only for now (no
  headless MUD session wired up). `--queue`'s own `net_find_and_connect` call is deferred from
  startup to a real "QUEUE FOR BATTLEGROUNDS" button (top-right, per the founder's own
  placement); login still happens up front unchanged, landing the player in Town instead.
  `--connect` (direct dev connect to a known arena_server) is untouched -- no queue step to
  defer there, so it skips Town entirely, same as before. Town's own ground is a 12x12
  grey/brown checkerboard spanning the exact same footprint (`ARENA_HALF_EXTENT * 2.2f`) as
  battlegrounds' own ground plane -- "same size as the battlegrounds scene." Reuses
  battlegrounds' existing right-drag+wheel orbit camera and the same queuing "please wait"
  screen the post-match requeue button already used, so the window doesn't look hung during the
  up-to-60s matchmaker wait. Battlegrounds' own code is untouched -- the whole existing frame
  body is skipped via an early `continue` while in Town, not woven into it. Visually verified
  under Xvfb: TOWN label, checkerboard, and the button all render correctly (mingw unavailable
  locally, same standing note as the rest of this client).

## 2026-08-02 (4)

- feat(combat-log): second pane in `apps2/battlegrounds_gui`, bottom-right (mirrors the chat
  pane's bottom-left), showing damage taken and deaths for the current match. Founder: "add a
  second chat pane to GFD that shows the combat log." No wire packet carries discrete
  damage/kill events -- derived client-side by diffing `arena_state.heroes[]` frame-to-frame
  (`combat_log_scan`), which works identically for local play, net_mode, and replay/observing
  since all three write into the same `arena_state`. Attacker attribution via the existing
  `attack_target` field (already wire-synced, S170-162); unattributed damage (DoTs, creeps,
  skillshots) just shows the amount. Deliberately excludes heals to keep the log readable.
  Always visible, unlike the chat pane (not gated on a real player JWT -- bots/`--ticket`
  launches still show it). Native Linux build + link verified clean (mingw unavailable locally,
  same standing note as the rest of this client).

## 2026-08-02 (3)

- feat(chat): in-match MUD chat, `apps2/battlegrounds_gui`'s own real affordance surfacing
  `apps2/mud`'s persistent-world chat. `deliverChat` now relays say/yell/guild lines to IDUNA's
  new `POST /api/v1/chat/messages` (tell stays private, not relayed) via `idunaclient`'s new
  `PostChatMessage`/`GetChatMessages`. The Battlegrounds client polls every ~1.5s and renders a
  scrolling log; Enter opens a real chat-input line (consumes all other keybinds while focused,
  same "held/focused, not toggled" idiom the rest of the client already uses) and posts back as
  channel `battlegrounds`. Own JWT reused from login -- inert (no polling/posting at all) for
  bots/`--ticket`/dev-agent launches, which have no real player identity to chat as. One-way for
  now, named honestly: `apps2/mud` doesn't yet poll for Battlegrounds-originated messages, so MUD
  players don't see chat sent from a match -- real, separate, unbuilt follow-up.

## 2026-08-02 (2)

- feat(battlegrounds-gui): real fork of REDGARDEN's `apps/arena` into `apps2/battlegrounds_gui/`
  at commit `61baafb`. Corrects the previous approach (live cross-repo checkout of REDGARDEN at
  CI build time) per founder direction: "REDGARDEN isnt literally the GUI its supposed to be a
  starting place for the GUI like a clean fork." Self-contained -- own `packages/simulation` +
  `packages/common`, not sharing GFD's existing top-level `packages/`/`packages2/` (which have
  unrelated real content, e.g. `packages2/common/protocol.h` is a completely different wire
  protocol). CI now builds directly from this local copy, no cross-repo checkout step. Verified:
  clean standalone build, live login->ticket->connect reaches "20/20 connected" in a real match.
- feat(battlegrounds-gui): rebound ability casts from Q/W/E to 1/2/3 and added continuous,
  camera-relative WASD movement alongside the existing click-to-move (re-sent every ~100ms while
  held, same underlying move-to-point mechanic, no new wire packet). Fork-only -- REDGARDEN's own
  copy is untouched. Also fixed a real bug found live testing the founder's test account:
  `PLAY.bat` never set `IDUNA_BASE_URL`, so a real downloaded client always got "Could not reach
  login server" (its `127.0.0.1` default only works on the same box as IDUNA).

## 2026-08-02

- ci: `GoblinFoxDragon Factory` now also cross-compiles REDGARDEN's `apps/arena` (the real MUD
  GUI frontend, `REDGARDEN_GUI_NORTHSTAR.md`) as a Windows artifact. The existing
  `GoblinFoxDragon.exe` build target is `apps/lobby`, a stale SHANKPIT-lobby fork (window title
  still literally says "SHANKPIT", boots into SHANKPIT's own `SCENE_GARAGE_OSAKA`) -- not the
  real MMO client. New steps check out REDGARDEN (public repo, no token needed) and cross-compile
  its `apps/arena` with the same mingw/SDL2 toolchain this workflow already sets up, mirroring
  REDGARDEN's own `ci.yml` Windows step verbatim so the two pipelines can't silently drift.
  Bundled as `DragonsNShit_MUD_GUI_Client_*.zip` (exe + `SDL2.dll` + `PLAY.bat`, no
  `REDGARDEN_TICKET_SECRET` needed -- the client's own real IDUNA login screen mints a real
  ticket), uploaded alongside the existing artifacts.
- docs(redgarden-gui-northstar): Milestone 5's "no GUI login path" gap closed -- real
  email+password login screen shipped in REDGARDEN's `apps/arena` (`9c98342` + Winsock fix
  `e6fb748`), backed by a new IDUNA endpoint (`POST /api/v1/redgarden/self-ticket`, `5cd0fd0`).
  Working test account (`test@test.com`/`testtest`, character `TestWarrior`) verified end-to-end.

- docs(redgarden-gui-northstar): Milestone 5 (end-to-end validation) attempted honestly --
  marked PARTIAL, not DONE. Direct smoketest of `cmdBattlegrounds`'s exact real call sequence
  (`CreateCharacter`/`GetCharacter`/`MintBattlegroundsTicket`) against the live IDUNA service
  confirmed all three real, fast (ms-scale), and correct -- the strongest confirmation this
  identity/ticket chain has had yet. Also found `Xvfb`/`glxinfo` now work in this environment
  (Mesa software GL renders correctly) -- earlier "no display" notes are stale. Interactive
  telnet validation of the `battlegrounds` command's own text output was attempted repeatedly and
  abandoned as unreliable test-harness noise, not a code bug (the logic it calls is independently
  proven correct above); full interactive match-play validation (draft/cast/chain/credit) wasn't
  attempted -- two real, named, scoped blockers (a skillchain-aware bot heuristic; GUI-input
  automation tooling) remain open, not built here. New §9 in the northstar with the full honest
  breakdown.

- feat(job, mud): SMN gets real Avatar abilities -- founder, real-time: "zagan beleth vassago as
  summoner avatars GFD." New `job.SummonerAbilities()` (`summon_zagan`/`summon_beleth`/
  `summon_vassago`, real `Ability` data through the same `RecastTracker` every other job uses)
  wired into `apps2/mud`'s `abilitiesForJob`/`cmdJA`. Each avatar applies real
  `server/status` effects to the caster's live duel opponent, translated from that hero's own
  REDGARDEN kit (`docs/HEROES_VS0.md`) rather than invented: Zagan -> Bind (closest existing Kind
  to "stun," this package has no Stun Kind), Beleth -> Poison+Silence (her own real Q+W, ported
  faithfully since she already carries both on separate slots), Vassago -> Silence + a small
  direct hit (her real Q), damage clamped to never drop the opponent below 1 HP so it doesn't
  need to touch `duel.Manager`'s own win-condition path. Real, honestly-flagged simplification,
  not a full kit port: no armor-shred/mirror (Protect is a buff-only Kind in this package, not
  Category-flexible per Effect), no cast-refund, no delayed-burst zone, and no mob-targeted
  version at all (`mob.Mob` has no status stack yet -- a real, separate structural gap).
  2 new tests in `server/job` (data shape + a real `RecastTracker` integration check). Live
  smoke-tested via two telnet sessions (character creation, `setjob SMN`, duel challenge/accept,
  `ja summon_*`) -- confirmed `setjob SMN` correctly applies real SMN stats (HP:60/MP:90, matching
  `job.jobStats[SMN]`) and the duel flow works through the same command-dispatch path `ja` uses;
  the final `ja summon_*` output specifically wasn't reliably captured due to test-harness
  timing fragility (nc/FIFO scripting), not a code issue -- `go build`/`go test ./...` clean, and
  direct review of `cmdSummonAvatar`'s locking (no `gw.mu` held anywhere upstream of it, confirmed
  by reading `cmdJA`'s and `handle()`'s own call sites) rules out the deadlock this function's own
  `gw.mu.Lock()` would otherwise risk.

- docs(redgarden-gui-northstar): Milestone 4 shipped -- reward-credit hook. REDGARDEN's
  `apps/arena_server` now credits real Flow (100 win / 25 loss) to a match participant's
  persistent DragonsNShit character via new IDUNA `GET /api/v1/characters/by-player/:player_id`
  + the existing `gold/credit` endpoint. REDGARDEN `1fcf09e`, IDUNA `33b7a0d`. Milestone table +
  status line updated -- only Milestone 5 (end-to-end validation) left.

- fix(idunaclient): real IDUNA login exchange -- a genuine, previously-undiscovered production
  bug found while wiring REDGARDEN_GUI_NORTHSTAR.md Milestone 3. `Client.do()` used to send
  `IDUNA_AGENT_SECRET` directly as the Bearer token; IDUNA's real `jwt.Verify`-based
  `RequireAuth` middleware has always rejected that with 401 (confirmed live against the running
  service, not just theorized from reading the code) -- every call this package has ever made
  (`GetCharacter`/`CreateCharacter`/`CreditGold`/etc., shared by both `apps2/mud` and
  `apps2/server-go`) has been silently failing, masked by "best-effort, non-blocking" error
  handling at every call site. `characters` table on the live IDUNA instance was empty; this is
  why. Fixed: `New()` now also reads `IDUNA_AGENT_NAME`; a new `ensureToken()` performs the real
  `POST /api/v1/auth/agent` exchange and caches the resulting JWT (refreshed within 60s of its
  real 1-hour expiry), used by every existing method for free since they all route through
  `do()`. Verified live end-to-end: a real character now creates successfully against the
  running IDUNA service (previously 401). 4 new tests. Backward-compatible with every existing
  test in this package (none set `IDUNA_AGENT_NAME`/`IDUNA_AGENT_SECRET`, so `do()` skips the
  login step exactly as before for those).

- fix(mud): real, stable player_id for IDUNA character creation -- another real gap found in the
  same pass. `gw.iduna.CreateCharacter`'s `player_id` argument was `conn.RemoteAddr().String()`
  (a TCP socket address) -- not a valid UUID, and different every reconnect. IDUNA's own ticket
  endpoints `uuid.Parse` the player_id and would reject it outright. New `mudPlayerIDCache`
  (`var/mud-player-ids.json`, same load/persist shape as the existing `mudCharCache`) mints and
  persists a real `crypto/rand` UUIDv4 per character name on first use -- stdlib-only, no new
  dependency (same reasoning `packages/common/hmac_sha256.h`'s own doc comment already gives for
  not linking a crypto library). Does NOT solve real player identity (OAuth/email login for a
  telnet interface is a genuinely separate, larger, undesigned question) -- only makes the
  existing anonymous, name-keyed identity model stable and UUID-shaped instead of an ephemeral
  socket address, flagged honestly rather than oversold.

- feat(mud): `battlegrounds`/`bg` command -- REDGARDEN_GUI_NORTHSTAR.md Milestone 3, the
  Battlegrounds entry point (§4.3's own open question, resolved as a discrete command, same
  shape as `cmdGo`'s own zone-transfer precedent, which §4.3 named as the closest existing one).
  Fetches the player's real character via IDUNA, mints a real REDGARDEN connect ticket via the
  new `idunaclient.MintBattlegroundsTicket` (IDUNA's new `POST /api/v1/redgarden/player-ticket`,
  see that repo's own CHANGELOG), and prints the exact `red_garden_arena --queue <host>
  --matchmaker-port 7778 --ticket <hex>` command line to run -- a telnet session can't launch a
  GUI process itself, so this is the honest, real "hand off" a text interface can do (REDGARDEN's
  own new `--ticket` flag, see that repo's own CHANGELOG, is what makes the printed command
  actionable). Job pick is a stub, not a menu -- Warrior is the only job Milestone 1 ported, so
  there's nothing to choose between yet.

- docs(redgarden-gui-northstar): Milestone 2 shipped, same session as Milestone 1 below -- real
  skillchain resonance detection in REDGARDEN's `arena_game.c`. A straight C port of this repo's
  own `server/skillchain.go` combination table (same real tiers/multipliers), tracked per-target
  and closed via a new `apply_weapon_skill_damage` choke point every real weapon-skill cast
  routes through. Verified real: Warrior's own Q(Scission)->R(Induration+Reverberation) closes an
  actual Tier 2 Distortion chain per the table. REDGARDEN `21ad0dc`. Milestone table + status
  line updated to match. Milestones 3-5 (entry-point hook, reward-credit hook, end-to-end
  validation) still ahead.

- docs(redgarden-gui-northstar): Milestone 1 shipped -- Warrior, the first DragonsNShit job
  ported into REDGARDEN's Battlegrounds as real ability content. Founder redirect this session,
  after "can i log into gfd gui yet?": "ok i asked for the mmorpg i provided the inputs continue
  to work on that." Real code landed in the sibling REDGARDEN repo (`cbcd4ed`) -- Q Hard Slash/W
  Power Slash/R Frostbite, real Great Sword weapon skills from this repo's own
  `server/skillchain.CanonicalWeaponSkills`, matching `server/job.jobStats[WAR]`'s real stat
  block. REDGARDEN has no TP resource, so MP substitutes for `server/combat.TPWSThreshold`'s 100
  TP -- an honest amendment, not a literal port, per founder direction ("we want our old systems
  like skillchains etc [to] work with redgarden affordances"). `docs2/REDGARDEN_GUI_NORTHSTAR.md`
  milestone table + status line updated to match. Milestones 2-5 (skillchain detection in
  `arena_game.c`, entry-point hook, reward-credit hook, end-to-end validation) still ahead.

- feat(idunaclient, mud): `apps2/mud`'s Flow (gold) is finally synced back to IDUNA on
  disconnect. Backend-unification follow-up, closing the real gap the previous correction found:
  `p.flow` was read from IDUNA on connect but never written back, because IDUNA had no way to
  credit gold at all -- only deduct. Now that IDUNA's own new `PATCH .../gold/credit` exists
  (`IDUNA` commit `1b7f43d`), added the symmetric client method `idunaclient.CreditGold`
  (3 new tests, `server/idunaclient/idunaclient_test.go` -- this package's first test file at
  all, DeductGold and every other existing method shipped with zero coverage; backfilling those
  is separate, larger work, not attempted here). Wired into `apps2/mud`'s own connect/disconnect
  flow: `startingFlow` captures the real balance right after the existing fetch-or-create IDUNA
  call, and the disconnect handler now computes the session's net Flow delta and calls
  `CreditGold`/`DeductGold` accordingly -- same silent-discard, best-effort convention the
  adjacent level/XP/position sync calls already use. `GOWORK=off go build ./...`/`go test ./...`
  clean.

- docs: corrected a real, load-bearing wrong claim in `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`
  ("apps2/mud has no real IDUNA persistence"). Found while investigating what "share live state"
  actually requires: that claim's own grep searched for the literal strings `idunaclient`/
  `idunaClient` and found nothing beyond construction -- but the real field is `gw.iduna`
  (different name, different grep target) and it genuinely is called: `gw.iduna.GetCharacter`/
  `CreateCharacter` on connect (seeding level/XP/gold from a real IDUNA row, via a local
  name→ID cache persisted to `var/mud-chars.json`), `gw.iduna.UpdateCharacterLevel`/
  `UpdatePosition` on disconnect. What's still true: only synced at connect/disconnect, not
  continuously like `apps2/server-go`; and `p.flow` (gold) is read on connect but never written
  back -- traced why: IDUNA's own `/characters/:id/gold` endpoint only accepts deducting gold
  server-side, no credit/add endpoint exists at all, so completing this needs new IDUNA API
  surface, real cross-repo work not attempted here. A partial fix covering only the decrease
  direction was considered and deliberately rejected (silently wrong for the increase case is
  worse than clearly not-done). Corrected in place in the audit doc's own §1 and §4, not
  silently overwritten.

- feat(server-go): respawn's XP penalty now persists back to IDUNA. Backend-unification
  follow-up, closing the "XP earned isn't written back to IDUNA yet" gap named in
  EMILY/BACKLOG.md's own unification item. `idunaclient.Client` already had a real,
  ready-to-use `UpdateCharacterLevel(characterID, level, currentXP)` method (PATCHes
  `/api/v1/characters/:id`) -- not built here, just not wired into this handler yet. The respawn
  handler now calls it after computing the new post-penalty `currentXP`, fire-and-forget via a
  goroutine + log-on-error, same pattern `PacketSkillXP`'s own `IncrementSkill` call already
  uses just above it (never blocks the UDP read loop on an HTTP round trip).
  `GOWORK=off go build ./...`/`go test ./...` clean.

- fix(server-go): respawn XP penalty was using the wrong percentage (10% instead of the real,
  live 8%). Backend-unification follow-up, same-day correction to the respawn-packet change
  right below. Found while trying to wire real per-player XP in: `HPState.RaiseDefault`
  (what the respawn handler called) applies `combat.DefaultRaisePenaltyPct` (10%) -- checked
  against `apps2/mud`'s own actual live behavior and that's not the real number. `apps2/mud`'s
  real `cmdHome` hand-computes an 8% penalty (`homepoint.DefaultXPPenaltyPct`) and doesn't call
  `HPState.Raise` at all -- a claim the respawn-packet CHANGELOG entry below got wrong ("apps2/
  mud's own real 'type home' flow... HPState.Raise" -- it doesn't). `server/homepoint`'s own
  `ReturnHome()` implements that real 8% mechanic too, but `cmdHome` duplicates it by hand
  instead of calling it (pre-existing in `apps2/mud`, unrelated to this fix, not touched). Fixed
  by passing `homepoint.DefaultXPPenaltyPct` explicitly into `HPState.Raise` (which does accept
  an arbitrary percentage) instead of trusting its own unrelated 10% default -- still reuses
  `Raise`'s real HP-reset/`IsKO`-clear behavior, just with the actually-live percentage. Also
  wired real per-player XP: `fetchCharacterCombatStats` now also returns IDUNA's real
  `Character.CurrentXP`, stored on `clientInfo` and mutated locally on respawn (not written back
  to IDUNA yet -- a further, named gap). 1 existing test updated for the new return signature,
  no new test needed for the percentage fix itself (`Raise`'s own arbitrary-percentage behavior
  is already covered upstream). `GOWORK=off go build ./...`/`go test ./...` clean.

- feat(server-go): respawn packet closes the KO loop the previous change opened. Backend-
  unification follow-up (EMILY/BACKLOG.md item 2). New `PacketRespawn`/`PacketRespawnResult`
  (`packages2/common/protocol.go`) -- a KO'd player's only way back on this backend, `apps2/mud`'s
  own real "type home" flow (`knockOut()` + `HPState.Raise`, 8% XP penalty) reduced to its core
  mechanic: `RaiseDefault(0)`, always against 0 XP since real per-player XP tracking doesn't
  exist in `apps2/server-go` yet (unlike `apps2/mud`'s own `p.charXP.CurrentXP`) -- a real,
  already-tested degenerate case (`server/combat`'s own `TestRaise_ZeroXPNoPanic`), not a crash
  risk, just an honestly-incomplete penalty number until real XP tracking lands here too, named
  in the code rather than silently wrong. No new tests -- `RaiseDefault`'s own behavior is
  already covered by 5+ existing tests upstream, same reasoning the KO-state change just used.
  `GOWORK=off go build ./...`/`go test ./...` clean.

- feat(server-go): real KO state via `server/combat.HPState`, gates further weapon-skill casting.
  Backend-unification follow-up (EMILY/BACKLOG.md item 2). `clientInfo.hp`/`maxHP` (raw ints)
  replaced with `hpState *combatTp.HPState` -- `apps2/mud` itself drives KO through its own
  separate `homepoint.State.IsKO` field rather than calling `HPState` directly, but the mechanics
  are the same shape, and reusing the already-tested type (`NewHPState`/`TakeDamage`/`IsKO`, 17
  existing tests in `server/combat/death_test.go`) beats re-deriving damage-floor/KO logic by
  hand. `PacketWSCast` now rejects casting from a KO'd caster and casting *at* an already-KO'd
  target (`ErrAlreadyKO`'s own failure mode, guarded before it can fire). Deliberately NOT
  implemented: any respawn/home-point flow once `killed=true` -- `apps2/mud`'s own `knockOut()`
  leaves a KO'd player waiting until they actively type `home` (8% XP penalty) or get Raised;
  porting that full flow is separate, larger follow-up work, not attempted in this slice, so a
  KO'd player on this backend currently just... stays KO'd forever. Named honestly, not hidden.
  `GOWORK=off go build ./...`/`go test ./...` clean (existing 6 tests, no new ones needed --  the
  underlying `HPState` behavior this wiring calls is already covered upstream).

- feat(server-go): real IDUNA job/level fetch on connect + real HP tracking, closing two of
  Sprint 3's own named gaps. Backend-unification follow-up (EMILY/BACKLOG.md item 2). New
  `fetchCharacterCombatStats` -- calls `idunaClient.GetCharacter` on `PacketConnect` (same
  best-effort tone `PacketTelecrystalUse` already uses toward IDUNA lookups: falls back to
  WAR/level 1, not a hard connection reject, if IDUNA has no character row yet or the fetch
  fails outright), computes starting HP via `jobpkg.HPAtLevel` -- the same formula `apps2/mud`'s
  own character sheet already uses, not reinvented. `clientInfo` gained real `jobMain`/`level`/
  `hp`/`maxHP` fields (HP itself in-memory only, same as `apps2/mud`'s own `p.hp` -- not
  persisted to IDUNA, matching every MMORPG's own "a life's current HP isn't durable state"
  convention). `PacketWSCast` now actually subtracts `result.Damage` from the target's real HP
  and reports `target_hp`/`target_max_hp`/`killed` in `PacketWSResult` -- Sprint 3 only ever
  reported a damage number without touching anything; this is the first slice where a weapon
  skill actually hurts someone. 1 new test (`fetchCharacterCombatStats`'s WAR/level-1 fallback,
  verified deterministically by pointing at an unreachable IDUNA URL rather than a live server).
  `GOWORK=off go build ./...`/`go test ./...` clean. Still not done: no death/respawn handling
  once `killed=true` fires (the target just sits at 0 HP), enmity untouched, `apps2/mud`'s telnet
  players still don't share this state.

- feat(server-go): real weapon-skill casting + skillchain resonance wired into `apps2/server-go`'s
  UDP loop. Founder: "yes unify the backends" -> "whatever makes sense" -> "clean builds first"
  (backlog dump + sprint plan, Sprint 3 -- EMILY/BACKLOG.md). First real slice of
  `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`'s own unification recommendation: rather than
  rewriting `apps2/mud`'s RPG logic, `apps2/server-go` now directly imports the same tested
  `server/combat`/`server/skillchain` packages `apps2/mud`'s own `cmdAttack`/`cmdWS` already use.
  New wire packets `PacketWSCast`/`PacketWSResult` (`packages2/common/protocol.go`). Every
  `BtnAttack` now feeds real TP (`combatTp.TPState.AddTP`, flat 1H-sword delay assumed -- no real
  weapon/gear system wired into this backend yet) alongside the existing `HandleShankFire`
  hitscan, not replacing it. `PacketWSCast` validates against `server/skillchain`'s real weapon-
  skill registry, checks real TP via `CanWeaponSkill`, and scores a real skillchain against
  whatever last landed on the target (`server/skillchain.Chain`, PvP-shaped -- targets another
  connected client, not a mob, since this backend has no mob registry the way `apps2/mud` does).
  Decision logic extracted into a standalone `resolveWSCast` (same "extract for testability"
  reasoning `main_test.go`'s pre-existing `TestParseUserCmd` already established for
  `parseUserCmd`) -- 4 new tests (unknown skill, no-chain damage, a real Shining Blade -> Burning
  Blade Tier-2 Fusion closure, chain-window-expiry). Named, not silently skipped: no real HP/
  death tracking exists for `apps2/server-go`'s own connected players yet (`clientInfo` has no HP
  field at all), so damage is a placeholder number reported in the result packet, not applied to
  anything -- a separate, larger follow-up. `GOWORK=off go build ./...`/`go test ./...` clean
  across the whole module throughout ("clean builds first" taken as a continuous constraint).

- refactor: `gil` -> `flow`/`Flow` across the whole `dragonsnshit` module. Founder: "convert gil
  to flow" (backlog dump + sprint plan, Sprint 2 -- EMILY/BACKLOG.md). REDGARDEN already has
  real, shipped "Flow" economy terminology (S170-175); DragonsNShit's own currency naming now
  matches instead of keeping FFXI's "gil". Renamed: `apps2/mud/main.go`'s `player.gil` field (all
  call sites, all in-game command output text), the `"gil-drop"` loot item ID and its "100 Gil"
  display name (-> `"flow-drop"`/"100 Flow"), `server/quest`'s `RewardGil`/`Result.Gil` fields (+
  `trapx_chains.go`'s 20-odd `RewardGil:` literals), `server/auction`'s `ErrInsufficientGil`/
  `buyerGil` (+ `TestBuyInsufficientGil` -> `TestBuyInsufficientFlow`), `server/market/ah.go`'s
  own comments. `GOWORK=off go build ./...` and `go test ./...` clean across the whole module
  before and after. Two docs from earlier today (`REDGARDEN_GUI_NORTHSTAR.md`,
  `DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`) updated to reflect the completed rename rather than left
  silently stale.

- docs: REDGARDEN_GUI_NORTHSTAR.md, two real-time corrections in a row on the Battlegrounds
  design. Founder #1: "some of the docs say we arent bringing redgardens gameplay just the ui
  thats not right i want dragonsnshit mmo to feel like redgarden like battlegrounds for
  dragonsnshit is redgarden." Corrected the doc's original thesis (REDGARDEN as rendering-only
  skin, DragonsNShit's own systems replace REDGARDEN's gameplay underneath) -- wrong. REDGARDEN's
  full real-time combat framework (`arena_server`/`apps/matchmaker`, Q/W/R slot UI, item shop,
  node-capture map) becomes DragonsNShit's Battlegrounds, an instanced PvP mode, same relationship
  WoW Battlegrounds/FFXI's own minigames have to their main games. Founder #2, immediately after:
  "like not the same literal game loop maybe but we want to amend our ould systems like
  skillchains etc work with redgarden affordances." Refined further: the process/loop separation
  stays (Battlegrounds is still its own spawned-per-match process, not merged into the persistent
  world's own loop), but the *ability content* cast through REDGARDEN's Q/W/R slots is
  `apps2/mud`'s real job/weapon-skill/skillchain system ported into `arena_game.c`'s slot
  machinery -- a Battleground combatant picks a job (Warrior, Black Mage, ...), not one of
  REDGARDEN's 28 fixed heroes, and that job's real abilities render through REDGARDEN's existing
  cast-ring/projectile/zone-circle vocabulary, with real skillchain resonance between players'
  casts. §§1/4.1/4.2/5/6 rewritten in place across both corrections, each labeled and dated so
  the doc's own reasoning history stays legible. Milestone table now: port Warrior's real kit
  into `arena_game.c` first, then skillchain resonance, then the entry-point/reward-credit hooks,
  then end-to-end validation.

- docs: DragonsNShit has two non-unified backends — audit + bridge-target correction. Founder:
  "continue dragons n shit" (continuing "do the docs first"). New
  `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md`, correcting a real, load-bearing wrong assumption in
  both of today's earlier docs (`REDGARDEN_GUI_NORTHSTAR.md`, `REDGARDEN_MUD_BRIDGE_SPEC.md`),
  both written as if `apps2/mud` were the only DragonsNShit backend. It isn't: found
  `apps2/server-go`, a real UDP server on `:6969` with a real IDUNA-JWT-authenticated protocol
  (`PacketConnect`/`PacketUserCmd`/`PacketChat`, real Telecrystal travel + crafting + skill-XP,
  all actually calling IDUNA) -- `apps2/mud`'s own `idunaclient` is imported and instantiated but
  never once actually called (confirmed via repo-wide grep), a dead field, not real integration.
  `apps2/server-go`'s combat is SHANKPIT-shaped hitscan (`HandleShankFire`), not `apps2/mud`'s
  RPG job/skillchain/enmity depth -- the two backends don't share state at all. Also found
  `apps2/lobby`, an existing 884-line C client already targeting `apps2/server-go`'s protocol,
  smaller than REDGARDEN and blocked by the same `GL/glu.h` dependency issue that's hit this
  monorepo repeatedly -- reinforces REDGARDEN as the stronger client foundation, not a reason to
  abandon the direction. Revised recommendation: port `apps2/mud`'s RPG logic to run inside
  `apps2/server-go`'s authoritative loop, backed by IDUNA's already-existing
  `characters`/`character_skills`/`character_equipment`/`character_inventory` schema, before
  REDGARDEN's own bridge work lands -- REDGARDEN then targets `apps2/server-go` directly as a
  peer of `apps2/lobby`, no new listener needed (superseding `REDGARDEN_MUD_BRIDGE_SPEC.md`'s own
  "bolt a listener onto the text MUD" design, marked superseded in place, kept for its still-real
  movement/targeting gap-finding). `REDGARDEN_GUI_NORTHSTAR.md`'s milestone table rewritten in
  place to reflect this. Registered in golden-docs-index.

- docs: REDGARDEN ↔ apps2/mud packet-level bridge spec. Founder: "continue dragons n shit do the
  docs first." New `docs2/specs/REDGARDEN_MUD_BRIDGE_SPEC.md`, the concrete next layer under
  today's earlier `REDGARDEN_GUI_NORTHSTAR.md`, written against the real code on both sides
  (`REDGARDEN/packages/common/protocol.h`'s actual structs, `apps2/mud/main.go`'s actual `cmd*`
  handlers) rather than assumed shapes. Reuses REDGARDEN's real HMAC connect-ticket handshake
  verbatim (same scheme SHANKPIT/shankpit-460 already share). Maps REDGARDEN's real packets onto
  apps2/mud's real functions (`ArenaAttackCmd`→`cmdAttack`, `ArenaCastCmd`→`cmdWS`,
  `ArenaShopBuyCmd`→`cmdShopBuy`); drops `PACKET_ARENA_PICK` entirely (no hero draft in a
  persistent-character MMO). Two real gaps found and named while writing this, not glossed over:
  `apps2/mud` has zero continuous intra-zone movement server-side today (`cmdGo`'s own code
  confirms `n/s/e/w` only ever teleports between zones; `cmdAttack`'s auto-approach snaps
  position directly onto the target) -- `PACKET_ARENA_MOVE` has nothing to bridge onto without
  real new server code, reframing the northstar's own Milestone 3 scope; and
  `PACKET_ARENA_ATTACK`'s hero-slot-index targeting has no equivalent against apps2/mud's
  string-ID mob/player targeting. Proposes a genuine `MudEvent` list to replace REDGARDEN's flat
  HP-delta-driven visual-effect idiom (`attack_flash`/`heal_flash`), which can't carry
  skillchain/status-effect semantics the way a flat HP diff can't. UDP, port 2324 proposed
  (resolves one of the northstar's own open questions). `REDGARDEN_GUI_NORTHSTAR.md` updated
  in place: 2 open questions resolved/refined, 2 new ones surfaced, Related Docs table updated.
  Registered in golden-docs-index. Spec only, no code.

- docs: REDGARDEN-as-GUI northstar. Founder, real-time: "can we graft redgarden frontend onto GFD
  mud as a gui to make our mmorpg?" → "i dont care how you do it fork redgarden into GFD write the
  northstar this is the mmo. this is dragonsnshit" → "cli will continue to work" → "redgarden as a
  gui" → "like old school runescape." New `docs2/REDGARDEN_GUI_NORTHSTAR.md`: forks REDGARDEN's
  real-time SDL2/OpenGL client (rendering/input machinery only -- click-to-move, hero-silhouette
  rendering, Q/W/R cast-ring/projectile/zone-circle UI, item-shop chrome, connect-ticket auth; not
  its MOBA hero-kit combat sim) onto `apps2/mud`'s real, already-shipped FFXI-parity Go MMORPG
  backend (22 jobs, skillchains/magic bursts, enmity, conquest, NM spawns/treasure pool, crafting
  guilds, parties/linkshells -- telnet `:2323` today) as a second, parallel client protocol
  alongside a new binary listener; telnet keeps working unchanged, per founder direction. Design
  call: REDGARDEN contributes the rendering grammar, `apps2/mud` keeps owning the RPG mechanics
  underneath -- no REDGARDEN hero identity carries over, only its UI vocabulary. Amends
  `docs2/MMO_NORTHSTAR.md`'s "Integration Architecture" section (frontend line updated from
  "SHANKPIT runtime, extended" to point at the new doc) and flags that MMO_NORTHSTAR's own
  milestone table (last updated 2026-06-21) is stale against `apps2/mud`'s real shipped state --
  a large body of FFXI-parity systems work (S76-S87) landed since without that table being
  updated. 7-milestone table, spec only, all NOT STARTED past this doc itself. Registered in
  `EMILY/context/golden-docs-index.md`.

## 2026-07-23 (8)
- fix(mud): found live, playing again after redeploy -- worm Poison was still lethal despite tonight's earlier "Worm is Slow-only, not Poison" fix. Root cause: `mobSpellPool`'s map keys were capitalized ("Worm", "Slime", "Lizard"...) but real mob IDs are always lowercase ("worm-meadow-4", "slime-swamp-2"...); `strings.Contains(mobID, prefix)` is case-sensitive, so the lookup never matched anything for *any* mob kind and every single mob silently fell through to the generic fallback pool, which still includes Poison. The earlier fix was correct in intent but never executed at runtime. Fixed by lowercasing every key in `mobSpellPool` to match real mob IDs. `go test ./server/...` clean, rebuilt, redeployed live, re-verified in person: killed a full worm (5 hits) with zero Poison procs, leveled Lv.1 to Lv.3.

## 2026-07-23 (7)
- fix(mud): found live, testing "does the economy work" as a real player -- shop, bazaar, bank, quest-accept/quest-turn-in/quests, npcs/talk, equip/gear, and craft are all real, fully working commands (confirmed live: bought an Echo Drop for 50 of 500 starting gil, accepted a real quest from a real NPC, checked bank balance) that cmdHelp never mentioned. A new player had zero way to discover any of them short of reading the source -- including Echo Drop, the exact 50-gil cure that would have prevented the Poison death found and fixed earlier tonight. Added an "Economy & items" and "Quests & NPCs" section to the in-game help listing the core early-game commands; deliberately still not listing all 100+ commands that exist (job-specific spells, FIELDOFFICE/TRAPX faction-war systems like k9-deploy/district/enforcement/integrity/tech-pressure) since that's advanced/endgame surface that would trade one discoverability problem for an unreadable wall of text. Redeployed live.

## 2026-07-23 (6)
- docs2/HERO_BRIDGE_PREREQUISITES.md: gap analysis answering "do we weave multiverse lore in" (honest answer: not yet) and "what are the prerequisites to bridge it." NM registration and loot are both real, already-working systems -- not blockers. The one real gap: every zone in the game is hand-written directly in apps2/mud/main.go's initWorld(), with no data-driven zone format the way data/items.json exists for items. Names the actual prerequisite chain (minimal zone data format -> pick one of HERO_CONTENT_FRAMEWORK.md's five worked examples -> build it -> numbers pass last). Published to okemily.com as "Not Woven In Yet."

## 2026-07-23 (5)
- feat(education): EduVM Phase 0 (docs2/EDUCATION_CURRICULUM_NORTHSTAR.md) -- arrays. New `array(N)` declaration syntax (`let arr = array(5);`), indexed read/write (`arr[i]`, `arr[i] = v;`), bounds-checked at runtime against a shared 256-slot arr_mem[] pool (new EDU_OP_LOAD_ARR/EDU_OP_STORE_ARR opcodes, new `[`/`]`/`array` lexer tokens). Verified end-to-end with a real bubble sort compiled and executed against the actual VM (packages/education/edu_test_arrays.c, 14 assertions, first test file this package has ever had). **Found and fixed a real, serious pre-existing bug while building the test harness**: `edu_compile_source`'s own `memset(out, 0, sizeof(*out))` was wiping the caller-supplied `bytecode`/`bytecode_cap` fields to NULL/0 *before* using them to init the bytecode writer -- every real compile has been failing "bytecode overflow" on its very first emitted byte since this function was written, meaning the Architect's Orb terminal's F7 compile in apps/lobby has likely never actually produced working bytecode in practice. Fixed by preserving the caller's buffer/cap across the reset. go build clean; apps/lobby's own C build still blocked by the pre-existing, unrelated, already-documented missing GL/glu.h system dependency (sudo-gated, not touched here).

## 2026-07-23 (4)
- docs2/HERO_CONTENT_FRAMEWORK.md: story-first process for turning any TYLER/multiverse_heroes.md entry into a dungeon, NM/raid boss, and loot drop, grounded in the real engine (server/mob.Mob, server/nm.NMSpawn placeholder/window/respawn model, server/itemdef.Item Category/JobMask/Flags, server/loot.Pool). Five fully worked examples (Bacon, Zagan, Nidhogg, Cain, Tesla) -- no numbers/stats anywhere yet, per the same docs-before-software discipline the hero compendium itself established. Golden-indexed as GFD-HERO-FRAMEWORK.

## 2026-07-23 (3)
- fix(mud): found live, playing as a real new character (Custodian) right after the melee-range fix -- worm's 20%-chance debuff proc could cast Poison (flat Potency=10, ticking every 1s game-loop tick for up to 30s = up to 300 total damage) against a level 1-5 character with only 90-150 max HP. Died to it once, for real, mid-session. Worm is this game's own zone-0 tutorial mob (worm.go's own doc comment: "mostly passive"); a single proc from the very first mob a new player fights could solo-kill them. Removed Poison from Worm's mobSpellPool, left with Slow only (a better flavor fit for a worm anyway, and non-lethal) -- Poison unchanged for Slime/Chaos/Leech, which aren't the tutorial mob. go build/test green, redeployed live under gfd-mud.service.
- fix(mud): deployed apps2/mud under real supervision for the first time (ops/systemd/gfd-mud.service — built 2026-06-27, never run under systemd before tonight) and found combat was completely non-functional from spawn. Root cause: MeadowWormSpawns places every worm 25-35 units from the town-centre spawn point (0,2,0) — by its own design comment, "away from the town centre" — but DefaultPlayerMeleeRange is only 3.0, and there's no intra-zone movement command (n/s/e/w means zone travel, not local positioning). Every new player's first `attack` set a target that could never land a hit, and tickAll's error handling only messaged on ErrMobDead/ErrMobNotFound, so ErrOutOfRange failed in total silence — indistinguishable from a hung connection. A pre-existing test (TestMeadowWormSpawns_OutsideTownRadius) had locked in the exact distance that caused this without anyone connecting it to the melee-range constant. Fixed: cmdAttack now auto-approaches (snaps the player to the target's position) when out of range at the moment of targeting; tickAll's ErrOutOfRange branch now tells the player why nothing is happening instead of continuing silently, as defense in depth for mobs that wander mid-fight. New exported mob.Dist + 2 tests (TestDist, TestMeadowWormSpawnsOutsideDefaultMeleeRange — the latter pins the geometry fact itself so a future spawn-layout change can't silently reintroduce the bug). Live-verified: registered a real character (Custodian), confirmed combat landed, leveled 1→4 in one sitting. go test ./... green.

## 2026-07-23
- docs2/EDUCATION_CURRICULUM_NORTHSTAR.md: scoping pass for teaching CS algorithms (sorting, knapsack) via the existing EduScript VM (packages/education) and its Architect's Orb terminal (apps/lobby, F7 compile/F8 run). Confirmed the VM/world-object binding (switches, gates, crates, bridges, portals) is real and live, but has no array/indexed-memory opcode and no user-defined functions — the one prerequisite every algorithm module needs, scoped as Phase 0. Also corrected a founder claim: the education system was never actually merged into SHANKPIT's apps2/lobby despite the "yolo" commit that created that folder — verified directly, zero education/VM code there; it lives only in GFD today. Golden-indexed as GFD-EDU-CURRICULUM. Design only, no code yet.

## 2026-06-27
- S129-10: Art direction reference sheets at docs2/art_direction_tiers.md — 5-tier palette guide (Initiate→Endgame), per-armor poly budgets + UV spec + shader rules
- S130-02/03/04: npcattention tick wired; disguise items (Guard/Civilian/Merchant); WEAR + REMOVE DISGUISE commands; sneak feeds attention state
- S129-07: equip/unequip job+level enforcement via itemdef.Registry; stat delta broadcast; gear list shows stat totals

- S129-06: gear.ComputeStats() + CanEquip() using itemdef.Registry; DefID added to ItemEntry; 10 tests

## 2026-06-25
- feat(watcher): TRAPX vigilante anomaly spawn system — DisruptionDebt accumulator, 4 archetypes (Founder/Chemist/Apparition/RiotBreaker), 3 power tiers, chaotic-neutral targeting by Trust score, 19 tests
- docs2/INVENTORY_EQUIPMENT_NORTHSTAR.md: FFXI-era inventory+equipment northstar with art direction (low-poly)
- server/npcattention: per-NPC stealth awareness (Hitman C47 parity) — disguise factions, suspicion [0,100], witness system, Scene tick
- server/nm: HillsNMs + CavesNMs + AllStartingZoneNMs — all 4 default zones now have NM spawn definitions
- server/mob/hills.go + caves.go: rabbit/beetle/wolf (Hills), bat/spider/skeleton (Caves) with spawn sets
- data/items.json: 52-item seed (weapons/armor/accessories/consumables/crystals/materials/key items)
- server/inventory: bag container (Bag/Stack/Mog, stack merge, Rare conflict, Gobbiebag expand, key items)
- server/itemdef: item definition registry (JobMask, ItemFlags, Category, Registry/LoadJSON/ByID/ByName, CanEquip)

- feat: S128-06 Scar system (scar/scar.go — Registry, 4 causes, +5%/scar visibility bonus, ScarBurn, MUD command, 11 tests) + S128-07 K9 Merciless Operation (k9/operation.go — 4-phase, 3 counterplay lanes, 18 tests) (Apple #3870)

## 2026-06-24
- feat: S125-04 TRAPX economy — 5 items, 3 TRAPX vendors, enforcement.DistrictPressure price scaling (Apple #3656)
- feat: S126-15 TRAPX craft recipes — repaired-bike/faction-gear/atlas-page; cartography.DiscoverAll on atlas-page (Apple #3644)
- feat: S126-14 campaign battle mode — server/campaign, 10 nodes, /campaign join/status, weekly reset, 15 tests (Apple #3641)
- feat: S126-13 NM respawn scheduler — nm.Registry, NMRespawnScheduler, RespawnMinutes, announceNMPop, 8 tests (Apple #3566)
- feat: S126-04 bilingual party chat — BOTH lang, setlang command, per-recipient AT expansion in deliverChat (Apple #3538)
- feat: S126-03 world event broadcast — server/worldevent, /api/world-events endpoint, faction war→worldEventReg, 12 tests (Apple #3536)
- feat: S126-02 weather → mood loop — Storm:Fear+15, Rain:Fatigue+10, Clear:Fatigue-5 per district tick (Apple #3533)
- feat: S126-01 NPC schedule system — server/schedule, 3 NPCs seeded, hourly tick → broadcastAll, 14 tests (Apple #3531)
- feat: S125-13 Mog House personal storage — server/moghouse, 50-item cap, Store/Retrieve/List MUD commands, 20 tests (Apple #3528)
- feat: S125-12 server/auction — standalone AH engine (List/Buy/Cancel, 5% fee, 15-min expiry, 15 tests) Apple #3526
- feat: S125-03 TRAPX faction war engine — 72h cycle, 24h conflicts, FO win condition, war/fw MUD command (Apple #3514)
- feat: S125-02 zone presence — departure broadcasts, cmdExamine player inspect (Apple #3513)
- feat: FFXI auto-translate — server/autotranslate 72 JP/EN phrases, ExpandLine(), at MUD command, [alias] token expansion in say (Apple #3505)
- S123-05: VS0 Detroit slice — fo-school-1 seeded, Jiangshi FO pre-held, alertness=35 preset, takecontrol entry, Emily OS ambient voice (10 fragments)
- S123-04: flip phone MUD interface — 5 tabs (FO/heat/receipts/crew/CAST), CRT box-drawing, Watcher alertness contribution, districtIDForZone
- S123-03: multi-timeline branch system — Branch/Registry, rogue_swarm auto-branch, 'timeline' MUD command, conflict detection, 16 tests
- S123-02: TYLER ledger bridge; 4 TYLER verb types, 4 CAST lore docs, 'terminal' MUD command, archive entry receipts
- S123-01: TYLER scene cluster 200–207; 8 districts, portal connections (VS0+Tyler's route), TYLER faction NPCs (Heikegani/Kuroshio/Yōkai/Eastwind/Jiangshi), urban terrain for 205-207
- S122-06: city/district/align/broadcast/enforcement MUD commands; watcher+enforcement+neighborhood wired into initTRAPXCity and tickAll
- S122-05: TRAPX faction rep overlay — Frequency/Bloc/Procurement Houses on fame.Nation; rank-gated benefits, 11 tests
- S122-04: TRAPX RPG class unlock chains — 8 chains (DRK/BST/BRD/SAM/SMN/BLU/GEO/RUN), 24 quests total, job-stone rewards, wired into questBank
- S122-03: Neighborhood personality — Tolerance/Pride/Cohesion/Visibility axes, Fear/Fatigue mood drift, myth seeding (10 lore fragments per district), 23 tests
- S122-02: Watcher (alertness/trust/bias per district) + Enforcement (5-level state machine: Quiet→Lockdown, cop density 0-8, K9 eligibility, FO effects)
- S122-01: TRAPX city scene cluster — 5 districts (200-204) + zone exits + 7 city NPCs (mini bike, corner kid, pawn shop, broadcast, warehouse, frequency, scar keeper) + urbanChunk() worldapi terrain
- S121-01: TRAPX city state API — server/trapxapi package; GET /api/v1/trapx/city-state + POST /api/v1/trapx/events at :7071; Emily Prime Dragon integration
- S120-03: FIELDOFFICE MUD wiring — claim/contest/fo-status/fo-list/k9-deploy/k9-swarm/receipts/attention/integrity/tech-pressure commands wired into apps2/mud/main.go; 1Hz city sim tick
- S120-02: server/beatsync — BeatSync stub (Engine/Tick/Run, 4 beat types in 4/4, Kick/Snare/Bass/Hat, WorldEffect city hooks, sine strength curve, 17 tests) — Apple #3356
- S120-01: server/ledger — Receipt ledger + anti-exploit (append-only, verb types, ByFO/ByActor/ByVerb/Since, ReceiptBurst 30s, flip-score exploit detection, SUSPICIOUS_PATTERN flag, 15 tests) — Apple #3353
- S119-05: server/techpressure — Tech Pressure doom clock (5-tier: LeashFrays/ProcurementWar/QuietAudit/Packmind/CrownProtocol; TierUnlock/DogDeploy/SwarmActivity inputs; decay; BirdCorrection; CROWN_PROTOCOL one-shot; 18 tests) — Apple #3351. S119 ENGINE FOUNDATION COMPLETE.
- S119-04: server/integrity — Control Integrity + Rogue Swarm (per-district CI 0-1, dog decay superlinear, jammer/flip decay, CleanAudit/BirdCorrection recovery, ROGUE_SWARM trigger at 0.15, containment objectives, SCAR_WRITTEN, Registry, 24 tests) — Apple #3349
- S119-03: server/attention — Attention meter (0-1000, superlinear dog gain n^1.3, decay, AUDIT_THRESHOLD→OversightSect, VENDOR_THRESHOLD→ShadowOperator, ecosystem effects, Registry, 18 tests) — Apple #3346
- S119-02: server/k9 — K9 unit + Swarm (Sentry/Escort/Audit modes, 0.85^n diminishing returns, Mark/Latch/HowlBeacon/CustodyLock/ReceiptBurst, Battery drain, BATTERY_LOW/DEAD events, cap=8 per FO, 33 tests) — Apple #3344

- S119-01: server/fieldoffice — FieldOffice state machine (4-phase: Unclaimed/Held/Contested/Containment; Flow/Pressure tick; Flip/Defend/Contest windows; Rogue Swarm containment objectives; 20 tests pass) — Apple #3340

## 2026-06-23
- S118-01: PLD spells (Flash enmity spike, Sentinel/Rampart def buffs, Holy/Banish/Banish II light dmg); PLD job gate
- S117-01: NIN ninjutsu (6 elements × 2 tiers); DEX scaling; NIN job gate; cast <spell> dispatch
- S116-01: DRK dark magic (Drain HP steal, Aspir MP absorb, Absorb-STR/DEX/VIT/INT/MND); DRK/RDM job gate
- S115-01: WHM teleport spells (teleport-meadow/hills/caves/swamp + tele- aliases); 100 MP; combat target reset; arrival broadcast
- S114-01: BRD songs (march/paeon/ballad/minne/carol/mambo); zone-AoE status buffs; BRD job gate; 3-minute duration; party broadcast
- S113-01: BLM black magic nukes (6 elements × 3 tiers); INT scaling; BLM/RDM job gate; target-mob required
- S112-01: Party-targeted spell casting; cast <spell> <player> resolves zone-local target; Cure/Protect/Shell/Haste/Regen/Refresh notify both caster and target
- S111-01: WHM buff spells (Protect/Shell/Haste/Regen/Refresh via status.Stack); Dia DoT on combat target; RDM allowed Refresh
- S110-01: ls-kick/ls-promote MUD commands (guild Officer+); S110-02: shop/shop buy/shop sell at NPC vendors (guildmaster/merchant/scout)
- S109-02: Fame MUD wiring; talk <npc> fame gate; NPC [LOCKED] indicator; quest fame reward tags; 2 new gated NPCs
- S109-01: Add nation fame system (server/fame); Earn/Rank/MeetsRank/Summary; TurnIn now returns RewardFame+FameNation; MUD fame/rep command shows reputation table
- S108: fishing skill (server/gather/fishing.go) + food buff system (server/food/) + MUD wiring (fish/fish-points/eat/food commands)
- S106-01+02: PvP duel system (Manager/Challenge/Accept/ReportHP, 15 tests) + MUD duel/accept/forfeit/leaderboard wiring
- S105-03+04: weather engine (Phase/Engine/ForcePhase, 12 tests) replaces random weather; broadcast + BST tame bonus + prompt indicator
- S105-01+02: cartography Atlas (Visit/Has/ExitMap/10 tests) + MUD explore command + map ✓ indicator
- S104-04: NPC dialogue + quest commands (npcs/talk/quest-accept/quest-turn-in/quests) wired into MUD + kill tracking
- S104-03: server/quest — NPC quest system (Bank/Journal/State/TurnIn), 5 starter quests, 16 tests
- S104-02: BST pet MUD commands (bst/jug-pet/pet-release/pet-status/pet-heel/pet-heal) + pet auto-attack in tickAll
- S104-01: server/pet — BST Beastmaster pet companion system (Tame/JugPet/Tick/Heal/Release), 8 kinds, 20 tests
- S101-01/02/03: bank deposit/withdraw/balance; random weather events (60s/10% per zone, 5 types); survey command with directional player listing
- S100-01/02/03: rest/meditate regen (+5%HP/+3%MP/tick) + stand; target <mob> reticle with hp bar; /p <msg> party chat
- S99-03: bazaar personal shop commands — bazaar set/list/buy; world.bazaars map; gil transfer + item transfer; seller notification
- S99-01/02: mob spellcasting AI (20% hit→debuff from kind pool, 30s) + removedebuffs/echo-drop + cast cure/cure2 (WHM only)
- S98-01/02: IDUNA character persistence — idunaclient.CreateCharacter/UpdateCharacterLevel added; mudCharCache name→charID json store; fetch-or-create on login; save level/xp/pos on disconnect
- S97-01: wire server/job.RecastTracker into MUD — ja/recasts commands; provoke adds enmity CE; benediction restores HP/MP; recast updated on setjob
- S96-01: invisible/sneak aggro block in MUD — cast invisible/sneak commands; EvtMobAggro intercepted when player has active status; 60s expiry with broadcast
- S95-01/02: wire server/worldcrisis into apps2/mud — auto-start on login, phase broadcast, NM kill and mine objective contributions, Chaos Elemental crisis NMs spawn on Emergence, crisis-shard drop
- S94-01: wire server/telecrystal into apps2/mud — crystals/travel/touch commands; validate()/deduct gil/teleport with zone transfer; Dist2D range check for touch activation
- S93-01/02/03: wire server/gear, server/job.CharJob, server/merit into apps2/mud — equipment slots+IL (equip/unequip/gear), sub-job pairing+combined stats (setsubjob/subjob), merit bank with XP→merit conversion at cap (merits/merit-spend)
- S92-03: wire server/guild into apps2/mud — ls-create/ls-invite/ls-leave/ls-info commands; Feather-gated guild chat; GuildID synced to chat Router sessions
- S92-01/02: wire server/enmity and server/chat into apps2/mud — per-mob enmity tables, hate-based aggro retargeting, enmity command; chat Router for say/tell/yell/guild with session sync on zone transfer
- feat(mud): S90-03 auction house — ah browse/sell/buy/history/status/cancel, player gil, itemCategory table
- feat(mud): S90-02 conquest system — declare/conquest commands, kill-based points, 1-min tick, zone-wide broadcast
- feat(mud): S90-01 crafting system — inv/craft/recipes/craft-skills commands, inventory tracking through mine+loot+resolvePool, recipeIngredients table, skill gain
- S89-03: MUD loot pool + NM spawns — solo auto-award, party lot/pass/pool, King Worm + Marsh Leech NMs
- S89-02: MUD job system — 22 FFXI jobs (WAR default), HP/MP per-level scaling, setjob/jobs commands
- S89-01: MUD skillchain — real WS names+resonances, per-mob chain state, 8s window, zone-wide SC announcements
- S88-01: MUD progression wiring — XP+leveling, homepoint, field manuals, party+XPChain, KO/return system
- feat: S76-06 skill XP server-side — PacketSkillXP=16, server-go async IncrementSkill (cap 1.0/action), idunaclient.IncrementSkill
- feat: S76-05 World Crisis phase machine (worldcrisis pkg), PacketWorldCrisisUpdate/ObjectiveComplete, server-go tick goroutine+broadcaster, idunaclient.PatchWorldEvent
- feat: S76-04 crafting endpoint — LookupRecipe, PacketCraftRequest/Result, idunaclient ListItems/CreateItem/DestroyItem, server-go craft handler
- feat: S76-03 telecrystal travel — telecrystal registry (6 crystals), idunaclient pkg, PacketTelecrystalUse/Ack/Err protocol; server-go handler: auth→validate→IDUNA gold deduct→UpdatePosition→Ack
- feat: S76-01 idunaauth package (ES256 JWT, JWKS cache); PacketConnect IDUNA JWT gate; PacketAuthReject=8 wire protocol
- feat: S82-02 sub-job (CharJob/CombinedStats); S82-03 job abilities/recast (RecastTracker); S83-02 merit points (MeritBank); S83-03 item level (Equipment/EffectiveIL); S84-01 crafting guilds (8 types/SuccessChance); S84-02 HQ synthesis (HQTier); 69 tests
- feat: S81-05 Enmity (hate Table, AoE cure, overaggro); S81-06 Death/Raise (HPState, 10% XP penalty); S82-01 22 FFXI jobs (StatsFor/HPAtLevel/MPAtLevel); S83-01 Level XP (L99 cap, CharXP.AddXP, level^1.8); 60 tests
- feat: S86-02 Home Point (SetHome/ReturnHome, 8% XP penalty); S86-03 Field Manuals (ApplyBonus, ApplyAll stacking, expiry); S87-03 NM Aggro types (sight cone, sound radius, job detect, Sneak/Invisible blocking); 40 tests
- feat: S86-01 Conquest system (Region/Map, 3 nations, incumbent tie-break, weekly Tick); S87-01 NM spawn conditions (placeholder kill, time window, chance roll); S87-02 Treasure pool (Lot/Pass/Resolve, highest roll wins, 48 tests)
- feat: S85-01/02/03 party system — Party/Alliance/XPChain, 6-player cap, leader transfer, XP split, kill chain +10%/kill cap 50%, 37 tests
- feat: DragonsNShit MUD server (apps2/mud) — playable text MUD on :2323, all server packages wired, 1Hz game loop
- feat: S84-04 mining skill (server/gather) — FFXI-parity MiningPoint, loot table, HQ rolls, Meadow+Swamp presets, 27 tests

- feat: Swampville secondary starting zone — zone 3, scene 3 swamp terrain (clay/mud/water/mangrove), leech+slime+lizard mobs, 20 tests

## 2026-06-21
- feat: S81-04 status effects system (Poison/Paralyze/Slow/Silence/Bind/Haste/Regen/Refresh/Protect/Shell); 43 tests
- feat: S81-03 TP weapon skill points system; 34 tests (Apple #2530)
- feat: S81-01+02 skillchain+magic burst system; 14 resonances 3 tiers 31 tests (Apple #2521)
- feat: S80-01 auto-attack + mob tagging; AI state machine; 26 tests (Apple #2518)
- feat: S79-01 linkshell guild system; Feather/Feather Sack; 22 tests (Apple #2515)
- feat: S78-01 chat system say/tell/yell/guild (PacketChat=6); 19 tests (Apple #2472)

- docs: S77-01 DragonsNShit MMO_NORTHSTAR — 7 systems, IDUNA schema, 8-milestone product roadmap (Apple #2470)

## 2026-06-20

- feat: S42-01 worldapi :7070 live; S42-02 scene-differentiated ProceduralWorldStore (Apple #1449)

## 2026-06-18
- feat(worldapi): S41-02 DragonflyChunkGenerator — WorldStore hook + procedural fallback + block name→ID mapping (Apple #1421)

- feat(worldapi): S40-02 server/worldapi package — /chunks endpoint + ChunkGenerator interface (Apple #1413)


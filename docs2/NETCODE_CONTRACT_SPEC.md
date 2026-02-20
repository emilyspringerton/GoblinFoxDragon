# SHANKPIT Netcode Contract Spec (Definitive)

## 0) Primary Rule

### Server authoritative over
- Connection state (`active/inactive`)
- `client_id` assignment
- Spawn state (`spawn_id`, position, yaw/pitch, alive/dead)
- World simulation (movement, collisions, hits, health)
- Snapshot sequencing

### Client authoritative over
- Input sampling
- Local prediction of the local player only
- Rendering and interpolation
- UI

Prediction is allowed for responsiveness but must reconcile to server truth.

---

## 1) Identity, Lifecycle, and Client Count

### 1.1 Connection state machines

Client states:
1. `DISCONNECTED`
2. `CONNECTING`
3. `CONNECTED_UNASSIGNED`
4. `CONNECTED_ACTIVE`

Server stores `clients[MAX_CLIENTS]` fields:
- `active`
- `addr`
- `client_id`
- `last_heard_ms`
- `last_acked_cmd_seq`
- `spawn_id`
- `player_state`

### 1.2 `WELCOME` is only source of `client_id`
- Server assigns and sends `WELCOME {client_id, server_time_ms, protocol_version}`.
- Client never guesses/invents `client_id`.
- Client ignores snapshots before `WELCOME`.

### 1.3 Client count definition
- Server prints `clients: N` where `N = count(clients[i].active == true)`.
- `active=true` only after:
  - server sent `WELCOME`, and
  - server received at least one valid `CMD` from that address.
- Set `active=false` on:
  - explicit `DISCONNECT`, or
  - timeout (`CLIENT_TIMEOUT_MS`, e.g. 5000ms).

### 1.4 Anti-ghost rule
- Snapshot includes only `clients[i].active == true` players.
- On inactive transition, clear slot to neutral state and bump `client_epoch` (if present).

Acceptance:
- Repeated join/leave never creates phantom players.
- Server client count reflects active slots.

---

## 2) Packets and Sequencing

### 2.1 Packet minimum set
- `C->S HELLO {protocol_version, nonce}`
- `S->C WELCOME {client_id, server_time_ms, nonce_echo}`
- `C->S CMD {client_id, cmd_seq, ack_snapshot_seq, client_time_ms, input_axes, buttons, yaw, pitch}`
- `S->C SNAPSHOT {snapshot_seq, server_time_ms, ack_cmd_seq[for that client], players[]}`

Optional:
- `C->S DISCONNECT {client_id}`
- Ping/pong RTT packets

### 2.2 Sequencing invariants
- `cmd_seq` strictly increasing per client.
- `snapshot_seq` strictly increasing per server stream.
- Client drops stale/duplicate snapshots (`<= last_snapshot_seq`).
- Server stores `last_acked_cmd_seq` per client for diagnostics.

---

## 3) Snapshot Data and Write Policy

### 3.1 Per-player snapshot fields (required)
- `active`
- `client_id`
- `spawn_id`
- `alive`
- `pos`, `vel`
- `yaw`, `pitch`
- `health`
- `fire_seq`
- any additional game state required for rendering/gameplay

### 3.2 Client write policy
Maintain two state layers:

A) Authoritative remote state
- Updated from snapshot/interpolation only.
- Never locally predicted.

B) Predicted local state
- Advanced with local commands every frame.
- Corrected via reconciliation on snapshot receipt.

---

## 4) Prediction and Reconciliation

### 4.1 Client command generation
Each frame:
1. Sample input.
2. Update camera yaw/pitch from look input.
3. Build `cmd` from camera + buttons/axes.
4. Push to `cmd_history[cmd_seq]`.
5. Predict local movement immediately.
6. Send at fixed rate (e.g., 60 Hz).

### 4.2 Server simulation
- Apply ordered commands per client.
- Reorder within a small window if needed; drop old commands.
- Simulate at fixed tick.
- Include per-client `ack_cmd_seq` in snapshots.

### 4.3 Client reconcile
On valid snapshot:
1. Find local authoritative state by `client_id`.
2. Read `ack_cmd_seq`.
3. Rewind local predicted state to authoritative baseline.
4. Re-simulate commands `ack_cmd_seq+1 ... latest`.
5. If error exceeds hard threshold, hard snap.
6. Otherwise apply render-only smoothing offsets.

Acceptance:
- High RTT remains responsive locally and converges without oscillation.

---

## 5) Spawn/Respawn and Camera Correctness

### 5.1 `spawn_id` gate
Server increments `spawn_id` on respawn/teleport.

Client behavior when local `spawn_id` changes:
- Seed camera yaw/pitch from snapshot.
- Clear reconcile offsets.
- Reset command history baseline (`base_cmd_seq = ack_cmd_seq`).
- Force next 1–2 commands to seeded yaw/pitch.

Acceptance:
- Repeated respawns always align camera with authoritative facing.

---

## 6) Coordinate System (Single Locked Convention)

Convention:
- `+X` right, `+Y` up.
- `yaw=0` faces `-Z`.
- `forward = (sin(yaw), 0, -cos(yaw))`
- `right   = (cos(yaw), 0,  sin(yaw))`

Hard rule:
- Movement, camera, projectile/raycast aim, and server simulation all use the same basis.
- Model-facing adjustments are render-only.

Acceptance:
- `W` at yaw=0 moves toward `-Z` on client and server.
- Shooting at yaw=0 fires toward `-Z`.
- `A/D` are consistent relative to current view.

---

## 7) Weapon Animation in Net Play

Treat firing as events, not a latched state.

- Server increments `fire_seq` once per accepted shot.
- Snapshots include `fire_seq`.
- Client triggers recoil/muzzle animation only when `fire_seq` changes.

Acceptance:
- Exactly one recoil trigger per accepted shot, independent of snapshot rate.

---

## 8) Required Debug Instrumentation

At least 1 Hz diagnostics:

Server:
- `active_clients`
- per-slot `active`, `addr`, `last_heard_ms`

Client:
- `client_id`
- `last_snapshot_seq`
- `ack_cmd_seq`
- `cmd_seq`
- `spawn_id`
- active snapshot player count (`ghost detection`)

Net quality:
- snapshot dt avg/stddev
- reconcile events per second

Acceptance:
- If client count shows 0 unexpectedly, logs indicate whether cause is no `WELCOME`, no valid `CMD`, or timeout.

---

## Implementation Guardrails

- Minimal churn: only touch code directly needed to satisfy this contract.
- No behavior outside this spec.
- Prefer explicit invariants/assertions in code comments near packet handling.
- Any intentional deviation must be documented in commit message and this file.

---

## Repeatable Prompt Template

```text
You are fixing SHANKPIT netcode. Do not introduce new behaviors beyond this spec.

Implement docs2/NETCODE_CONTRACT_SPEC.md exactly:
- WELCOME assigns client_id; client never guesses id; client ignores snapshots until WELCOME.
- Server clients[i].active toggles correctly; server prints clients count = number of active slots.
- Server snapshots include only active clients; inactive slots never appear (fix ghost players).
- Add spawn_id per player; client seeds camera yaw/pitch on spawn_id change and clears reconcile/cmd history baseline.
- Lock coordinate convention: yaw=0 faces -Z; forward=(sin(yaw),0,-cos(yaw)), right=(cos(yaw),0,sin(yaw)).
  Apply convention in movement, camera, projectiles, and server sim. Model-facing correction is render-only.
- Add fire_seq per player; increment on server per accepted shot; client triggers weapon animations when fire_seq changes.
- Implement prediction + reconcile: rewind to snapshot state at ack_cmd_seq, re-simulate cmds, use hard snap threshold else render-only smoothing offset.
- Add debug overlay/log 1/sec with active_clients, client_id, snapshot_seq, ack_cmd_seq, cmd_seq, spawn_id, ghost count.

Provide minimal diff and map each change to a section in docs2/NETCODE_CONTRACT_SPEC.md.
```

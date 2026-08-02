# Headless MUD Sessions — Northstar

*Written 2026-08-02. Founder, real-time, verbatim: "how can we keep battlegrounds as is and have
a separate scene for our game world - to start the second scene can be the same as the first just
no match resources... we can iterate on that while unifying the experience with the affordances
of the mud" → "once we add the chat interface we can have the mud commands work in the chat
window and then all of a sudden we have a mmo that isnt just a mud anymore" → chose "go straight
for the session model" over a smaller read-only-first slice.*

---

## 1. The real gap, found while scoping this

`apps2/mud`'s entire command system (`cmdLook`, `cmdInventory`, `cmdMove`, `cmdAttack`, every
job/craft/party/guild command — `handle(p, line)`'s own real dispatch table) assumes a live,
connected telnet session: a `player` struct holding a real `net.Conn`, registered in
`gw.players[slot]`, with real in-memory position/HP/zone/inventory state. A player sitting in a
REDGARDEN Battlegrounds match has no such session running — Battlegrounds is a separate UDP
process apps2/mud knows nothing about. So both of the founder's own framings —
"MUD commands in the chat window" and "a second, walkable world scene" — are the same missing
piece from two directions: a character needs to be able to have real MUD state and take real MUD
actions **without an open telnet connection**.

## 2. The mechanism

`player.w` is a `*bufio.Writer`. Every existing `send`/`sendf`/`prompt` call writes through it.
`bufio.NewWriter` wraps *any* `io.Writer` — a real `net.Conn` for telnet, or an in-memory
`bytes.Buffer` for a headless session. This means **no command-processing code changes at all**:
`cmdLook`, `cmdInventory`, `handle()`'s own dispatch table, all of it works completely unchanged
against a `player` built around a buffer instead of a socket. The only new code is (a)
constructing that `player` from a persisted IDUNA character instead of a fresh telnet login, and
(b) capturing/returning what got written to the buffer instead of flushing it to a socket.

```
Real telnet session:    player{ conn: <net.Conn>, w: bufio.NewWriter(conn) }       — existing
Headless session:       player{ conn: nil,        w: bufio.NewWriter(&buf) }       — new
                         both run through the exact same handle(p, line) dispatch
```

## 3. Architecture

### 3.1 Headless player registry

A new map, `gw.headlessPlayers map[string]*headlessSession` (`characterID` → session), separate
from `gw.players` (real telnet connections, keyed by `slot`/remote address) so the two can never
collide or be confused for each other. A `headlessSession` wraps a `*player` plus its own
`*bytes.Buffer` (for reading back output) and a last-active timestamp (for idle eviction).

### 3.2 Lifecycle

- **First command from a character with no headless session and no live telnet session**: load
  the real IDUNA `Character` record (level, XP, gold, scene, position, job — all real columns
  already on the `characters` table, `MMO_NORTHSTAR.md` §3), construct a `player` matching
  `handleConn`'s own real default-then-overlay shape (§4785 in `apps2/mud/main.go`), register it
  in `gw.headlessPlayers`, run the command via `handle(p, line)`, capture the buffer.
- **Subsequent commands**: reuse the existing headless session (don't reconstruct/reload every
  time — a character's mid-session state, like an open shop transaction or combat target, has to
  persist across chat messages the same way it would across telnet keystrokes).
- **A real telnet connection for the same character appears**: the headless session must be torn
  down and its state flushed to IDUNA first (same persist path `handleConn`'s own disconnect
  `defer` already uses) — never two live `player` structs for one character at once. Real,
  necessary conflict handling, not an edge case to skip.
- **Idle eviction**: a headless session with no activity for some window (proposal: 10 minutes)
  persists its state to IDUNA and is dropped, same shape as a telnet disconnect. Prevents an
  unbounded memory leak from players who queue into Battlegrounds and never chat again.
- **Zone broadcast visibility (open question, not resolved here)**: should a headless session
  registering in `gw.zoneMgr`/`gw.chatRouter` make it visible to real telnet players in that zone
  ("X has entered the world", `say` audible to them, etc.)? Milestone 1 below deliberately does
  **not** register a headless session in either — it can run commands and get real output, but is
  invisible to the rest of the live world. Right for a first slice (no risk of confusing/spamming
  real players with phantom arrivals); a real design question for later once this is proven.

### 3.3 Command routing from the Battlegrounds chat box

The existing chat input (`apps2/battlegrounds_gui`'s Enter-to-chat line, 2026-08-02) currently
always posts to IDUNA's `/api/v1/chat/messages` as channel `battlegrounds`. A line starting with
`/` is a MUD command instead: the client POSTs it to a new endpoint
(`POST /api/v1/mud/command`, IDUNA-relayed the same way chat is, since the Battlegrounds client
has no direct connection to apps2/mud's own telnet port) with the raw command text; apps2/mud
polls for pending commands the same way it will eventually need to poll for
Battlegrounds-originated chat (a real, already-named gap from the chat feature's own CHANGELOG
entry), runs it through a headless session, and the response comes back through the same
poll/relay path chat already uses. Not built in Milestone 1 (see below) — Milestone 1 proves the
headless-session mechanism itself, in isolation, before wiring the full round-trip.

### 3.4 The second scene

Once headless sessions are real, "a separate scene for our game world" stops needing new
server-side design — it's a REDGARDEN-side rendering-only feature: a new client mode reusing the
existing local (non-networked) single-hero render path already in `apps2/battlegrounds_gui`
(`arena_init()`, no matchmaker, no draft, no item shop/node-capture UI — "no match resources," the
founder's own framing) with the player's real position pulled from their headless session's own
IDUNA-synced state, WASD movement writing position updates back the same way, and the chat/command
box as the interaction surface for everything not yet rendered (which, at first, is nearly
everything — inventory, combat, NPCs). A real, separate, later piece of work — not scoped further
here.

## 4. What this is not

Not a replacement for real telnet play — telnet stays first-class forever (same founder direction
`REDGARDEN_GUI_NORTHSTAR.md` §3 already recorded: "cli will continue to work"). Not full parity
with a live telnet session on day one — combat, party, and anything requiring real-time
tick-by-tick presence (an NM spawning near you, a party member's HP changing live) doesn't make
sense for a headless, poll-driven session and isn't attempted here. Not a merge of Battlegrounds'
own real-time UDP loop with apps2/mud's 1Hz loop — Battlegrounds stays exactly as it is
(`REDGARDEN_GUI_NORTHSTAR.md` §4.1's own "process/loop separation stays" already settled this).

## 5. Milestones

| # | Milestone | Acceptance | Status |
|---|---|---|---|
| 0 | This northstar | Written, registered in golden-docs-index | DONE |
| 1 | Headless session mechanism proven | A real headless `player` can be constructed from a real IDUNA character and run one real command (`look`) via the real `handle(p, line)` dispatch, output captured correctly — proven with a direct Go test, not yet wired to chat/HTTP | |
| 2 | `/api/v1/mud/command` relay | New IDUNA-relayed endpoint (same shape as chat), apps2/mud polls and dispatches to a headless session, response relayed back | NOT STARTED |
| 3 | Battlegrounds chat box routes `/`-prefixed lines to it | Real end-to-end: type `/look` in a Battlegrounds match, see apps2/mud's real room description in the chat log | NOT STARTED |
| 4 | Idle eviction + telnet-conflict handling | Headless sessions persist and drop cleanly; a real telnet login for the same character correctly tears down and flushes any live headless session first | NOT STARTED |
| 5 | Second scene (client-side) | New REDGARDEN-fork client mode: local render, no match resources, position sourced from a headless session | NOT STARTED |

## 6. Related docs

- `docs2/REDGARDEN_GUI_NORTHSTAR.md` — Battlegrounds itself; this doc is additive, doesn't amend it
- `docs2/MMO_NORTHSTAR.md` — IDUNA character schema this design reads/writes
- `docs2/DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md` — the apps2/mud vs apps2/server-go split; this design
  targets apps2/mud specifically (where the real command depth lives today)

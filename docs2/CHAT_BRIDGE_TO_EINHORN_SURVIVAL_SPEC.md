# GFD ↔ EINHORN_SURVIVAL Cross-Server Chat Bridge — Spec

**Status:** Scoping pass, no code yet — see `EMILY/BACKLOG.md` S171-04.
**Founder:** "can we dev cross server chat? GFD to paper?"

---

## What exists today (checked, not assumed)

**GFD side** (`GoblinFoxDragon/apps2/server-go`): a real UDP server with a pure, no-I/O chat
router (`server/chat/chat.go`) — four channels: `ChatSay` (radius-based, in-scene), `ChatTell`
(1:1 by name), `ChatYell` (scene-wide broadcast), `ChatGuild`. Wired at `apps2/server-go/
main.go:800` — `PacketChat` is parsed, routed via `chatRouter.Deliver(...)`, and the resulting
per-recipient `Delivery` list is sent by looking up each recipient's live UDP address in
`clientAddrs`, **a function-local `map[string]*net.UDPAddr` declared at `main.go:274`** — not a
package-level variable, not accessible from outside `main()`'s own scope. This matters: a bridge
that needs to inject an unprompted message to "everyone currently connected" from a separate
poller can't just call into the existing send path as-is. It needs either (a) `clientAddrs` lifted
to a package-level, mutex-guarded map, or (b) the bridge poll folded into the same goroutine that
already owns `clientAddrs`, checked on a timer inside the existing packet-read loop rather than as
a truly separate goroutine. No broadcast-to-all helper exists yet either way — flagged here so a
future build doesn't assume one.

**EINHORN_SURVIVAL/GTA7 side**: real Paper server, real chat via Bukkit's async chat event
(`io.papermc.paper.event.player.AsyncChatEvent` on this Paper version — confirmed via the same
`javap` discipline used for GTA7's other real-API checks before assuming the class/package). GTA7
already has a working pattern for exactly this shape of problem: `IdunaClient` (direct HTTP,
cached JWT, async off the main thread) already authenticates as a real IDUNA agent and posts/reads
structured records. The chat bridge's EINHORN_SURVIVAL side is almost entirely "reuse
`IdunaClient`'s pattern," not new infrastructure.

**IDUNA**: already the coordination hub for everything else in this monorepo (Apples, blog, the
TYLER reading room, WOTAN player identity) — the founder's own original framing ("IDUNA is the
natural shared coordination point... rather than inventing a new channel") holds up against what's
actually there. No chat-relay endpoint exists yet.

---

## Proposed design

### IDUNA: new `internal/chatbridge` package

Same "own small SQLite file, thin HTTP handler" shape as `internal/blog` and `internal/tyler` —
not a queue/pubsub system, a simple append-and-poll store, since message volume here is low
(player chat, not a firehose) and both sides are already comfortable polling (GTA7's own Enforcement/
Watcher/Rogue Swarm ticks are all poll-based, not event-push).

```go
type Message struct {
    ID          int64
    FromServer  string // "gfd" | "einhorn_survival"
    FromName    string // real in-game display name, not an IDUNA player_id -- either
                        // side's chat is keyed by display name, not by a shared account yet
    Body        string
    SentAt      time.Time
}
```

Endpoints (mirrors blog/tyler's own shape exactly):
- `POST /api/v1/chat-bridge/messages` — publish. Requires a new `chatbridge.write` permission
  (same migration-based grant pattern as `blog.write`/`tyler.write`/`apples.write`), granted to
  two new real agents — `GFD-CHAT-BRIDGE` and `GTA7-SERVER` (GTA7 already exists and already has a
  real agent; reuse it rather than minting a third).
- `GET /api/v1/chat-bridge/messages?since_id=<id>&exclude_server=<server>` — poll for new
  messages, excluding ones the caller itself just sent (so GFD doesn't echo its own message back
  to itself, same idea EINHORN_SURVIVAL needs the mirror of). Auth-gated, not public — this is
  live player chat, not something to expose unauthenticated the way blog/tyler reads are.

### GFD side

- On `ChatYell` specifically (scene-wide is the closest existing channel to "public enough to
  bridge," `ChatSay`'s radius-gating and `ChatTell`'s 1:1 nature don't fit a cross-server relay)
  at `main.go:800`'s existing handler: after `chatRouter.Deliver(...)`, also POST to IDUNA
  (async — a goroutine with its own short-timeout HTTP client, not blocking the packet loop).
- A polling goroutine (needs `clientAddrs` addressed per the "What exists today" note above)
  that GETs new EINHORN_SURVIVAL-origin messages every few seconds and sends a synthesized chat
  packet to every connected client, prefixed to make the cross-server origin obvious (e.g.
  `[Paper] <FounderName>: message text`) — never presented as if it came from a GFD player.

### EINHORN_SURVIVAL/GTA7 side

- New `ChatBridgeListener` (mirrors `PlayerIdentityListener`'s existing shape): on
  `AsyncChatEvent`, POST to IDUNA async via a small wrapper around the existing `IdunaClient`.
- A repeating `Bukkit.getScheduler().runTaskTimer` poll (same pattern as `EnforcementManager`/
  `RogueSwarmManager`'s own ticks) that GETs new GFD-origin messages and calls
  `Bukkit.broadcastMessage`, prefixed `[DragonsNShit] <Name>: message text`.

---

## Rate limiting / spam handling (explicitly flagged as unscoped by the original backlog note)

Not designed in detail here — this is the one piece of the original ask ("rate limiting/spam
handling") this pass is deliberately leaving open rather than guessing at numbers. Real options,
for whoever picks this up next:
- Simplest: IDUNA's own `chatbridge.write` POST endpoint rate-limits per-agent (not per-player —
  neither side authenticates individual players to IDUNA for chat, only the server-level agent),
  which caps total bridge throughput but can't stop one loud player from consuming that whole
  budget.
- More correct, more work: track per-`FromName` message counts server-side (in GFD's/GTA7's own
  chat-hook code, before it ever reaches IDUNA) and drop/throttle at the source. This needs a real
  per-player rate state on each side, which neither side currently has for chat.
- Punt entirely for a v0: low real player counts on both servers today make abuse unlikely in the
  short term, and this can be revisited once there's an actual incident to design around instead
  of a hypothetical one — consistent with this session's `feedback-*` memory guidance to build
  what's needed, not what might hypothetically be needed.

## Phased plan

1. **IDUNA**: `internal/chatbridge` package + endpoints + migration (permission + two agent
   grants). Verify end-to-end via `curl`, same discipline as every other IDUNA feature this
   session (GTA7-SERVER, tyler.write) — before either game server touches it.
2. **EINHORN_SURVIVAL/GTA7**: `ChatBridgeListener` + poll task. Ship first, alone — it's the
   smaller lift (no `clientAddrs` restructuring needed) and gives something real to test the IDUNA
   side against before touching GFD's live server.
3. **GFD**: `clientAddrs` lifted to package scope (or the poll folded into the existing loop),
   then the publish hook on `ChatYell` and the receive/broadcast poll. Higher-risk change (a live,
   already-running UDP server), do this last and test locally before touching the live process.
4. **Rate limiting**, once real usage patterns exist to design against, not before.

## Open questions

- Should `ChatTell`/`ChatGuild` ever bridge, or is `ChatYell`-only (and EINHORN_SURVIVAL's
  equivalent, all public chat) the permanent scope? Leaning toward "public-only, permanently" —
  a cross-server DM relay is a much bigger trust/abuse surface for very little value.
- Character identity: bridged messages are display-name-only, no shared account linkage. Given
  WOTAN already links GTA7 players to real IDUNA `player_id`s (S170-239-adjacent work), a future
  pass could show a verified badge for players whose name is provably the same person on both
  sides — not attempted here.

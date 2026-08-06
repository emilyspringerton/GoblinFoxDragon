# GFD ↔ EINHORN_SURVIVAL Cross-Server Chat Bridge — Spec

**Status:** IDUNA side done (extended the existing `/api/v1/chat/messages` endpoint, not a new
one — see the correction in "What exists today" below). EINHORN_SURVIVAL/GTA7 and GFD sides not
yet built. See `EMILY/BACKLOG.md` S171-04.
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
actually there. **Correction, found after the first draft of this doc:** a chat-relay endpoint
already exists — `POST/GET /api/v1/chat/messages` (migration `202608020001_chat_messages.sql`,
`internal/http/handlers/chat_messages.go`), built for a real, separate bridge (GoblinFoxDragon's
own `apps2/mud` telnet chat <-> REDGARDEN's Battlegrounds GUI client). Same shape as what this doc
originally proposed building from scratch: any authenticated caller (any valid IDUNA JWT, no
extra permission gate) may `POST`/`GET`, `sender_source`/`channel` are plain validated strings, no
identity linkage to players. **The original draft of this doc proposed a parallel
`internal/chatbridge` package before this was found — don't build that; extend this one instead**,
same "check for an existing system before building a parallel one" discipline already applied
elsewhere in this session (shankpit-460's bot AI).

### IDUNA: extend the existing endpoint, not a new package

`internal/http/handlers/chat_messages.go`'s `validChatSources` map gets two new entries,
`gfd_server` and `einhorn_survival`, alongside the existing `mud`/`battlegrounds`. A new
`validChatChannels` entry, `gta7`, is added for EINHORN_SURVIVAL-origin messages (paralleling how
`battlegrounds` is both a source and its own channel already). GFD-origin messages use the
already-real `yell` channel value — GFD's own zone-wide broadcast, the one channel this bridge
relays (see Design Decisions below for why not `say`/`tell`/`guild`).

**No new IDUNA endpoint, no new migration, no new permission, no new agent grant needed.**
`GTA7-SERVER` (already exists, already has a real secret) can call `/api/v1/chat/messages` today
with zero additional IDUNA-side provisioning — `RequireAuth` is the only gate on this route
(confirmed in `main.go`), same as it already was for the mud/battlegrounds bridge. GFD's own
future outgoing/incoming chat code will need its own real IDUNA agent (none of GFD's existing
processes currently authenticate to IDUNA as a general-purpose agent the way GTA7 does) —
provisioning that agent is real remaining work, just not an IDUNA schema/endpoint change.

The original message shape this doc first proposed (kept here for reference, but the real schema
above supersedes it):

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

**Real endpoints, already live** (superseding the `chat-bridge`-prefixed ones above):
- `POST /api/v1/chat/messages` — `{"channel": "yell"|"gta7", "sender_name": "...",
  "sender_source": "gfd_server"|"einhorn_survival", "body": "..."}` -> `{"id": N}`.
- `GET /api/v1/chat/messages?since_id=<id>&limit=<n>` — no `exclude_server` filter exists on this
  endpoint (unlike the original proposal) — callers filter client-side by checking
  `sender_source` against their own, same amount of code either way, no IDUNA change needed for
  it.

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
- Simplest: rate-limit `/api/v1/chat/messages` POSTs per-agent (not per-player — neither side
  authenticates individual players to IDUNA for chat, only the server-level agent), which caps
  total bridge throughput but can't stop one loud player from consuming that whole budget. This
  endpoint doesn't currently rate-limit at all (confirmed: no `middleware.AuthRateLimit` wrapping
  it in `main.go`, unlike some other routes) — adding one would affect the existing mud/
  battlegrounds bridge too, not just this one, so scope that change carefully if it's picked up.
- More correct, more work: track per-`FromName` message counts server-side (in GFD's/GTA7's own
  chat-hook code, before it ever reaches IDUNA) and drop/throttle at the source. This needs a real
  per-player rate state on each side, which neither side currently has for chat.
- Punt entirely for a v0: low real player counts on both servers today make abuse unlikely in the
  short term, and this can be revisited once there's an actual incident to design around instead
  of a hypothetical one — consistent with this session's `feedback-*` memory guidance to build
  what's needed, not what might hypothetically be needed.

## Phased plan

1. ~~**IDUNA**: `internal/chatbridge` package + endpoints + migration~~ — **done differently than
   planned**: `internal/http/handlers/chat_messages.go`'s `validChatSources`/`validChatChannels`
   extended (`gfd_server`/`einhorn_survival` sources, `gta7` channel), reusing the existing
   `/api/v1/chat/messages` endpoint instead of building a parallel one. No migration, no new
   permission, no new agent grant — `GTA7-SERVER`'s existing credential already works against this
   route today. IDUNA commit (see `CHANGELOG.md`).
2. **EINHORN_SURVIVAL/GTA7**: `ChatBridgeListener` + poll task. Ship first, alone — it's the
   smaller lift (no `clientAddrs` restructuring needed) and gives something real to test against
   before touching GFD's live server.
3. **GFD**: `clientAddrs` lifted to package scope (or the poll folded into the existing loop),
   then the publish hook on `ChatYell` and the receive/broadcast poll. Higher-risk change (a live,
   already-running UDP server), do this last and test locally before touching the live process.
   Also needs a real IDUNA agent minted for GFD (none of GFD's processes currently authenticate to
   IDUNA as a general-purpose agent) — a real prerequisite this phase can't skip.
4. **Rate limiting**, once real usage patterns exist to design against, not before.

## Open questions

- Should `ChatTell`/`ChatGuild` ever bridge, or is `ChatYell`-only (and EINHORN_SURVIVAL's
  equivalent, all public chat) the permanent scope? Leaning toward "public-only, permanently" —
  a cross-server DM relay is a much bigger trust/abuse surface for very little value.
- Character identity: bridged messages are display-name-only, no shared account linkage. Given
  WOTAN already links GTA7 players to real IDUNA `player_id`s (S170-239-adjacent work), a future
  pass could show a verified badge for players whose name is provably the same person on both
  sides — not attempted here.

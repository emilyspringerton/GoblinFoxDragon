# GFD ↔ EINHORN_SURVIVAL Cross-Server Chat Bridge — Spec

**Status:** DONE — all three sides live. IDUNA extended `/api/v1/chat/messages` (not a new
endpoint). EINHORN_SURVIVAL/GTA7 posts real player chat and relays GFD-origin messages into
Minecraft. GFD's `apps2/server-go` posts real `ChatYell` chat and relays EINHORN_SURVIVAL-origin
messages via the existing `broadcastCh` mechanism (no `clientAddrs` restructuring needed after
all — see the correction below). Real player-facing verification (does a message typed on one
side actually show up on the other, seen by an actual person) still hasn't happened — both
directions are verified only as far as "posts/polls succeed, no crashes, no errors logged."
**Founder:** "can we dev cross server chat? GFD to paper?" → "continue" (×3, across separate
turns, each time picking this up as the next scoped-but-unfinished item).

---

## What exists today (checked, not assumed)

**GFD side** (`GoblinFoxDragon/apps2/server-go`): a real UDP server with a pure, no-I/O chat
router (`server/chat/chat.go`) — four channels: `ChatSay` (radius-based, in-scene), `ChatTell`
(1:1 by name), `ChatYell` (scene-wide broadcast), `ChatGuild`. Wired at `apps2/server-go/
main.go:800` — `PacketChat` is parsed, routed via `chatRouter.Deliver(...)`, and the resulting
per-recipient `Delivery` list is sent by looking up each recipient's live UDP address in
`clientAddrs`, a function-local `map[string]*net.UDPAddr` declared at `main.go:274` — not a
package-level variable. **Correction, found during implementation:** this doesn't actually block a
bridge poller, because a second mechanism already exists and was missed on the first read —
`broadcastCh`, a `chan []byte` (declared right next to `clientAddrs`) already feeding a dedicated
broadcast goroutine that iterates `clientAddrs` and sends to every connected client. The World
Crisis ticker already uses exactly this pattern (a `time.Ticker` goroutine started inside `main()`
that periodically sends a packet into `broadcastCh`). The chat bridge's own receive-side poller
just does the same thing — no `clientAddrs` restructuring needed after all. **Real, pre-existing
issue found and left unfixed, flagged rather than silently touched:** the broadcast goroutine
iterates `clientAddrs` (a plain Go map) with no lock, while the main loop writes to that same map
elsewhere — a genuine data race (Go maps aren't safe for concurrent read/write), pre-existing and
unrelated to this feature. Out of scope to fix here; noted so it isn't mistaken for something this
change introduced.

Also found live during implementation, not part of the original scoping: `server/idunaclient`
(the package `apps2/server-go` already imports and instantiates as `idunaClient := idunaclient.New()`
at `main.go:268`) already has real, working `PostChatMessage`/`GetChatMessages` methods — built for
`apps2/mud`'s side of the *other* real bridge this endpoint already serves (mud↔Battlegrounds).
And the live `gfd-server-go.service` systemd unit already runs with real `IDUNA_AGENT_NAME`/
`IDUNA_AGENT_SECRET` (`DRAGONSNSHIT-MUD`, confirmed via the deployed env file) — meaning **GFD
already had everything needed to talk to IDUNA's chat relay before this feature started**, no new
agent, no new credential. The original spec below (written before either of these two discoveries)
proposed more new infrastructure than turned out to be necessary — kept for the record, not
because it's still accurate.

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

### GFD side — DONE (`apps2/server-go`)

- `server/chat/chat.go`'s `encodeChat` exported to `EncodeChat` (trivial, pure function, no
  behavior change) so the bridge poller can build outbound chat packets outside the `chat`
  package.
- `server/idunaclient/idunaclient.go`: new `PostChatMessageAs(channel, senderName, senderSource,
  body)`, with the existing `PostChatMessage` (used by `apps2/mud`) becoming a thin wrapper that
  calls it with `"mud"` — the existing call site is untouched.
- `main.go`'s existing `PacketChat` handler: on `ChatYell` specifically, after
  `chatRouter.Deliver(...)`, also fires `go idunaClient.PostChatMessageAs("yell", name,
  "gfd_server", body)` — a bare goroutine, not blocking the packet-read loop, logs on failure
  only.
- A new `time.Ticker`-driven goroutine started inside `main()` (same shape as the existing World
  Crisis ticker): polls `idunaClient.GetChatMessages` every 5s, filters for
  `sender_source == "einhorn_survival"` client-side (no `exclude_server` param on the real
  endpoint), encodes via `chat.EncodeChat(chat.ChatYell, "[Paper] "+name, body)`, sends into the
  existing `broadcastCh`. First tick only records the current high-water mark, doesn't broadcast
  — a restart doesn't replay EINHORN_SURVIVAL's chat history into the game.

### EINHORN_SURVIVAL/GTA7 side — DONE

- `ChatBridgeListener` (mirrors `PlayerIdentityListener`'s existing shape): on the real
  `AsyncChatEvent`, converts the message `Component` to plain text via
  `PlainTextComponentSerializer` and POSTs to IDUNA via `IdunaClient.postChat` — safe to call the
  blocking HTTP client directly here since `AsyncChatEvent` already fires off the main thread.
- `ChatBridgePoller`: an async-scheduled repeating task (same pattern as `EnforcementManager`/
  `RogueSwarmManager`'s own ticks) that polls every 5s, hops back to the main thread via
  `Bukkit.getScheduler().runTask(...)` to call `Bukkit.broadcastMessage`, prefixed
  `[DragonsNShit] <Name>: message text`. Same high-water-mark-on-first-tick behavior as the GFD
  side, for the same reason.

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
2. ~~**EINHORN_SURVIVAL/GTA7**: `ChatBridgeListener` + poll task~~ — **done.** Built, deployed,
   confirmed enabling cleanly with zero exceptions.
3. ~~**GFD**: `clientAddrs` lifted to package scope~~ — **done differently than planned**: turned
   out unnecessary once `broadcastCh` was found (see the correction in "What exists today").
   `EncodeChat` export, `PostChatMessageAs`, the `ChatYell` publish hook, and the receive/broadcast
   poller are all built and deployed. **No new IDUNA agent was needed either** — `apps2/server-go`
   was already running with a real, live `DRAGONSNSHIT-MUD` credential. **Real deploy incident,
   found and fixed, not GFD-code-related**: the live `server-go` process turned out to be an
   orphan (`PPID=1`, started manually at some point in the past, never actually supervised by the
   `gfd-server-go.service` systemd unit) — holding the UDP port and causing the real systemd unit
   to crash-loop on restart. Confirmed it was fatbaby's own process (not root's, despite a
   misleading cgroup path) before stopping it with `SIGTERM` and starting the systemd-managed
   instance for what's apparently the first time it's actually been under supervision.
4. **Rate limiting**, once real usage patterns exist to design against, not before. Still open.

**Overall status: all three build phases done.** What's *not* yet done: real player-facing
verification — nobody has typed a message on one server and watched it appear on the other. Both
directions are verified only as far as "the code runs, posts/polls succeed, nothing crashes."

## Open questions

- Should `ChatTell`/`ChatGuild` ever bridge, or is `ChatYell`-only (and EINHORN_SURVIVAL's
  equivalent, all public chat) the permanent scope? Leaning toward "public-only, permanently" —
  a cross-server DM relay is a much bigger trust/abuse surface for very little value.
- Character identity: bridged messages are display-name-only, no shared account linkage. Given
  WOTAN already links GTA7 players to real IDUNA `player_id`s (S170-239-adjacent work), a future
  pass could show a verified badge for players whose name is provably the same person on both
  sides — not attempted here.

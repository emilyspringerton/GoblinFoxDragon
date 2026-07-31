# The Bridge: REDGARDEN ↔ apps2/mud

Wire-protocol spec for `docs2/REDGARDEN_GUI_NORTHSTAR.md` (the product-level design; read that
first). This doc is the concrete packet-level layer, same role `THE_BRIDGE_SPEC.md` plays for
SHANKPIT ↔ Bedrock, written against the real code on both sides — `REDGARDEN/packages/common/
protocol.h` and `apps2/mud/main.go`'s actual command handlers — not assumed shapes.

## Goals

- Keep `apps2/mud`'s Go server as the single authoritative game loop. The telnet listener on
  `:2323` is untouched — this adds a second listener, it doesn't touch the first.
- Reuse REDGARDEN's real, working connect-ticket handshake and UDP transport verbatim, not a new
  auth scheme.
- Every GUI action decodes into a call to the SAME internal function telnet's text commands
  already call (`cmdGo`, `cmdAttack`, `cmdWS`, `cmdShopBuy`, ...) — one action dispatch, two
  client protocols in front of it, exactly `REDGARDEN_GUI_NORTHSTAR.md` §4's own diagram.

## The honest gap this spec has to name, not gloss over

Checked directly before writing this: `apps2/mud` has **no continuous intra-zone movement
today**. `cmdGo`'s own comment says it outright — `n/s/e/w` means zone-to-zone travel, full stop;
the destination zone's fixed spawn point is where `p.pos` gets set, not a coordinate the player
steered to. `cmdAttack`'s auto-approach snaps `p.pos` directly onto the target's position rather
than walking there. So `player.pos` (`mob.Pos{X,Y,Z}`, real, already exists) is currently only
ever *teleported*, never *smoothly moved*. REDGARDEN's `PACKET_ARENA_MOVE` (a continuous
click-to-move target the server steers toward every tick, `arena_tick_movement`'s own real
behavior) has no equivalent to bridge onto — this is new server-side work, not a wiring exercise,
and it's the actual, correctly-scoped content of `REDGARDEN_GUI_NORTHSTAR.md`'s own Milestone 3
("input bridge"), not something this spec can define away. Milestone 4's own "flat plane +
box-obstacle placeholders" scoping (rendering) has a load-bearing twin on the server side: a
minimal per-zone bounding box + steer-toward-target tick, matching REDGARDEN's own
`arena_tick_movement` shape, has to exist before Milestone 3 can honestly claim done.

## Transport & handshake

New UDP listener, port TBD (proposal: `2324`, next to the existing `2323` telnet port). Reuses
REDGARDEN's real handshake verbatim — `packages/common/hmac_sha256.h` ported in (same scheme
already shared by REDGARDEN, shankpit-460, and SHANKPIT itself), `PACKET_CONNECT`'s real wire
shape:

```
NetHeader (type=PACKET_CONNECT, ...)
+ 20-byte ticket payload (16 bytes identity/expiry-relevant, 4-byte expires_at, little-endian)
+ 16-byte HMAC-SHA256 MAC (truncated), keyed on REDGARDEN_TICKET_SECRET-equivalent env var
```

`apps2/mud`'s connect handler verifies the MAC and `expires_at` exactly as `arena_server`'s
`verify_connect_ticket` does (`REDGARDEN/apps/arena_server/src/main.c`) before accepting the
client — same fail-closed default (`WARNING: ... not set -- all connect attempts will be
rejected`) if the secret env var is absent. IDUNA already mints these tickets and already gates
`apps2/mud`'s own `PacketConnect` (S76-01) — this is one more consumer of that same minting flow,
not a new IDUNA integration.

## Packet mapping

REDGARDEN's real, already-defined packets (`packages/common/protocol.h`), left as-is, mapped onto
`apps2/mud`'s real, already-defined command functions:

| REDGARDEN packet | Payload (real struct) | Maps to (real `apps2/mud` function) | Notes |
|---|---|---|---|
| `PACKET_ARENA_MOVE` | `ArenaMoveCmd{target_x, target_z, unit_owner}` | **new** — no equivalent exists, see gap above | `unit_owner` is meaningless here (no clone-control concept in the MUD); ignore that field |
| `PACKET_ARENA_ATTACK` | `ArenaAttackCmd{target_owner, commander_unit}` | `cmdAttack(p, target)` | `target_owner` (a hero-slot index in REDGARDEN) needs to become a mob/player-ID lookup instead — REDGARDEN has no concept of targeting a non-hero mob at all, this is a real shape mismatch, not just a rename |
| `PACKET_ARENA_CAST` | `ArenaCastCmd{slot, hover_target}` | `cmdWS(p, wsName)` | REDGARDEN's Q/W/R numeric slot needs a per-job slot→weapon-skill-name table (see `REDGARDEN_GUI_NORTHSTAR.md` §4.2's "Warrior first" proposal); `hover_target` unused until mob/player targeting exists |
| `PACKET_ARENA_SHOP_BUY` | `ArenaShopBuyCmd{item_id}` | `cmdShopBuy(p, itemID)` | Closest real 1:1 mapping in this whole table — `apps2/mud`'s shop already takes an item ID string; REDGARDEN's `uint8_t item_id` just needs a lookup table into that string space |
| `PACKET_ARENA_SHOP_SELL` | `ArenaShopSellCmd{slot}` | `cmdShopSell` / `cmdUnequip` | REDGARDEN sells by equip *slot*; `apps2/mud`'s `cmdShopSell` takes an item ID — needs the equip-slot → currently-equipped-item lookup `cmdUnequip` already does |
| `PACKET_ARENA_PICK` | `ArenaPickCmd{hero_id}` | **not used** | No hero draft in an MMO — a character's job comes from `charJob`/`jobID`, set once at character creation, not picked per match. This packet type is dropped entirely for this bridge, not mapped |
| `PACKET_CONNECT` | ticket payload | new `apps2/mud` connect handler | See Transport & handshake above |

Not attempted here: chat (`cmdSay`/`cmdTell`/`cmdYell`/`cmdGuildChat`) has no REDGARDEN packet
type at all — REDGARDEN has no in-match text chat. A new packet type is needed for GUI-side chat
input, not scoped in this pass.

## Snapshot format (server → client)

REDGARDEN's `PACKET_ARENA_SNAPSHOT`/`PACKET_ARENA_SNAPSHOT_HEROES` broadcast a flat per-hero
struct (`x, z, hp, max_hp, alive, hero_id`) every tick, and every REDGARDEN visual effect
(`attack_flash`, `heal_flash`, the whole HP-delta-driven idiom this session leaned on constantly)
is *reconstructed client-side* from consecutive snapshots — there's no explicit "you got hit"
event on the wire at all, deliberately, for REDGARDEN's own MOBA scope.

That idiom is too thin for `apps2/mud`'s real combat depth. A weapon-skill hit, a skillchain
resonance, a status-effect application, an enmity shift — none of these reduce to "HP went down"
without losing the exact information REDGARDEN's own GUI would need to render the right visual
(cast ring vs. skillchain flash vs. poison tick are different UI, not the same generic
`attack_flash`). Proposed departure from REDGARDEN's own wire philosophy: the new snapshot needs
a genuine **event list**, not just a state diff:

```c
typedef struct {
    float x, y, z;
    uint16_t hp, max_hp;
    uint16_t mp, max_mp;
    uint8_t tp;              /* 0-100, drives the cast-ring "ready" state for cmdWS */
    uint8_t zone_id;
    char target_mob_id[32];  /* "" if untargeted -- mob.Mob.ID is a real string, not a slot index */
} MudCharacterState;

typedef struct {
    uint8_t event_type;      /* WS_HIT, SKILLCHAIN, STATUS_APPLIED, MOB_DIED, ... -- not designed here */
    char actor_id[32];
    char target_id[32];
    int32_t amount;          /* damage/heal magnitude, 0 if not applicable */
    char label[32];          /* weapon skill name / skillchain resonance / status name */
} MudEvent;
```

Exact `event_type` enum, batching/reliability (UDP has no delivery guarantee — does a missed
`MOB_DIED` event leave a client's UI stuck mid-fight?), and nearby-mob/nearby-player visibility
rules are **not designed here** — this is the shape of the departure from REDGARDEN's flat-diff
idiom, not the finished wire format.

## Server responsibilities

- Run the new UDP listener alongside the existing telnet listener; same 1Hz game loop drives both,
  no second simulation.
- Verify connect tickets exactly as `arena_server` does; reject unauthenticated packets.
- Decode `PACKET_ARENA_*` commands into the SAME `cmd*` function calls telnet already makes — no
  parallel/duplicate combat-resolution path.
- New: per-tick position steering toward a `PACKET_ARENA_MOVE` target, bounded by a per-zone box
  (see the gap section above) — does not exist yet.
- Serialize `MudCharacterState` + recent `MudEvent`s every tick to connected GUI clients.

## Client responsibilities (REDGARDEN fork side)

- Send `PACKET_CONNECT` with a real IDUNA-minted ticket; render the existing draft/pick screen
  path is dropped entirely (no `PACKET_ARENA_PICK`) — character select becomes "which existing
  IDUNA character," not "which hero this match."
- Render `MudCharacterState` using REDGARDEN's existing hero-silhouette/HP-bar/HUD machinery.
- Render `MudEvent`s using REDGARDEN's existing flash/cast-ring vocabulary, keyed off
  `event_type` once that enum is designed — not built yet.

## Explicitly not resolved here

- The `event_type` enum and reliability story (see Snapshot format above).
- `PACKET_ARENA_ATTACK`'s target-ID shape mismatch (hero-slot index vs. mob/player string ID) —
  needs a real design pass, not a one-line fix.
- Per-job slot→ability mapping (`REDGARDEN_GUI_NORTHSTAR.md` §4.2/§6 Milestone 5's own open
  scope).
- New chat packet type.
- Server-side intra-zone movement/steering implementation itself (named as a real gap above, not
  designed here — this doc identifies the work, doesn't do it).

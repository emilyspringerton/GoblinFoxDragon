# SSH Transport & Identity — NORTHSTAR

**Status:** Stage 1 (§4 economy gate + Amendment 1 chat/guest-tier gate) shipped 2026-09-12.
Stages 2-8 scoped, not started. **Source spec is founder-authored and verbatim-authoritative** —
this doc is the phased-status tracker + real, checked findings against this codebase, not a
paraphrase. Full source spec text lives in `docs2/SSH_TRANSPORT_IDENTITY_SPEC.md` (verbatim) and
`docs2/SSH_TRANSPORT_IDENTITY_AMENDMENT_1.md` (verbatim) — read those first for the real
requirements; this doc tracks what's actually built against them.

## Why this exists

Two real, live problems named by the founder: (1) telnet has no per-packet integrity — an
on-path attacker can inject commands into an already-authenticated session, and a bank/AH/bazaar
are reachable over that stream; (2) a character is claimed by typing its name — no accountable
identity, so any account with accumulated value is takeable by anyone who reads or guesses the
name. SSH (encrypted transport + public-key identity, no server-held secret) fixes both. Full
8-stage rollout order in the source spec §8 — each stage independently shippable, §4 named
explicitly as "the actual security fix... ship it first."

## Real, checked finding that changed the plan: telnet characters are ALREADY real and persistent

Amendment 1 §C.1 describes telnet's "current behavior" as ephemeral ("a typed name creating a
fresh level 1, duplicable, non-persistent" character) and asks that this be made deliberate.
Checked directly against this codebase before implementing anything: that description does NOT
match what's actually implemented. `apps2/mud/main.go`'s `mudCharCache` (`var/mud-chars.json`,
name → IDUNA `character_id`) means a telnet character is a REAL, named, IDUNA-persisted row —
real level (now real *per-job* levels, GFD-124433), real job, real gold, real inventory, all
looked up and resumed by name via `GetCharacter(cachedID)`.

Raised directly to the founder rather than either (a) silently implementing literal ephemerality
(which would have meant stripping persistence from every existing telnet character — a large,
likely-irreversible, unrequested destructive change) or (b) silently ignoring the spec's own
explicit ephemerality requirement. **Founder's real, live-tested answer:** the underlying IDUNA
rows staying real and persistent is fine (even useful — e.g. future NPC-driven market dynamics
that react to what ephemeral players consumed before disconnecting), but from a **player's own
actual, tested experience**, reconnecting with the same name does *not* resume a character today
— confirmed by the founder playing it directly. Real quote: "does not need to be fixed these
specs supersede any needed fixes and reframes it as a feature." So the resume-by-name code path's
real-world unreliability is treated as the intended guest-tier behavior going forward, not a bug
to hunt down and repair — the connect banner (below) states this plainly rather than promising a
resume that doesn't reliably happen.

## Stage 1 — SHIPPED 2026-09-12 (§4 + Amendment 1 §A/B/C.1-banner/D-partial)

**Real, server-layer command gate** (`apps2/mud/main.go`'s `guestGate`, called from `handle()`
before its own dispatch switch, not hidden behind a menu):

- **Blocked for every guest (= every real telnet connection today):** `bank`, `bazaar`, `ah`
  (economy, §4 core) and `say`/`'`, `tell`/`t`, `yell`/`y`, `guild`/`g`, and the `/p` party-chat
  shortcut (chat, Amendment 1 §B.1/B.3).
- **Explicitly left permitted**, matching Amendment 1 §B.4 exactly: movement, combat, jobs/
  subjobs (including the new per-job leveling, GFD-124433), party/linkshell *membership*
  commands (`party`, `invite`, `accept`, `ls-create`, `ls-invite` — chat is gated, membership
  isn't), quests, exploration, dungeons/NMs.
- Every refusal explains the real reason and the real upgrade path (Amendment 1 §D: "explain the
  upgrade rather than only denying") rather than a bare denial — e.g. "[Guest] The bank, auction
  house, and bazaar are SSH-only... SSH key binding is coming soon."
- Structured log line on every blocked attempt (`log.Printf("[guest-blocked] slot=%s cmd=%q
  category=%s", ...)`) — the real, minimal slice of Amendment 1 §E's instrumentation ask (full
  conversion-rate/griefing-signal dashboards are real, separate, deferred work, not built here).
- Connect banner states guest-tier scope and the real, tested non-persistence-by-reconnect
  behavior **before** the name prompt (Amendment 1 §C.1 + §D banner requirements).
- New `player.isGuest bool` field: `true` for every real `handleConn` (telnet) session, `false`
  (Go's own zero value) for a headless/Town-GUI session — those already carry a real WOTAN-
  authenticated `characterID` handed in by the caller, not a typed name, so they're never the
  anonymous-identity attack surface this spec is about. No SSH exists yet, so `isGuest` is
  `true` for literally every live connection right now — that's the real, honest, current state
  of the world, not a bug; stage 5 (identity) is what gives a connection a path to `isGuest =
  false` some other way than being headless.

**Real, live-verified, not just unit-tested:** built a throwaway instance (`-port 2399`, isolated
`var/` via a scratch working directory — the real, live production instance on `:2323` was never
touched or restarted), connected with a real Python socket script (no client, no library),
confirmed: the guest banner prints before the name prompt; `bank`/`bazaar`/`ah`/`say`/`tell`/
`yell`/`guild`/`/p` all return the real explain-the-upgrade message; `look`/`jobs`/movement/combat
remain fully functional. Matches the spec's own acceptance requirement ("verified by direct
socket testing, not client-side inspection") literally, not just in spirit.

**11 new tests** (`apps2/mud/guest_gate_test.go`): every blocked command, every permitted command,
non-guest (headless) sessions are never blocked, and the refusal messages actually mention SSH
(not a bare denial). `GOWORK=off go build/vet/test ./...` clean across the whole module.

### Real, honest, NOT done in Stage 1 (named, not silently skipped)

- **Amendment 1 §C.2/C.3 (guest display marker, reserved-name protection):** not built. With
  zero SSH identities existing yet, every single connection is a guest — a `[guest]` tag on
  literally everyone has no differentiating value today, and there's nothing yet to reserve a
  name *against* (no SSH-bound name exists to protect). Real, deliberate sequencing: build this
  once stage 5 (identity) gives the codebase its first non-guest names to distinguish against,
  not before.
- **Amendment 1 §C.4 (idle eviction, per-IP guest session cap):** not built — real, separate
  resource-limit work, more naturally paired with §2.4's own general connection-limit
  requirements (stage 4) than bolted onto the chat/economy gate alone.
- **Amendment 1 §E (full instrumentation):** only the one log line above shipped. Guest session
  count/duration/source-IP distribution, name-collision/reserved-name-rejection tracking,
  guest→SSH conversion events, and griefing signals (kill-steal rate, NM tag contention) are all
  real, separate, not-yet-built follow-up — the doc itself names conversion rate as the number
  that eventually justifies stage 8 (telnet deprecation), so this matters before that decision
  point, not before Stage 1 ships.
- **§4's own "free-text field" list beyond chat commands** (bazaar listing text, AH notes,
  character bios/emote/custom titles) — checked directly: this codebase has no character bio,
  emote, or custom-title feature at all today, and bazaar/AH are already fully economy-gated
  (a guest can't reach the listing-text field because it can't reach `bazaar`/`ah` at all). No
  real gap found here, not silently skipped.

## Stages 2-8 — scoped by the source spec, not started

Real, honest status against each remaining stage (source spec §8's own numbering):

2. **§7 TLS on GUI login.** Not investigated this pass. Real, separate, likely-different-repo
   work (the GUI login almost certainly lives in `battlegrounds_gui`/Town, not `apps2/mud`) —
   needs its own real investigation before any implementation claim.
3. **§5 process isolation + data backup.** Not started. Real prerequisite work
   (systemd hardening directives, dedicated unprivileged user, read-only filesystem except one
   data path) against whatever unit currently runs the live `:2323` service.
4. **§2 SSH listener (high port, no identity yet).** Not started. Real, substantial new
   capability — an SSH server implementation (public-key-only, TOFU, PTY/resize handling, host
   key generated once and persisted outside the repo) feeding the existing game loop the same
   read/write stream telnet already does. `apps2/mud`'s own command dispatch (`handle()`) is
   already transport-agnostic enough that an SSH-fed session should be able to reuse it directly
   — a real, positive finding from this pass, not yet exercised.
5. **§3 identity binding + key management.** Not started. Depends on stage 4 existing first
   (there's no SSH public key to bind a fingerprint to yet).
6. **§1 port swap to 22.** Not started. Depends on stages 3-5.
7. **§6 device-flow hardening (number matching, OTP-to-session binding, TTL/attempt caps).** Not
   investigated this pass — real, separate work against whatever WOTAN/IDUNA device-flow
   implementation already exists (`IDUNA/internal/auth/device`).
8. **Telnet deprecation notice.** Explicitly gated on real conversion-rate data per the source
   spec's own §D/E — not a decision to make until stage 5+ has been live long enough to measure.

## Related

- `docs2/SSH_TRANSPORT_IDENTITY_SPEC.md` — verbatim source spec (§0-§9)
- `docs2/SSH_TRANSPORT_IDENTITY_AMENDMENT_1.md` — verbatim Amendment 1 (guest tier + channel
  gating, supersedes §4's permitted/blocked lists where they conflict)
- `GFD-124433` (EMILY/BACKLOG.md SECTION 389) — per-job leveling, shipped same session,
  unrelated feature but touches the same `apps2/mud/main.go` command-dispatch area

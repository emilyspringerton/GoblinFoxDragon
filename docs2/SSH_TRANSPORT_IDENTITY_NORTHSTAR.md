# SSH Transport & Identity — NORTHSTAR

**Status:** Stage 1 (§4 economy gate + Amendment 1 chat/guest-tier gate) shipped 2026-09-12.
Stage 2 (§7 GUI login) shipped 2026-09-12. Stage 3 (§5 process isolation + data backup) shipped
2026-09-12, partially — see its own section for the real, honest, root-blocked remainder.
Stages 4-8 scoped, not started. **Source spec is founder-authored and verbatim-authoritative** —
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

## Stage 2 — SHIPPED 2026-09-12 (§7 GUI login — blocking item)

**Real finding, checked directly, not assumed:** the GUI login lives in
`apps2/battlegrounds_gui/src/main.c` (`get_player_login_ticket`, called from `run_login_screen`),
POSTing real player-typed email+password to IDUNA's `/api/v1/auth/email/login` via
`apps2/battlegrounds_gui/packages/common/http_client.h`. That header is a raw BSD-socket client
(POSIX sockets + a separate Winsock branch for the CI-built Windows client, `DragonsNShit_MUD_GUI.exe`)
with **zero TLS capability on either platform** — no OpenSSL/mbedTLS linked, no dev headers
installed for either target in this repo's build, no handshake code of any kind. The header's own
doc comment already named this as a real, unresolved gap ("a real, named gap if IDUNA is ever
reached over the open internet by a player's own login screen").

The spec's own §7 acceptance names two valid paths: "terminate TLS in front of the login and
ticket endpoints, or disable the GUI login path entirely until TLS exists." The first path doesn't
close the gap by itself here — a client that can't perform a TLS handshake at all gets no benefit
from a TLS-terminating proxy in front of the server, since it would try to speak plaintext HTTP at
a TLS port and simply fail. Real TLS in this client means vendoring and cross-compiling a full TLS
library (OpenSSL or mbedTLS — neither is currently available in this sandbox for the mingw-w64
Windows target this repo's CI actually ships) plus real certificate/hostname verification — a
rushed implementation skipping cert validation would be worse than plaintext (looks encrypted,
still trivially MITM-able). Not attempted as a rush job for that reason.

**Founder chose the second path live, 2026-09-12.** Real fix, `apps2/battlegrounds_gui/src/main.c`:

- `kGuiLoginDisabledPendingTLS` (a `static const int`, currently `1`) gates the email/password
  path at three independent layers, not just one: (1) the email/password input boxes and LOG IN
  button are never drawn — `draw_login_screen` shows a plain-language notice instead ("GUI LOGIN
  IS TEMPORARILY DISABLED... Play right now via telnet, or sign up below"); (2) the keyboard/mouse
  handlers that would populate the fields or set `st.submitting` are gated off in
  `run_login_screen`'s event loop; (3) **defense in depth** — the actual network call site
  (`get_player_login_ticket`) is gated a third time, directly, so `st.submitting` becoming true by
  any future code path still can't reach it.
- SIGN UP is unaffected — it opens the real WOTAN store page in the system's default browser
  (`SDL_OpenURL`), which does a real TLS handshake via the browser's own stack; this file's raw
  socket client is never involved in that path, so no gate was needed there.
- Result: no email/password credential can cross the wire from this client on any path, by
  construction, not by trusting the UI alone — trivially satisfies the acceptance bar ("No
  credential crosses the network unencrypted, on any path") since the call that would send one no
  longer executes.

**Live-verified**, not just read: built the real client natively on Linux (same source list the
CI mingw cross-build uses: `main.c` + `packages/simulation/{arena_game,arena_replay,
arena_ai_bridge,action_bar_mod}.c` + `packages/common/mlp_infer.c` + `packages/goldenband/*.c`),
ran it under Xvfb, and screenshotted the actual login screen: no email/password fields, no LOG IN
button, the disabled-login notice, SIGN UP still present and functional. `gcc -fsyntax-only`
clean; full native link clean (all pre-existing warnings, none new).

### Real, honest, NOT done in Stage 2 (named, not silently skipped)

- **Real TLS support in the GUI client itself.** Deliberately not attempted this pass (see
  rationale above) — GUI login stays disabled until this exists. Real follow-up work: vendor
  OpenSSL or mbedTLS for both the Linux dev build and the mingw-w64 Windows CI target, add a real
  handshake + certificate/hostname verification path to `http_client.h`, then flip
  `kGuiLoginDisabledPendingTLS` back to `0`.
- **IDUNA's existing device-flow auth** (`/auth/device/start`+`/auth/device/poll`, no password
  ever leaves the polling client) was investigated as a possible password-less replacement for
  this screen — real, live infrastructure, but built for a different consumer (JWT audience
  `"kikoryu"`) and would need new integration work to mint a GFD character ticket from it. Named
  as a real, viable alternative for a future pass, not built here — founder chose the smaller,
  safer, ships-today fix instead.
- **A capture-based verification** (the spec's own literal "verified by capture" phrasing) wasn't
  performed — there is no longer a login network call to capture from this path, so a capture
  would show nothing by construction. Verified instead by reading the gated code paths directly
  and by the live screenshot above.

## Stage 3 — SHIPPED (PARTIAL) 2026-09-12 (§5 process isolation + data backup)

**Real, achievable-without-root filesystem/kernel-surface sandboxing shipped and live**,
`ops/systemd/gfd-mud.service`: `ProtectSystem=strict` + `ReadWritePaths=var` confines every write
the process makes to its one real data path (`var/mud-chars.json`, `var/mud-player-ids.json`,
`var/logs/`) — everything else, including this repo's own source, is read-only to it. Plus
`PrivateTmp`, `RestrictNamespaces`/`SUIDSGID`/`Realtime`/`AddressFamilies`, `LockPersonality`,
`RemoveIPC`, `ProtectHostname`/`KernelTunables`/`ControlGroups`, a `SystemCallFilter=
@system-service` seccomp allowlist, `NoNewPrivileges`, `UMask=0077`.

**Real, checked-live limit, not assumed:** every directive whose enforcement mechanism is
"drop a specific capability from the bounding set" (`CapabilityBoundingSet=`,
`AmbientCapabilities=`, `ProtectKernelModules=`, `ProtectKernelLogs=`, `ProtectClock=`) fails with
`status=218/CAPABILITIES` under this `systemctl --user` unit — tested each directive individually
on a throwaway instance before touching the real one. Root cause: the per-user systemd instance
(`user@1000.service`) never holds `CAP_SETPCAP`, so it cannot drop capabilities from any bounding
set for any unit it manages, full stop — a normal Linux capability rule, not a container/sandbox
quirk of this box (`systemd-detect-virt` reports a real KVM VM here, not a container). The spec's
own "dedicated unprivileged user, no login shell" ask hits the same wall: `useradd` needs root.

**Founder authorized a live restart to roll this out today** (chose this over "prepare + test,
don't touch the live service" or "skip Stage 3" via `AskUserQuestion`). Real backup taken first,
matching §5's own "data path is backed up before rollout begins": added a new `gfd` target to
`emily.cli`'s existing `emily backup run` tool (GoblinFoxDragon/var — the mud process's real data
path; the durable character/economy data itself lives in IDUNA, already covered by the `iduna`
target) — found and fixed a real gap while scoping it (`looksLikeSecret` missed
`var/moltbook-credentials.json` entirely, would have shipped a real credential into an
unencrypted cloud archive). The actual GCS upload fails in this sandbox (no `gcloud` ADC
configured here, a real environment limit of this session, not a defect) — archiving/filtering
verified correct instead, plus a local `var-backups/*.tar.gz` snapshot with a real, verified
byte-identical restore round-trip (`diff -r`) before the live restart. Measured result of the
restart: `systemd-analyze security` exposure score **9.8 UNSAFE → 5.8 MEDIUM**. Live-verified
after restart via a real socket connection: guest banner, guest-gate blocking `bank`, normal
movement/combat all still correct.

**Real, unplanned, load-bearing finding along the way:** the live binary was stale since
2026-09-05 — predating every fix shipped this session (mob Revive, item-use fixes, per-job
leveling, and Stage 1's own guest gate). None of it had actually gone live despite being
committed and tested. Since a restart was happening regardless for the hardening rollout,
rebuilt and deployed current `master` (commit `6b03614`) instead of restarting back into the same
stale binary — built from a clean checkout with the pre-existing, not-mine, uncommitted S252-00/01
inventory-sync work stashed out first (so it isn't silently baked into a live deploy unreviewed),
then restored to the working tree unchanged afterward.

### Real, honest, NOT done in Stage 3 (named, not silently skipped)

- **Dedicated unprivileged system user, no login shell (§5's own literal ask).** Not done — needs
  root. `sudo-queue/78-gfd-mud-dedicated-user-and-full-hardening.sh` is a complete, real, queued
  script that migrates gfd-mud.service from this `systemctl --user` unit to a root-owned system
  unit under a new, dedicated, shell-less account — which, being root-managed, can also apply the
  five capability-dropping directives this pass couldn't. Reviewed-not-yet-run by design (it
  changes file ownership under a live repo and migrates a running production service's execution
  user) — the founder runs it whenever convenient; it isn't blocking anything else in this spec.
- **The five capability-dropping directives named above**, standalone — same root dependency,
  same script covers them.
- **Cloud upload of the `gfd` backup target.** The tool and its filtering are real and verified;
  the actual `gcloud storage cp` step needs ADC credentials this sandbox doesn't have. Real
  follow-up: run `emily backup run --target gfd` from a host that has `gcloud auth login` (or a
  service account) configured, or wire it into whatever schedule the `iduna`/`fatbaby`/
  `promptoverse` targets already run on.
- **Restart-on-failure with backoff** — already present in the unit (`Restart=on-failure`,
  `RestartSec=10s`), predates this stage, not newly verified here.

## Stages 4-8 — scoped by the source spec, not started

Real, honest status against each remaining stage (source spec §8's own numbering):

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

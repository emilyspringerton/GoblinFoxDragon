# Amendment 1 — Guest Tier & Channel Gating

Amends: `SSH_TRANSPORT_IDENTITY_SPEC.md` §4. Status: Requirements only. Supersedes §4's
permitted/blocked lists where they conflict. Read §4 first. This narrows it; it does not replace
the rest of the spec.

(This also supersedes an earlier, shorter same-day amendment to §4 that moved `say`/`tell`/`yell`/
linkshell chat/party chat/free-text-fields to blocked — that amendment's real content is fully
subsumed by §B below, which is more complete and formally the same requirement.)

## A. Decision: no authentication on telnet

Telnet gets no login, no OTP, no device flow, no credential of any kind.

Rationale — authentication protects something, and after this amendment a telnet session holds
nothing:

* No persistence, no economy, no trade, no chat, no accountable identity.
* An OTP would authenticate the handshake, not the stream. The channel remains plaintext and
  injectable either way, so the added friction buys no actual trust.
* Telnet's only job is zero-friction first contact. A login prompt on the connection being
  advertised as "any raw TCP client works" defeats the reason it exists.

Everything requiring identity lives on SSH. Telnet is a demo that shares a world.

## B. Channel gating (amends §4)

### B.1 Move to blocked

All player-originated text on telnet is blocked. Specifically:

* `say`, `tell`, `yell`, and any other player-to-player message channel
* Linkshell chat
* Party chat
* Any free-text field another player can read — bazaar listing text, AH notes, character
  descriptions, emote text, custom titles

Rationale: an unauthenticated, unbannable, zero-cost identity is an impersonation primitive.
Economy gating (§4) stops direct theft. Chat gating stops talking someone into handing it over.
Social engineering routes around a permissions boundary rather than breaking it, so both gates
are required — neither is sufficient alone. The specific threat is a guest posing as a linkshell
officer or known player and requesting mats, gil, or gear from an authenticated player.

### B.2 Remains permitted

Server-emitted text: combat resolution, zone and room descriptions, skillchain and magic burst
announcements, NM spawn notices, system broadcasts, conquest tallies.

### B.3 Directed messages must be blocked in both directions

An SSH player must not be able to `tell` a guest, and a guest must not be able to receive one.
One-directional blocking leaves the guest usable as a receiving channel for a reply-based con.
Guests are not addressable by name from any channel.

### B.4 Confirmed permitted (unchanged from §4)

Movement, combat, jobs, subjobs, weapon skills, skillchains, magic burst, dungeons, NMs, parties,
linkshells (membership only, not chat), exploration, quests, duels, leaderboard.

Known accepted risk: movement and combat remain griefable by anonymous accounts — kill-stealing,
train-pulling, NM tagging. Accepted for now. Visible, recoverable, and low severity relative to
impersonation. Revisit if abused; instrument so abuse is detectable (§D).

## C. Guest tier model

### C.1 Ephemerality is explicit, not incidental

Current behavior (a typed name creating a fresh level 1, duplicable, non-persistent) is correct
for this tier. Make it deliberate and documented rather than emergent:

* No state written to durable storage on disconnect.
* Reconnecting with the same name yields a new character, not a resumed one.
* Stated plainly in the connect banner, before the name prompt.

### C.2 Namespacing

Guests must be unmistakable at a glance to an authenticated player:

* A persistent visible marker on the character — a `[guest]` tag, a per-session suffix
  (`Gandalf#a3f`), or both. Implementer's choice of form; the requirement is that it appears
  everywhere a guest name is rendered.
* Applies to: `who`, zone occupant lists, combat logs, treasure pool rolls, party rosters,
  leaderboards, and any server broadcast naming the character.

### C.3 Reserved names

Any name bound to an SSH identity is reserved against guest use. A guest may not appear as
`Gandalf` if a persistent `Gandalf` exists.

Without this, impersonation returns through the display layer after being closed in chat.
Reservation is permanent for the life of the bound account, not released on logout.

### C.4 Session hygiene

* Idle eviction for guest sessions.
* Cap concurrent guest sessions per source IP.
* Guests excluded from persistent leaderboards; a separate ephemeral board is fine.

## D. Upgrade path

With chat and economy gated, telnet is a combat and progression demo. The SSH upgrade must be
discoverable early, not surfaced at the moment of first refusal.

* Connect banner states what the guest tier is and how to bind a key.
* Blocked-command responses explain the upgrade rather than only denying.
* A periodic, low-frequency, non-intrusive in-session reminder.

Do not make the prompt nagging or modal. The demo has to remain pleasant to play, because it is
the entire acquisition funnel.

## E. Instrumentation

Log, with enough structure to answer "is this being abused":

* Guest session count, duration, source IP distribution
* Guest name collisions and reserved-name rejection attempts
* Blocked-command attempts by command and by source
* Guest→SSH conversion events
* Griefing signals: kill-steal rate, NM tag contention, repeated same-IP guest churn

Conversion rate is the number that decides when telnet deprecation (§8 stage 8) is justified.

## F. Acceptance

* [ ] No player-originated text can cross from a guest session to any other player, by any path
* [ ] `tell` to a guest fails; `tell` from a guest fails
* [ ] Guest characters render with their marker in every listing and broadcast
* [ ] An SSH-bound name cannot be claimed by a guest
* [ ] Guest disconnect writes nothing durable; reconnect yields a new character
* [ ] Banner states ephemerality before the name prompt
* [ ] Combat, skillchains, dungeons, and progression remain fully playable as a guest
* [ ] Blocked-command attempts and conversion events appear in logs
* [ ] All of the above verified by direct socket testing, not client inspection

## G. Out of scope

* Any authentication mechanism on telnet
* Anti-griefing systems beyond instrumentation
* Changes to the SSH identity model (see parent spec §3)

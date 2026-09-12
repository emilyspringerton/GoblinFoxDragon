# SSH Transport & Identity — Implementation Spec

**Status:** Requirements only. Implementer owns all design and code decisions.
**Audience:** Implementation agent with full repo access.
**Placeholders:** `<GAME_HOST>` = public hostname. `<ADMIN_ADDR>` = private/VPN address.

---

## 0. Why

Two problems, one change:

1. **Plaintext session hijack.** Telnet has no per-packet integrity. Any on-path attacker can
   inject commands into an *already-authenticated* session. With a bank, auction house, and
   player bazaars reachable over that stream, strong login auth does not help — the attacker
   waits until the player authenticates and then types as them.
2. **Nameless identity.** A character is claimed by typing its name. Published to a large
   audience, every account with accumulated value is takeable by anyone who guesses or reads
   the name.

SSH solves both: encrypted, integrity-checked transport, and public-key identity with no
server-held secret.

**Non-goal:** removing telnet. Telnet is the frictionless front door and stays, gated.

---

## 1. Port topology

Requirement: a human or agent typing `ssh <GAME_HOST>` with no flags must reach the game.
Admin SSH must never be reachable on the public interface.

| Service | Bind | Notes |
|---|---|---|
| Game SSH | `0.0.0.0:22` | public, game-only |
| Admin sshd | `<ADMIN_ADDR>:22` | private iface / VPN / Tailscale |
| Telnet | `0.0.0.0:2323` | unchanged, gated per §4 |

Both daemons may use port 22 — they bind different addresses. If no private interface exists
yet, the interim is admin sshd on a high port; game still takes 22.

**HARD CONSTRAINT: the game process MUST NOT be capable of granting shell access.** Do not
implement an admin bypass inside the game server. No "if fingerprint == mine, spawn shell."
The game binary is the most exposed, fastest-changing code on the box; it must never be the
thing standing between an attacker and root. Admin access comes from a separate, unmodified
sshd process or it does not exist.

### Acceptance
- [ ] `ssh <GAME_HOST>` from an external network reaches the game
- [ ] Admin sshd is unreachable from any public interface (verify by external port scan)
- [ ] Game process contains no code path to a shell, subprocess, or privilege escalation
- [ ] Changes to admin sshd are validated in a **second** session before the first is closed

---

## 2. Game SSH server

### 2.1 Transport
- Accept SSH connections; hand each session to the existing game loop as a read/write stream.
- **Public key auth only.** Password auth disabled. Keyboard-interactive disabled. If the server
  can accept a password it will hold a secret, and holding secrets is what this change removes.
- Trust-on-first-use: any offered public key is accepted for connection. Authorization to a
  *character* is separate (§3).
- Accept and ignore the SSH username, or treat it as a character-name hint. Must not error.

### 2.2 Host key
- Generated once, persisted outside the repo, included in deploy secrets and backups.
- Regenerating it triggers MITM warnings for every returning client. Treat as permanent.

### 2.3 Terminal
- Honor PTY requests: initial window size and subsequent resize events.
- Degrade cleanly with no PTY (agents frequently connect without one) — no crash, no hang.
- Game output currently assumes a dumb stream; verify ANSI and wrapping against a real terminal.

### 2.4 Resource limits
- Idle timeout, max session duration, max concurrent sessions per source IP.
- Auth attempt limit per connection.

### Acceptance
- [ ] Password auth is refused
- [ ] Session works with and without PTY
- [ ] Resize events reflow correctly
- [ ] Host key survives restart and redeploy

---

## 3. Identity

### 3.1 Model
- A character binds to an **SSH public key fingerprint**.
- Unknown key → claim flow → fingerprint bound to chosen name, permanently.
- Known key → resume that character. No prompt, no secret transmitted.
- One WOTAN account may hold **multiple** fingerprints.

### 3.2 Key management (in-session, authenticated only)
- Add an additional public key to the current account.
- List bound keys.
- Revoke a key **other than the one in use**.

Rationale: without this, a lost key is a permanently lost character and you own that support
burden forever.

### 3.3 Abuse controls
- Rate-limit new character claims per source IP per hour.
- Reserve an operator namespace before any public announcement.
- Name validation: length, charset, collision, impersonation of reserved names.

### 3.4 WOTAN linkage
- SSH fingerprints and browser-based WOTAN logins resolve to the **same** account and the same
  character. One identity, two credential types.
- Humans may use the device flow (§6). Agents use keys. Neither is required to use the other.

### Acceptance
- [ ] Same key twice → same character, no prompt
- [ ] Different key, same claimed name → refused, not silently reassigned
- [ ] Second key added, then first revoked, still resolves to same character
- [ ] Claim rate limit enforced under a scripted burst

---

## 4. Telnet tiering

Telnet stays open and anonymous. It must carry **nothing worth stealing**.

**Permitted on telnet:** movement, combat, jobs, subjobs, weapon skills, skillchains, magic
burst, dungeons, NMs, parties, linkshells, chat, exploration, quests, duels, leaderboard.

**Blocked on telnet — must be enforced server-side, not hidden in the UI:**
- Bank (any operation)
- Auction house (list, buy, sell, cancel)
- Bazaar (list, buy)
- Player-to-player trade or item transfer of any kind
- Any durable item or currency write that survives session end

Blocked commands return a message directing the player to SSH.

**This gate is the actual security fix and is independent of §2 and §3.** Ship it first. It can
land in hours; the rest can take weeks.

### Acceptance
- [ ] Every blocked command refused over telnet at the server layer
- [ ] Confirmed by direct socket testing, not client-side inspection
- [ ] No path exists by which a telnet session mutates persistent economy state
- [ ] Combat and progression remain fully playable over telnet

---

## 5. Process isolation

Game SSH daemon:
- Dedicated unprivileged user, no login shell
- Never root; bind above 1024 or grant capability without root
- Filesystem: read-only except an explicit data path
- No new privileges; private tmp; restricted device and kernel access
- Restart-on-failure with backoff
- Structured logs: auth outcomes, claims, rate-limit hits, blocked-command attempts

**Compromise of the game must yield a shell-less user with access to one directory. Nothing more.**

### Acceptance
- [ ] Process runs as non-root, verified live
- [ ] Cannot write outside its data path
- [ ] Cannot spawn a shell
- [ ] Data path is backed up before rollout begins

---

## 6. Device flow fixes (human/browser path)

The existing OTP flow is an OAuth device authorization grant (RFC 8628) — correct pattern,
incomplete hardening. Required:

1. **Number matching.** Display a short code in the terminal session. The browser prompt
   presents three options and requires selecting the matching one. Defeats push fatigue —
   a blind "approve?" prompt is approvable by a spammed user who never saw the terminal.
2. **Bind OTP to the pending connection**, not only the account. An attacker's session must not
   be able to consume an OTP the legitimate user generated for their own.
3. **TTL ≤ 90s, single use, ≤ 3 attempts, invalidate on failure.**
4. **Context in the approval prompt:** source IP, timestamp, what is being authorized.

### Acceptance
- [ ] Approving without seeing the terminal is not possible
- [ ] OTP issued for session A cannot authenticate session B
- [ ] Expiry and attempt cap enforced server-side

---

## 7. GUI login — blocking item

The GUI login currently transmits email and password without TLS.

This is the only item in this spec that harms **third parties** rather than the operator:
password reuse means a sniffed login compromises the user's unrelated accounts.

**Required before any public announcement:** terminate TLS in front of the login and ticket
endpoints, or disable the GUI login path entirely until TLS exists.

### Acceptance
- [ ] No credential crosses the network unencrypted, on any path, verified by capture

---

## 8. Rollout order

Each stage is independently shippable and independently valuable.

| # | Stage | Gate to next |
|---|---|---|
| 1 | §4 telnet economy gate | Blocked commands verified by socket test |
| 2 | §7 TLS on GUI login | Capture shows no plaintext credential |
| 3 | §5 process isolation + data backup | Restore from backup tested |
| 4 | §2 SSH listener, high port, no identity | Sessions stable with/without PTY |
| 5 | §3 identity binding + key management | Key add/revoke round-trips |
| 6 | §1 port swap to 22 | Admin access verified in a second session first |
| 7 | §6 device flow hardening | — |
| 8 | Telnet deprecation notice | Only when SSH traffic justifies it |

**Do not announce publicly before stages 1 and 2 are complete.** Everything after that can land
while the game is live.

### Rollback
Every stage must be revertible without data loss. Before stage 6, confirm a working admin path
that does not depend on the change being made.

---

## 9. Out of scope

- Rewriting game logic
- Instancing, PvP arena repair, GUI scene coverage
- Any cryptocurrency, token, staking, or bonding mechanism
- Any real-money entry into the economy

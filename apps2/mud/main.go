// DragonsNShit MUD — text-based telnet interface integrating all server packages.
//
// Connect: nc localhost 2323   (or telnet localhost 2323)
// Port is configurable via -port flag.
//
// Packages wired: mob, combat/tp, status, zone, gather/mining, xp, homepoint,
// field, party (XP chain), skillchain (weapon skills + SC detection).
// Game loop ticks at 1 Hz.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"dragonsnshit/server/field"
	"dragonsnshit/server/gather"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/party"
	"dragonsnshit/server/skillchain"
	"dragonsnshit/server/status"
	"dragonsnshit/server/xp"
	"dragonsnshit/server/zone"
	combatTp "dragonsnshit/server/combat"
)

// ── constants ──────────────────────────────────────────────────────────────────

const (
	tickRate       = time.Second
	respawnDelay   = 60 * time.Second
	defaultHP      = 500
	defaultMaxHP   = 500
	defaultMP      = 200
	defaultMaxMP   = 200
	playerDamage   = 30
	playerMeleeRng = 4.0

	mudPort = 2323

	// XP awarded per kill = mob.MaxHP * xpPerHP.
	xpPerHP       = 10
	// Range within which party members receive XP (same zone = always in range).
	partyXPRange  = 99999.0
)

// Zone adjacency: zoneID → direction → destination zoneID
var exits = map[int]map[string]int{
	0: {"north": 1, "south": 2, "east": 3},
	1: {"south": 0},
	2: {"north": 0},
	3: {"west": 0},
}

var dirAliases = map[string]string{
	"n": "north", "s": "south", "e": "east", "w": "west",
	"u": "up", "d": "down",
}

var zoneDesc = map[int]string{
	0: "A wide grassy meadow. The air smells of earth and dew. Worm burrows dot the ground.",
	1: "Rolling green hills stretch to the horizon. The wind is strong up here.",
	2: "A dark stone cave. Water drips somewhere in the darkness. Your footsteps echo.",
	3: "Swampville. Thick air, murky water, dead mangrove trees. Something slithers nearby.",
}

// ── world state ───────────────────────────────────────────────────────────────

type deadMob struct {
	m         mob.Mob
	respawnAt time.Time
	zoneID    int
}

// mobChainState records the last weapon skill that hit a mob (for skillchain detection).
type mobChainState struct {
	Attrs  []skillchain.Resonance
	At     time.Time
	Slot   string // who threw the WS
}

type world struct {
	mu             sync.Mutex
	zoneMgr        *zone.Manager
	mobRegs        map[int]*mob.Registry
	minePoints     map[int][]*gather.MiningPoint
	deadQueue      []deadMob
	players        map[string]*player // slot → player
	rng            *rand.Rand
	fieldManuals   map[int]*field.Manual // zoneID → active manual (nil if none)
	parties        map[string]*party.Party // partyID (leader slot) → party
	playerParty    map[string]string // slot → partyID
	xpChains       map[string]*party.XPChain // partyID → chain
	pendingInvites map[string]string // invitee slot → inviter slot
	mobChains      map[string]*mobChainState // mobID → last WS chain state
}

var gw *world

// ── player ────────────────────────────────────────────────────────────────────

type player struct {
	slot        string
	name        string
	zoneID      int
	pos         mob.Pos
	hp, maxHP   int
	mp, maxMP   int
	tp          *combatTp.TPState
	statFX      *status.Stack
	combat      *mob.PlayerCombat
	miningSkill float64
	charXP      *xp.CharXP
	homePoint   *homepoint.State
	wsSkill     string // current weapon skill name (from CanonicalWeaponSkills)
	conn        net.Conn
	w           *bufio.Writer
	inbox       chan string
}

func (p *player) send(msg string) {
	p.w.WriteString(msg + "\r\n")
	p.w.Flush()
}

func (p *player) sendf(f string, args ...interface{}) {
	p.send(fmt.Sprintf(f, args...))
}

func (p *player) prompt() {
	netHaste := p.statFX.NetHastePct()
	p.sendf("\r\n[ Lv.%d  HP:%d/%d  MP:%d/%d  TP:%d  Haste:%d%%  Zone:%s ]",
		p.charXP.Level, p.hp, p.maxHP, p.mp, p.maxMP, p.tp.Current,
		netHaste, zoneName(p.zoneID))
	p.sendf("> ")
}

func zoneName(id int) string {
	if z, ok := gw.zoneMgr.Get(id); ok {
		return z.Name
	}
	return fmt.Sprintf("Zone%d", id)
}

// ── game world init ───────────────────────────────────────────────────────────

func initWorld() *world {
	w := &world{
		mobRegs:        make(map[int]*mob.Registry),
		minePoints:     make(map[int][]*gather.MiningPoint),
		players:        make(map[string]*player),
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
		fieldManuals:   make(map[int]*field.Manual),
		parties:        make(map[string]*party.Party),
		playerParty:    make(map[string]string),
		xpChains:       make(map[string]*party.XPChain),
		pendingInvites: make(map[string]string),
		mobChains:      make(map[string]*mobChainState),
	}

	w.zoneMgr = zone.New(zone.DefaultZones())

	for _, zoneID := range []int{0, 1, 2, 3} {
		w.mobRegs[zoneID] = mob.New()
	}
	for _, m := range mob.MeadowWormSpawns() {
		_ = w.mobRegs[0].Spawn(m)
	}
	for _, m := range mob.SwampvilleSpawns() {
		_ = w.mobRegs[3].Spawn(m)
	}

	w.minePoints[0] = gather.MeadowMiningPoints()
	w.minePoints[3] = gather.SwampMiningPoints()

	return w
}

// ── XP helpers ────────────────────────────────────────────────────────────────

// awardXP grants xpAmount to p, applying field manual bonus for their zone.
// Must be called with gw.mu held.
func awardXP(p *player, baseXP int, now time.Time) {
	total := baseXP
	if m, ok := gw.fieldManuals[p.zoneID]; ok && m != nil {
		total = field.ApplyAll(baseXP, p.zoneID, now, []*field.Manual{m})
	}
	leveled, err := p.charXP.AddXP(total)
	p.homePoint.CurrentXP = p.charXP.CurrentXP
	if err == xp.ErrAtCap {
		p.sendf("  XP: +%d (Level cap reached — 99)", total)
		return
	}
	needed := p.charXP.XPToNextLevel()
	if total != baseXP {
		p.sendf("  XP: +%d (x2 field manual bonus!  Lv.%d  %d/%d to next)",
			total, p.charXP.Level, p.charXP.CurrentXP, needed)
	} else {
		p.sendf("  XP: +%d (Lv.%d  %d/%d to next)", total, p.charXP.Level, p.charXP.CurrentXP, needed)
	}
	if leveled {
		p.sendf("  >>> LEVEL UP! You are now level %d! <<<", p.charXP.Level)
		broadcastZoneNoLock(p.zoneID, fmt.Sprintf(">>> %s reaches level %d! <<<", p.name, p.charXP.Level), p.slot)
	}
}

// broadcastZoneNoLock sends msg to all players in zoneID except exceptSlot.
// Does NOT acquire gw.mu — only call when already holding the lock.
func broadcastZoneNoLock(zoneID int, msg, exceptSlot string) {
	for _, op := range gw.players {
		if op.zoneID == zoneID && op.slot != exceptSlot {
			op.send("\r\n" + msg)
			op.prompt()
		}
	}
}

// resolveKill handles mob kill: XP award (solo or party), chain bonus, respawn queuing.
// Must be called with gw.mu held.
func resolveKill(p *player, killedMob *mob.Mob, reg *mob.Registry, now time.Time) {
	baseXP := killedMob.MaxHP * xpPerHP

	partyID, inParty := gw.playerParty[p.slot]
	if inParty {
		pt, ok := gw.parties[partyID]
		if ok {
			chain := gw.xpChains[partyID]
			chainBonus := chain.Record(now.UnixNano())
			chainXP := baseXP + baseXP*chainBonus/100
			if chainBonus > 0 {
				p.sendf("  Chain bonus: +%d%%  (chain #%d)", chainBonus, chain.Count)
			}

			allMembers := pt.All()
			// Only members in the same zone share XP.
			dists := make([]float64, len(allMembers))
			for i, slot := range allMembers {
				if op, ok := gw.players[slot]; ok && op.zoneID == p.zoneID {
					dists[i] = 0 // in range
				} else {
					dists[i] = partyXPRange + 1 // out of range
				}
			}
			split := pt.XPSplit(chainXP, dists, partyXPRange)
			for i, slot := range allMembers {
				if split[i] > 0 {
					if op, ok := gw.players[slot]; ok {
						awardXP(op, split[i], now)
					}
				}
			}
		} else {
			awardXP(p, baseXP, now)
		}
	} else {
		// Solo XP chain
		_ = gw.xpChains // no chain for solo players in this implementation
		awardXP(p, baseXP, now)
	}

	// Queue mob respawn.
	gw.deadQueue = append(gw.deadQueue, deadMob{
		m:         *killedMob,
		respawnAt: now.Add(respawnDelay),
		zoneID:    p.zoneID,
	})
}

// knockOut marks p as KO'd and broadcasts the death message.
// Must be called with gw.mu held.
func knockOut(p *player) {
	if p.homePoint.IsKO {
		return // already KO'd
	}
	p.homePoint.IsKO = true
	p.combat.TargetMobID = ""
	p.send("\r\n** YOU HAVE BEEN KNOCKED OUT! **")
	if p.homePoint.HasHome() {
		p.send("Type 'home' to return to your Home Point (8% XP penalty).")
	} else {
		p.send("No Home Point set — use 'sethome' while alive to register one.")
		p.send("Type 'home' anyway to respawn at zone default (no penalty without home).")
	}
	broadcastZoneNoLock(p.zoneID, fmt.Sprintf("%s has been knocked out!", p.name), p.slot)
}

// ── game loop (1 Hz) ─────────────────────────────────────────────────────────

func gameLoop() {
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()
	for range ticker.C {
		gw.mu.Lock()
		tickAll()
		gw.mu.Unlock()
	}
}

func tickAll() {
	now := time.Now()
	dt := tickRate.Seconds()

	playerPosByZone := map[int][]mob.PlayerPositions{}
	for _, p := range gw.players {
		playerPosByZone[p.zoneID] = append(playerPosByZone[p.zoneID], mob.PlayerPositions{
			Slot:    p.slot,
			SceneID: p.zoneID,
			Pos:     p.pos,
		})
	}

	for zoneID, reg := range gw.mobRegs {
		events := reg.Tick(now, dt, playerPosByZone[zoneID])
		for _, ev := range events {
			broadcastMobEvent(zoneID, ev)
		}
	}

	for _, p := range gw.players {
		if p.homePoint.IsKO || p.combat.TargetMobID == "" {
			continue
		}
		reg := gw.mobRegs[p.zoneID]
		res, evts, err := reg.TickPlayer(p.slot, p.combat, p.pos, p.zoneID, now)
		if err != nil {
			if err == mob.ErrMobDead || err == mob.ErrMobNotFound {
				p.combat.TargetMobID = ""
				p.send("\r\n[Your target is gone.]")
				p.prompt()
			}
			continue
		}
		if res.Dealt > 0 {
			gained := p.tp.AddTP(combatTp.Delay1HSword, float64(p.statFX.NetHastePct()))
			p.sendf("\r\nYou hit for %d damage. (TP: %d [+%d])", res.Dealt, p.tp.Current, gained)
			if res.Died {
				p.send("The creature collapses!")
				deadMobID := p.combat.TargetMobID
				p.combat.TargetMobID = ""
				delete(gw.mobChains, deadMobID)
				var killedMob *mob.Mob
				if len(evts) > 0 {
					if m, ok := reg.Get(evts[len(evts)-1].MobID); ok {
						killedMob = m
					}
				}
				if killedMob != nil {
					resolveKill(p, killedMob, reg, now)
				}
			}
			p.prompt()
		}
		_ = evts
	}

	for _, p := range gw.players {
		if p.homePoint.IsKO {
			continue
		}
		res := p.statFX.Tick(now)
		for _, ev := range res.Events {
			switch ev.Target {
			case status.TargetHP:
				if ev.Value < 0 {
					p.hp += ev.Value
					if p.hp <= 0 {
						p.hp = 0
						knockOut(p)
					} else {
						p.sendf("\r\n[%s] You take %d damage. HP: %d/%d", ev.Kind, -ev.Value, p.hp, p.maxHP)
					}
				} else {
					p.hp += ev.Value
					if p.hp > p.maxHP {
						p.hp = p.maxHP
					}
					p.sendf("\r\n[%s] HP recovered. HP: %d/%d", ev.Kind, p.hp, p.maxHP)
				}
				p.prompt()
			case status.TargetMP:
				p.mp += ev.Value
				if p.mp > p.maxMP {
					p.mp = p.maxMP
				}
			}
		}
	}

	remaining := gw.deadQueue[:0]
	for _, d := range gw.deadQueue {
		if now.After(d.respawnAt) {
			m := d.m
			m.HP = m.MaxHP
			m.State = mob.StateIdle
			m.Pos = m.HomePos
			_ = gw.mobRegs[d.zoneID].Spawn(m)
		} else {
			remaining = append(remaining, d)
		}
	}
	gw.deadQueue = remaining
}

func broadcastMobEvent(zoneID int, ev mob.Event) {
	for _, p := range gw.players {
		if p.zoneID != zoneID {
			continue
		}
		switch ev.Kind {
		case mob.EvtMobAggro:
			if ev.Slot == p.slot {
				p.sendf("\r\n[!] %s turns toward you with malicious intent!", ev.MobID)
				p.prompt()
			}
		case mob.EvtMobAttack:
			if ev.Slot == p.slot {
				if p.homePoint.IsKO {
					continue
				}
				p.hp -= ev.Damage
				if p.hp <= 0 {
					p.hp = 0
					p.sendf("\r\n[!] %s hits you for %d damage!", ev.MobID, ev.Damage)
					knockOut(p)
				} else {
					p.sendf("\r\n[!] %s hits you for %d damage! HP: %d/%d", ev.MobID, ev.Damage, p.hp, p.maxHP)
					p.prompt()
				}
			}
		case mob.EvtMobBurrow:
			p.sendf("\r\n%s burrows underground.", ev.MobID)
			p.prompt()
		case mob.EvtMobSurface:
			p.sendf("\r\n%s surfaces from the ground!", ev.MobID)
			p.prompt()
		case mob.EvtMobReset:
			p.sendf("\r\n%s disengages and returns home.", ev.MobID)
			p.prompt()
		}
	}
}

// ── command handling ──────────────────────────────────────────────────────────

func handle(p *player, line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		p.prompt()
		return
	}
	parts := strings.Fields(line)
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	if full, ok := dirAliases[cmd]; ok {
		cmd = full
	}

	gw.mu.Lock()
	defer gw.mu.Unlock()

	switch cmd {
	case "look", "l":
		cmdLook(p)
	case "north", "south", "east", "west", "up", "down":
		cmdGo(p, cmd)
	case "go":
		if len(args) == 0 {
			p.send("Go where?")
			return
		}
		cmdGo(p, strings.ToLower(args[0]))
	case "attack", "a", "kill", "k":
		if p.homePoint.IsKO {
			p.send("You are KO'd. Type 'home' to return to your Home Point.")
			return
		}
		if len(args) == 0 {
			p.send("Attack what?")
			return
		}
		cmdAttack(p, strings.Join(args, " "))
	case "stop":
		p.combat.TargetMobID = ""
		p.send("You stop attacking.")
	case "ws", "weaponskill":
		if p.homePoint.IsKO {
			p.send("You are KO'd.")
			return
		}
		override := strings.Join(args, " ")
		cmdWS(p, override)
	case "setws":
		if len(args) == 0 {
			p.sendf("Current WS: %q  (use 'wslist' to see options, 'setws <name>' to change)", p.wsSkill)
			p.prompt()
			return
		}
		cmdSetWS(p, strings.Join(args, " "))
	case "wslist":
		cmdWSList(p)
	case "mine":
		if p.homePoint.IsKO {
			p.send("You are KO'd.")
			return
		}
		cmdMine(p)
	case "status", "st", "stats":
		cmdStatus(p)
	case "who":
		cmdWho(p)
	case "say", "'":
		msg := strings.Join(args, " ")
		if cmd == "'" {
			msg = strings.Join(parts[1:], " ")
		}
		cmdSay(p, msg)
	case "tell", "t":
		if len(args) < 2 {
			p.send("Usage: tell <name> <message>")
			return
		}
		cmdTell(p, args[0], strings.Join(args[1:], " "))
	case "map":
		cmdMap(p)
	case "mobs":
		cmdMobs(p)
	case "mine-points", "mp":
		cmdMinePoints(p)
	case "sethome":
		if p.homePoint.IsKO {
			p.send("Cannot set home while KO'd.")
			return
		}
		p.homePoint.SetHome(p.zoneID, p.pos)
		p.sendf("Home Point registered at %s.", zoneName(p.zoneID))
	case "home":
		cmdHome(p)
	case "read-manual", "rm":
		cmdReadManual(p)
	case "invite":
		if len(args) == 0 {
			p.send("Usage: invite <player-name>")
			return
		}
		cmdInvite(p, args[0])
	case "accept":
		cmdAccept(p)
	case "party", "pt":
		cmdParty(p)
	case "leave-party", "lp":
		cmdLeaveParty(p)
	case "help", "?":
		cmdHelp(p)
	case "quit", "exit", "q":
		p.send("Farewell, adventurer.")
		p.conn.Close()
	default:
		p.sendf("Unknown command: %q — type HELP for a list.", cmd)
	}
}

func cmdLook(p *player) {
	z, _ := gw.zoneMgr.Get(p.zoneID)
	p.sendf("\r\n=== %s ===", z.Name)
	p.send(zoneDesc[p.zoneID])

	// Field manual active?
	if m, ok := gw.fieldManuals[p.zoneID]; ok && m != nil && m.Active(time.Now()) {
		p.sendf("  [!] %s active — XP +%d%%", m.Name, m.BonusPct)
	}

	ex := exits[p.zoneID]
	if len(ex) > 0 {
		dirs := make([]string, 0, len(ex))
		for d := range ex {
			dirs = append(dirs, d)
		}
		p.sendf("Exits: %s", strings.Join(dirs, ", "))
	}

	reg := gw.mobRegs[p.zoneID]
	mobList := reg.All()
	if len(mobList) > 0 {
		p.send("Creatures here:")
		for _, id := range mobList {
			m, _ := reg.Get(id)
			stateStr := ""
			if m.State == mob.StateBurrowed {
				stateStr = " (burrowed)"
			} else if m.State == mob.StatePursuing {
				stateStr = " (!)"
			}
			p.sendf("  [%s] %s  HP:%d/%d%s", m.ID, m.Kind, m.HP, m.MaxHP, stateStr)
		}
	} else {
		p.send("No creatures in sight.")
	}

	for _, op := range gw.players {
		if op.slot != p.slot && op.zoneID == p.zoneID {
			p.sendf("  %s is here.", op.name)
		}
	}
	p.prompt()
}

func cmdGo(p *player, dir string) {
	if p.homePoint.IsKO {
		p.send("You are KO'd — type 'home' to return to your Home Point.")
		return
	}
	ex, ok := exits[p.zoneID]
	if !ok {
		p.send("No exits.")
		p.prompt()
		return
	}
	dest, ok := ex[dir]
	if !ok {
		p.sendf("You can't go %s from here.", dir)
		p.prompt()
		return
	}

	p.combat.TargetMobID = ""
	_ = gw.zoneMgr.Transfer(p.slot, dest)
	p.zoneID = dest

	destZone, _ := gw.zoneMgr.Get(dest)
	p.pos = mob.Pos{X: destZone.SpawnX, Y: destZone.SpawnY, Z: destZone.SpawnZ}

	broadcastZoneNoLock(p.zoneID, fmt.Sprintf("%s arrives.", p.name), p.slot)
	cmdLook(p)
}

func cmdAttack(p *player, target string) {
	reg := gw.mobRegs[p.zoneID]
	var found *mob.Mob
	for _, id := range reg.All() {
		m, _ := reg.Get(id)
		if strings.HasPrefix(strings.ToLower(id), strings.ToLower(target)) ||
			strings.HasPrefix(strings.ToLower(m.Kind), strings.ToLower(target)) {
			found = m
			break
		}
	}
	if found == nil {
		p.sendf("No mob matching %q here.", target)
		p.prompt()
		return
	}
	if found.State == mob.StateBurrowed {
		p.send("Your target is underground — wait for it to surface.")
		p.prompt()
		return
	}
	p.combat.TargetMobID = found.ID
	p.sendf("You target %s (%s). Auto-attacking.", found.ID, found.Kind)
	p.prompt()
}

func cmdWS(p *player, overrideName string) {
	if !p.tp.CanWeaponSkill() {
		p.sendf("Not enough TP (%d/100).", p.tp.Current)
		p.prompt()
		return
	}
	if p.combat.TargetMobID == "" {
		p.send("No target.")
		p.prompt()
		return
	}

	wsName := p.wsSkill
	if overrideName != "" {
		wsName = overrideName
	}
	ws, ok := skillchain.CanonicalWeaponSkills[wsName]
	if !ok {
		p.sendf("Unknown weapon skill %q. Use 'wslist' to see available skills.", wsName)
		p.prompt()
		return
	}

	reg := gw.mobRegs[p.zoneID]
	baseDamage := playerDamage * 3
	mobID := p.combat.TargetMobID

	// Skillchain detection: check if prior WS on this mob is within the chain window.
	chainBonus := 0
	chainName := ""
	now := time.Now()
	if prev, exists := gw.mobChains[mobID]; exists && prev.Slot != p.slot {
		elapsed := now.Sub(prev.At)
		if result, formed := skillchain.Chain(prev.Attrs, ws.Attrs, elapsed, skillchain.DefaultChainWindow); formed {
			chainBonus = int(float64(baseDamage) * result.Multiplier)
			chainName = result.Resonance.String()
		}
	}

	totalDamage := baseDamage + chainBonus
	res, evts, err := reg.Hit(mobID, p.slot, totalDamage)
	_ = evts
	if err != nil {
		p.sendf("WS failed: %v", err)
		p.prompt()
		return
	}
	p.tp.UseWeaponSkill()

	// Announce WS + chain.
	if chainName != "" {
		p.sendf(">>> %s <<< %d damage — SKILLCHAIN: %s! (+%d bonus)", ws.Name, res.Dealt, strings.ToUpper(chainName), chainBonus)
		broadcastZoneNoLock(p.zoneID,
			fmt.Sprintf(">>> SKILLCHAIN: %s! <<< %s → %s", strings.ToUpper(chainName), p.name, ws.Name), p.slot)
	} else {
		p.sendf(">>> %s <<< You unleash your weapon skill for %d damage!", ws.Name, res.Dealt)
	}

	// Update mob chain state.
	gw.mobChains[mobID] = &mobChainState{Attrs: ws.Attrs, At: now, Slot: p.slot}

	if res.Died {
		p.send("The creature collapses!")
		delete(gw.mobChains, mobID)
		wsTarget := mobID
		p.combat.TargetMobID = ""
		if m, ok := reg.Get(wsTarget); ok {
			resolveKill(p, m, reg, now)
		}
	}
	p.prompt()
}

func cmdSetWS(p *player, name string) {
	if _, ok := skillchain.CanonicalWeaponSkills[name]; !ok {
		// Try case-insensitive match.
		for wsName := range skillchain.CanonicalWeaponSkills {
			if strings.EqualFold(wsName, name) {
				name = wsName
				goto found
			}
		}
		p.sendf("Unknown weapon skill %q. Use 'wslist' to see available skills.", name)
		p.prompt()
		return
	}
found:
	p.wsSkill = name
	ws := skillchain.CanonicalWeaponSkills[name]
	attrs := make([]string, len(ws.Attrs))
	for i, a := range ws.Attrs {
		attrs[i] = a.String()
	}
	p.sendf("Weapon skill set to %q  [%s]", name, strings.Join(attrs, ", "))
	p.prompt()
}

func cmdWSList(p *player) {
	p.send("\r\n=== Available Weapon Skills ===")
	names := make([]string, 0, len(skillchain.CanonicalWeaponSkills))
	for n := range skillchain.CanonicalWeaponSkills {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		ws := skillchain.CanonicalWeaponSkills[n]
		attrs := make([]string, len(ws.Attrs))
		for i, a := range ws.Attrs {
			attrs[i] = a.String()
		}
		cur := ""
		if n == p.wsSkill {
			cur = " <--"
		}
		p.sendf("  %-20s  [%s]%s", n, strings.Join(attrs, ", "), cur)
	}
	p.send("Use 'setws <name>' to change your weapon skill.")
	p.prompt()
}

func cmdMine(p *player) {
	points, ok := gw.minePoints[p.zoneID]
	if !ok || len(points) == 0 {
		p.send("There are no mining points in this zone.")
		p.prompt()
		return
	}
	var pt *gather.MiningPoint
	for _, mp := range points {
		if !mp.Depleted() {
			pt = mp
			break
		}
	}
	if pt == nil {
		p.send("All mining points here are depleted. Wait for them to respawn.")
		p.prompt()
		return
	}
	y, err := pt.Mine(p.miningSkill)
	if err != nil {
		p.sendf("Mining failed: %v", err)
		p.prompt()
		return
	}
	if y.ItemID == "" {
		p.sendf("You swing your pickaxe at %s but find nothing.", pt.Name)
	} else if y.HQ {
		p.sendf(">>> HQ! <<< You excavate %s from %s! (x%d)", y.ItemName, pt.Name, y.Qty)
		p.miningSkill += 0.5
	} else {
		p.sendf("You mine %s from %s.", y.ItemName, pt.Name)
		p.miningSkill += 0.1
	}
	if p.miningSkill > gather.SkillCap {
		p.miningSkill = gather.SkillCap
	}
	p.sendf("  (Mining skill: %.1f)", p.miningSkill)
	p.prompt()
}

func cmdStatus(p *player) {
	koStr := ""
	if p.homePoint.IsKO {
		koStr = "  ** KNOCKED OUT — type 'home' to return **\r\n"
	}
	needed := p.charXP.XPToNextLevel()
	xpStr := fmt.Sprintf("%d/%d", p.charXP.CurrentXP, needed)
	if p.charXP.Level >= xp.MaxLevel {
		xpStr = "MAX"
	}
	p.sendf("\r\n=== %s ===", p.name)
	if koStr != "" {
		p.send(koStr)
	}
	p.sendf("  Level:        %d  (%s XP)", p.charXP.Level, xpStr)
	p.sendf("  Zone:         %s", zoneName(p.zoneID))
	p.sendf("  HP:           %d / %d", p.hp, p.maxHP)
	p.sendf("  MP:           %d / %d", p.mp, p.maxMP)
	ws := skillchain.CanonicalWeaponSkills[p.wsSkill]
	wsAttrs := make([]string, len(ws.Attrs))
	for i, a := range ws.Attrs {
		wsAttrs[i] = a.String()
	}
	p.sendf("  TP:           %d / 300  (WS at 100 — %s [%s])", p.tp.Current, p.wsSkill, strings.Join(wsAttrs, ", "))
	p.sendf("  Mining skill: %.1f / %.0f", p.miningSkill, gather.SkillCap)
	p.sendf("  Haste:        %d%%", p.statFX.NetHastePct())
	if p.statFX.IsParalyzed() {
		p.send("  ** PARALYZED **")
	}
	if p.statFX.IsSilenced() {
		p.send("  ** SILENCED **")
	}
	if p.statFX.IsBound() {
		p.send("  ** BOUND **")
	}
	if p.homePoint.HasHome() {
		p.sendf("  Home Point:   %s", zoneName(p.homePoint.Home.SceneID))
	} else {
		p.send("  Home Point:   not set (use 'sethome')")
	}
	if p.combat.TargetMobID != "" {
		p.sendf("  Target:       %s", p.combat.TargetMobID)
	}
	// Party
	if pid, ok := gw.playerParty[p.slot]; ok {
		if pt, ok := gw.parties[pid]; ok {
			chain := gw.xpChains[pid]
			p.sendf("  Party:        %d/%d members  (chain: %d, +%d%% XP)",
				pt.Size(), party.MaxPartySize, chain.Count, chain.Count*party.ChainBonusPerKill)
		}
	}
	if m, ok := gw.fieldManuals[p.zoneID]; ok && m != nil && m.Active(time.Now()) {
		p.sendf("  Field Manual: %s (+%d%% XP active)", m.Name, m.BonusPct)
	}
	p.prompt()
}

func cmdWho(p *player) {
	p.sendf("\r\n=== Online Players (%d) ===", len(gw.players))
	for _, op := range gw.players {
		partyStr := ""
		if _, ok := gw.playerParty[op.slot]; ok {
			partyStr = " [P]"
		}
		koStr := ""
		if op.homePoint.IsKO {
			koStr = " [KO]"
		}
		p.sendf("  Lv.%-3d %-16s  %s%s%s", op.charXP.Level, op.name, zoneName(op.zoneID), partyStr, koStr)
	}
	p.prompt()
}

func cmdSay(p *player, msg string) {
	if msg == "" {
		p.send("Say what?")
		p.prompt()
		return
	}
	full := fmt.Sprintf("%s says: %s", p.name, msg)
	broadcastZone(p.zoneID, full, "")
}

func cmdTell(p *player, target, msg string) {
	var dest *player
	for _, op := range gw.players {
		if strings.EqualFold(op.name, target) {
			dest = op
			break
		}
	}
	if dest == nil {
		p.sendf("No player named %q online.", target)
		p.prompt()
		return
	}
	dest.sendf("[Tell from %s]: %s", p.name, msg)
	p.sendf("[Tell to %s]: %s", dest.name, msg)
	p.prompt()
}

func cmdMap(p *player) {
	p.sendf("\r\n=== World Map ===")
	zones := gw.zoneMgr.ZoneIDs()
	for _, id := range zones {
		z, _ := gw.zoneMgr.Get(id)
		count := gw.zoneMgr.PlayersInCount(id)
		mobCount := len(gw.mobRegs[id].All())
		marker := "  "
		if id == p.zoneID {
			marker = "->"
		}
		manualStr := ""
		if m, ok := gw.fieldManuals[id]; ok && m != nil && m.Active(time.Now()) {
			manualStr = fmt.Sprintf(" [+%d%%XP]", m.BonusPct)
		}
		p.sendf("%s [%d] %-12s  %d mob(s)  %d player(s)%s", marker, id, z.Name, mobCount, count, manualStr)
		if ex, ok := exits[id]; ok {
			for dir, dest := range ex {
				dz, _ := gw.zoneMgr.Get(dest)
				p.sendf("       %s → %s", dir, dz.Name)
			}
		}
	}
	p.prompt()
}

func cmdMobs(p *player) {
	reg := gw.mobRegs[p.zoneID]
	ids := reg.All()
	if len(ids) == 0 {
		p.send("No mobs in this zone.")
		p.prompt()
		return
	}
	p.sendf("\r\n=== Mobs in %s ===", zoneName(p.zoneID))
	for _, id := range ids {
		m, _ := reg.Get(id)
		p.sendf("  %-22s  kind=%-8s  hp=%d/%d  state=%s  XP=%d",
			id, m.Kind, m.HP, m.MaxHP, m.State, m.MaxHP*xpPerHP)
	}
	p.prompt()
}

func cmdMinePoints(p *player) {
	pts := gw.minePoints[p.zoneID]
	if len(pts) == 0 {
		p.send("No mining points in this zone.")
		p.prompt()
		return
	}
	p.sendf("\r\n=== Mining Points in %s ===", zoneName(p.zoneID))
	for _, pt := range pts {
		dep := ""
		if pt.Depleted() {
			dep = "  [DEPLETED]"
		}
		p.sendf("  %-20s  diff=%.0f  remaining=%d/%d%s",
			pt.Name, pt.Difficulty, pt.Remaining(), pt.MaxAttempts, dep)
		p.sendf("    Success: %.0f%%  HQ: %.0f%%  (your skill: %.1f)",
			pt.SuccessChance(p.miningSkill), pt.HQChance(p.miningSkill), p.miningSkill)
	}
	p.prompt()
}

func cmdHome(p *player) {
	if !p.homePoint.IsKO {
		p.send("You are not KO'd. Use 'sethome' to register a Home Point.")
		p.prompt()
		return
	}
	if !p.homePoint.HasHome() {
		// No home set: respawn at zone 0 default with no penalty.
		p.homePoint.IsKO = false
		p.hp = 1
		oldZone := p.zoneID
		p.zoneID = 0
		p.combat.TargetMobID = ""
		z0, _ := gw.zoneMgr.Get(0)
		p.pos = mob.Pos{X: z0.SpawnX, Y: z0.SpawnY, Z: z0.SpawnZ}
		_ = gw.zoneMgr.Transfer(p.slot, 0)
		broadcastZoneNoLock(oldZone, fmt.Sprintf("%s fades from the zone.", p.name), p.slot)
		p.send("No Home Point set — you respawn at the Meadow (no penalty).")
		p.send("Use 'sethome' in a zone to register your Home Point.")
		cmdLook(p)
		return
	}

	// Compute XP penalty manually (8% of current XP in this level).
	penalty := p.charXP.CurrentXP * homepoint.DefaultXPPenaltyPct / 100
	if penalty < 0 {
		penalty = 0
	}
	p.charXP.CurrentXP -= penalty
	if p.charXP.CurrentXP < 0 {
		p.charXP.CurrentXP = 0
	}
	p.homePoint.CurrentXP = p.charXP.CurrentXP

	oldZone := p.zoneID
	p.homePoint.IsKO = false
	p.hp = 1
	p.combat.TargetMobID = ""
	crystal := p.homePoint.Home
	_ = gw.zoneMgr.Transfer(p.slot, crystal.SceneID)
	p.zoneID = crystal.SceneID
	p.pos = crystal.Pos

	broadcastZoneNoLock(oldZone, fmt.Sprintf("%s fades from the zone.", p.name), p.slot)
	broadcastZoneNoLock(p.zoneID, fmt.Sprintf("%s rises weakly at the Home Point crystal.", p.name), p.slot)
	p.sendf("You return to %s. (-%.0f%% XP penalty: -%d XP)",
		zoneName(p.zoneID), float64(homepoint.DefaultXPPenaltyPct), penalty)
	cmdLook(p)
}

func cmdReadManual(p *player) {
	if p.homePoint.IsKO {
		p.send("You are KO'd.")
		return
	}
	now := time.Now()
	var m *field.Manual
	switch p.zoneID {
	case 0:
		m = field.MeadowSurvivalGuide(now)
	case 3:
		m = field.SwampFieldManual(now)
	default:
		p.sendf("No field manual available in %s.", zoneName(p.zoneID))
		p.prompt()
		return
	}
	gw.fieldManuals[p.zoneID] = m
	broadcastZoneNoLock(p.zoneID,
		fmt.Sprintf("%s reads the %s. XP in this zone is +%d%% for 30 minutes!", p.name, m.Name, m.BonusPct), "")
	p.prompt()
}

func cmdInvite(p *player, targetName string) {
	var target *player
	for _, op := range gw.players {
		if strings.EqualFold(op.name, targetName) {
			target = op
			break
		}
	}
	if target == nil {
		p.sendf("No player named %q online.", targetName)
		p.prompt()
		return
	}
	if target.slot == p.slot {
		p.send("You can't invite yourself.")
		p.prompt()
		return
	}
	if _, ok := gw.playerParty[target.slot]; ok {
		p.sendf("%s is already in a party.", target.name)
		p.prompt()
		return
	}

	// Create party if not already leader of one.
	partyID, alreadyInParty := gw.playerParty[p.slot]
	if !alreadyInParty {
		pt := party.New(p.slot)
		partyID = p.slot
		gw.parties[partyID] = pt
		gw.playerParty[p.slot] = partyID
		gw.xpChains[partyID] = &party.XPChain{}
	}
	pt := gw.parties[partyID]
	if pt.Leader != p.slot {
		p.send("Only the party leader can invite.")
		p.prompt()
		return
	}
	if pt.Full() {
		p.send("Party is full (6 players max).")
		p.prompt()
		return
	}

	gw.pendingInvites[target.slot] = partyID
	target.sendf("\r\n[Party] %s invites you to join their party. Type 'accept' to join.", p.name)
	p.sendf("Party invite sent to %s.", target.name)
	p.prompt()
}

func cmdAccept(p *player) {
	partyID, ok := gw.pendingInvites[p.slot]
	if !ok {
		p.send("No pending party invitation.")
		p.prompt()
		return
	}
	delete(gw.pendingInvites, p.slot)
	pt, exists := gw.parties[partyID]
	if !exists {
		p.send("That party no longer exists.")
		p.prompt()
		return
	}
	if err := pt.Invite(pt.Leader, p.slot); err != nil {
		p.sendf("Could not join party: %v", err)
		p.prompt()
		return
	}
	gw.playerParty[p.slot] = partyID
	// Notify all party members.
	for _, slot := range pt.All() {
		if op, ok := gw.players[slot]; ok {
			op.sendf("\r\n[Party] %s joined the party. (%d/%d)", p.name, pt.Size(), party.MaxPartySize)
		}
	}
	p.prompt()
}

func cmdParty(p *player) {
	partyID, ok := gw.playerParty[p.slot]
	if !ok {
		p.send("You are not in a party.")
		p.prompt()
		return
	}
	pt := gw.parties[partyID]
	chain := gw.xpChains[partyID]
	p.sendf("\r\n=== Party (%d/%d) ===", pt.Size(), party.MaxPartySize)
	for _, slot := range pt.All() {
		leaderMark := ""
		if slot == pt.Leader {
			leaderMark = " [Leader]"
		}
		if op, ok := gw.players[slot]; ok {
			koStr := ""
			if op.homePoint.IsKO {
				koStr = " [KO]"
			}
			p.sendf("  Lv.%-3d %-16s  %s%s%s", op.charXP.Level, op.name, zoneName(op.zoneID), leaderMark, koStr)
		}
	}
	bonusPct := chain.Count * party.ChainBonusPerKill
	if bonusPct > party.ChainBonusCap {
		bonusPct = party.ChainBonusCap
	}
	p.sendf("  XP Chain: #%d  (+%d%% bonus)", chain.Count, bonusPct)
	p.prompt()
}

func cmdLeaveParty(p *player) {
	partyID, ok := gw.playerParty[p.slot]
	if !ok {
		p.send("You are not in a party.")
		p.prompt()
		return
	}
	pt := gw.parties[partyID]
	err := pt.Leave(p.slot)
	delete(gw.playerParty, p.slot)

	if err == party.ErrPartyEmpty || pt.Size() == 0 {
		// Disband.
		delete(gw.parties, partyID)
		delete(gw.xpChains, partyID)
		p.send("[Party] Party disbanded.")
	} else {
		// Notify remaining members.
		for _, slot := range pt.All() {
			if op, ok := gw.players[slot]; ok {
				op.sendf("\r\n[Party] %s left the party. (%d/%d)", p.name, pt.Size(), party.MaxPartySize)
			}
		}
		p.sendf("[Party] You left the party.")
	}
	p.prompt()
}

func cmdHelp(p *player) {
	p.send(`
Commands:
  look / l            — describe current zone, mobs, players
  n / s / e / w       — move to adjacent zone
  go <dir>            — same as direction shortcut
  attack <mob>        — target and auto-attack a mob (id or kind prefix)
  stop                — cease attacking
  ws [skillname]      — unleash weapon skill (requires TP >= 100); chains with others' WS
  setws <name>        — change your weapon skill (see 'wslist')
  wslist              — list all weapon skills and their SC resonances
  mine                — attempt mining at a nearby point
  mine-points / mp    — list mining points in this zone
  status / st         — show your stats (level, XP, homepoint, party)
  who                 — list online players
  say <text>  / ' msg — speak in zone
  tell <name> <text>  — private message
  map                 — world map with zone connections + XP bonuses
  mobs                — list all mobs in current zone (with XP value)
  sethome             — register current zone as your Home Point
  home                — return to Home Point when KO'd (8% XP penalty)
  read-manual / rm    — activate Field Manual in current zone (+100% XP 30min)
  invite <name>       — invite player to your party (you become leader)
  accept              — accept a pending party invitation
  party / pt          — show party members and XP chain
  leave-party / lp    — leave your current party
  help / ?            — this list
  quit                — disconnect`)
	p.prompt()
}

// ── broadcast helpers ─────────────────────────────────────────────────────────

func broadcastZone(zoneID int, msg, exceptSlot string) {
	gw.mu.Lock()
	defer gw.mu.Unlock()
	broadcastZoneNoLock(zoneID, msg, exceptSlot)
}

// ── connection handler ────────────────────────────────────────────────────────

func handleConn(conn net.Conn) {
	defer conn.Close()
	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)

	send := func(s string) {
		w.WriteString(s + "\r\n")
		w.Flush()
	}

	send("Welcome to DragonsNShit MUD.")
	send("Enter your character name: ")
	w.Flush()

	nameRaw, err := r.ReadString('\n')
	if err != nil {
		return
	}
	name := strings.TrimSpace(nameRaw)
	if len(name) < 2 || len(name) > 20 {
		send("Name must be 2–20 characters.")
		return
	}

	slot := conn.RemoteAddr().String()

	p := &player{
		slot:        slot,
		name:        name,
		zoneID:      0,
		pos:         mob.Pos{X: 0, Y: 2, Z: 0},
		hp:          defaultHP,
		maxHP:       defaultMaxHP,
		mp:          defaultMP,
		maxMP:       defaultMaxMP,
		tp:          &combatTp.TPState{},
		statFX:      status.New(),
		combat:      &mob.PlayerCombat{BaseDamage: playerDamage, MeleeRange: playerMeleeRng},
		miningSkill: 0,
		charXP:      xp.NewCharXP(),
		homePoint:   homepoint.NewState(0),
		wsSkill:     "Fast Blade",
		conn:        conn,
		w:           w,
	}

	gw.mu.Lock()
	gw.players[slot] = p
	_ = gw.zoneMgr.Enter(slot, name, 0)
	broadcastZoneNoLock(0, fmt.Sprintf("%s has entered the world.", name), slot)
	gw.mu.Unlock()

	defer func() {
		gw.mu.Lock()
		// Leave party if in one.
		if partyID, ok := gw.playerParty[slot]; ok {
			pt := gw.parties[partyID]
			_ = pt.Leave(slot)
			delete(gw.playerParty, slot)
			if pt.Size() == 0 {
				delete(gw.parties, partyID)
				delete(gw.xpChains, partyID)
			} else {
				for _, s := range pt.All() {
					if op, ok := gw.players[s]; ok {
						op.sendf("\r\n[Party] %s left the world. (%d/%d)", name, pt.Size(), party.MaxPartySize)
					}
				}
			}
		}
		delete(gw.pendingInvites, slot)
		delete(gw.players, slot)
		gw.zoneMgr.Leave(slot)
		broadcastZoneNoLock(p.zoneID, fmt.Sprintf("%s has left the world.", name), slot)
		gw.mu.Unlock()
	}()

	p.send("\r\nWelcome, " + name + "!")
	p.send("Type HELP for commands.")

	gw.mu.Lock()
	cmdLook(p)
	gw.mu.Unlock()

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		handle(p, line)
	}
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	port := flag.Int("port", mudPort, "MUD TCP port")
	flag.Parse()

	gw = initWorld()

	go gameLoop()

	addr := fmt.Sprintf(":%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(fmt.Sprintf("listen: %v", err))
	}
	fmt.Printf("DragonsNShit MUD listening on %s\n", addr)
	fmt.Printf("Connect: nc localhost %d\n", *port)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("accept: %v\n", err)
			continue
		}
		go handleConn(conn)
	}
}

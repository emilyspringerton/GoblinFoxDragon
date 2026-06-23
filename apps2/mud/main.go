// DragonsNShit MUD — text-based telnet interface integrating all server packages.
//
// Connect: nc localhost 2323   (or telnet localhost 2323)
// Port is configurable via -port flag.
//
// All server packages (mob, combat/tp, status, zone, gather/mining, market,
// chat, guild) are wired into a single game loop ticking at 1 Hz.
// Players interact via line-based text commands over TCP.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	"dragonsnshit/server/gather"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/status"
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

type world struct {
	mu           sync.Mutex
	zoneMgr      *zone.Manager
	mobRegs      map[int]*mob.Registry // zoneID → registry
	minePoints   map[int][]*gather.MiningPoint // zoneID → points
	deadQueue    []deadMob
	players      map[string]*player // slot → player
	rng          *rand.Rand
}

var gw *world

// ── player ────────────────────────────────────────────────────────────────────

type player struct {
	slot    string
	name    string
	zoneID  int
	pos     mob.Pos
	hp, maxHP int
	mp, maxMP int
	tp      *combatTp.TPState
	statFX  *status.Stack
	combat  *mob.PlayerCombat
	miningSkill float64
	conn    net.Conn
	w       *bufio.Writer
	inbox   chan string // events pushed from game loop
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
	p.sendf("\r\n[ HP:%d/%d  MP:%d/%d  TP:%d  Haste:%d%%  Zone:%s ]",
		p.hp, p.maxHP, p.mp, p.maxMP, p.tp.Current,
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
		mobRegs:    make(map[int]*mob.Registry),
		minePoints: make(map[int][]*gather.MiningPoint),
		players:    make(map[string]*player),
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	w.zoneMgr = zone.New(zone.DefaultZones())

	// Seed mob registries.
	for _, zoneID := range []int{0, 1, 2, 3} {
		w.mobRegs[zoneID] = mob.New()
	}
	for _, m := range mob.MeadowWormSpawns() {
		_ = w.mobRegs[0].Spawn(m)
	}
	for _, m := range mob.SwampvilleSpawns() {
		_ = w.mobRegs[3].Spawn(m)
	}

	// Mining points.
	w.minePoints[0] = gather.MeadowMiningPoints()
	w.minePoints[3] = gather.SwampMiningPoints()

	return w
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

	// Build player position snapshots per zone.
	playerPosByZone := map[int][]mob.PlayerPositions{}
	for _, p := range gw.players {
		playerPosByZone[p.zoneID] = append(playerPosByZone[p.zoneID], mob.PlayerPositions{
			Slot:    p.slot,
			SceneID: p.zoneID,
			Pos:     p.pos,
		})
	}

	// Tick mob AIs.
	for zoneID, reg := range gw.mobRegs {
		events := reg.Tick(now, dt, playerPosByZone[zoneID])
		for _, ev := range events {
			broadcastMobEvent(zoneID, ev)
		}
	}

	// Player combat ticks.
	for _, p := range gw.players {
		if p.combat.TargetMobID == "" {
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
			// TP gain on hit.
			gained := p.tp.AddTP(combatTp.Delay1HSword, float64(p.statFX.NetHastePct()))
			p.sendf("\r\nYou hit for %d damage. (TP: %d [+%d])", res.Dealt, p.tp.Current, gained)
			if res.Died {
				p.send("The creature collapses!")
				p.combat.TargetMobID = ""
				// Queue respawn.
				if m, ok := reg.Get(evts[len(evts)-1].MobID); ok {
					gw.deadQueue = append(gw.deadQueue, deadMob{
						m:         *m,
						respawnAt: now.Add(respawnDelay),
						zoneID:    p.zoneID,
					})
				}
				p.tp.AddTP(combatTp.Delay1HSword, float64(p.statFX.NetHastePct()))
			}
			p.prompt()
		}
		_ = evts
	}

	// Status effect ticks.
	for _, p := range gw.players {
		res := p.statFX.Tick(now)
		for _, ev := range res.Events {
			switch ev.Target {
			case status.TargetHP:
				if ev.Value < 0 {
					p.hp += ev.Value // negative = damage
					if p.hp < 0 {
						p.hp = 0
					}
					p.sendf("\r\n[%s] You take %d damage. HP: %d/%d", ev.Kind, -ev.Value, p.hp, p.maxHP)
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

	// Mob respawns.
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
				p.hp -= ev.Damage
				if p.hp < 0 {
					p.hp = 0
				}
				p.sendf("\r\n[!] %s hits you for %d damage! HP: %d/%d", ev.MobID, ev.Damage, p.hp, p.maxHP)
				p.prompt()
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

	// Expand direction aliases.
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
		if len(args) == 0 {
			p.send("Attack what?")
			return
		}
		cmdAttack(p, strings.Join(args, " "))
	case "stop":
		p.combat.TargetMobID = ""
		p.send("You stop attacking.")
	case "ws", "weaponskill":
		cmdWS(p)
	case "mine":
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

	// Exits.
	ex := exits[p.zoneID]
	if len(ex) > 0 {
		dirs := make([]string, 0, len(ex))
		for d := range ex {
			dirs = append(dirs, d)
		}
		p.sendf("Exits: %s", strings.Join(dirs, ", "))
	}

	// Mobs.
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

	// Other players.
	for _, op := range gw.players {
		if op.slot != p.slot && op.zoneID == p.zoneID {
			p.sendf("  %s is here.", op.name)
		}
	}
	p.prompt()
}

func cmdGo(p *player, dir string) {
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

	// Interrupt combat on zone transfer.
	p.combat.TargetMobID = ""

	// Telecrystal-style zone transfer.
	_ = gw.zoneMgr.Transfer(p.slot, dest)
	p.zoneID = dest

	destZone, _ := gw.zoneMgr.Get(dest)
	p.pos = mob.Pos{X: destZone.SpawnX, Y: destZone.SpawnY, Z: destZone.SpawnZ}

	// Notify zone mates.
	broadcastZone(p.zoneID, fmt.Sprintf("%s arrives.", p.name), p.slot)

	cmdLook(p)
}

func cmdAttack(p *player, target string) {
	reg := gw.mobRegs[p.zoneID]
	// Find mob by ID or kind prefix.
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

func cmdWS(p *player) {
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
	reg := gw.mobRegs[p.zoneID]
	damage := playerDamage * 3 // WS = 3× damage burst
	res, evts, err := reg.Hit(p.combat.TargetMobID, p.slot, damage)
	_ = evts
	if err != nil {
		p.sendf("WS failed: %v", err)
		p.prompt()
		return
	}
	p.tp.UseWeaponSkill()
	p.sendf(">>> FAST BLADE <<< You unleash your weapon skill for %d damage!", res.Dealt)
	if res.Died {
		p.send("The creature collapses!")
		p.combat.TargetMobID = ""
	}
	p.prompt()
}

func cmdMine(p *player) {
	points, ok := gw.minePoints[p.zoneID]
	if !ok || len(points) == 0 {
		p.send("There are no mining points in this zone.")
		p.prompt()
		return
	}
	// Find first non-depleted point.
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
	p.sendf("\r\n=== %s ===", p.name)
	p.sendf("  Zone:         %s", zoneName(p.zoneID))
	p.sendf("  HP:           %d / %d", p.hp, p.maxHP)
	p.sendf("  MP:           %d / %d", p.mp, p.maxMP)
	p.sendf("  TP:           %d / 300  (WS at 100)", p.tp.Current)
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
	if p.combat.TargetMobID != "" {
		p.sendf("  Target:       %s", p.combat.TargetMobID)
	}
	p.prompt()
}

func cmdWho(p *player) {
	p.sendf("\r\n=== Online Players (%d) ===", len(gw.players))
	for _, op := range gw.players {
		p.sendf("  %-16s  %s", op.name, zoneName(op.zoneID))
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
		p.sendf("%s [%d] %-12s  %d mob(s)  %d player(s)", marker, id, z.Name, mobCount, count)
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
		p.sendf("  %-22s  kind=%-8s  hp=%d/%d  state=%s", id, m.Kind, m.HP, m.MaxHP, m.State)
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

func cmdHelp(p *player) {
	p.send(`
Commands:
  look / l            — describe current zone, mobs, players
  n / s / e / w       — move to adjacent zone
  go <dir>            — same as direction shortcut
  attack <mob>        — target and auto-attack a mob (id or kind prefix)
  stop                — cease attacking
  ws / weaponskill    — unleash weapon skill (requires TP >= 100)
  mine                — attempt mining at a nearby point
  mine-points / mp    — list mining points in this zone
  status / st         — show your stats
  who                 — list online players
  say <text>  / ' msg — speak in zone
  tell <name> <text>  — private message
  map                 — world map with zone connections
  mobs                — list all mobs in current zone
  help / ?            — this list
  quit                — disconnect`)
	p.prompt()
}

// ── broadcast helpers ─────────────────────────────────────────────────────────

func broadcastZone(zoneID int, msg, exceptSlot string) {
	for _, op := range gw.players {
		if op.zoneID == zoneID && op.slot != exceptSlot {
			op.send("\r\n" + msg)
			op.prompt()
		}
	}
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

	// Login: ask name.
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
		conn:        conn,
		w:           w,
	}

	gw.mu.Lock()
	gw.players[slot] = p
	_ = gw.zoneMgr.Enter(slot, name, 0)
	broadcastZone(0, fmt.Sprintf("%s has entered the world.", name), slot)
	gw.mu.Unlock()

	defer func() {
		gw.mu.Lock()
		delete(gw.players, slot)
		gw.zoneMgr.Leave(slot)
		broadcastZone(p.zoneID, fmt.Sprintf("%s has left the world.", name), slot)
		gw.mu.Unlock()
	}()

	p.send("\r\nWelcome, " + name + "!")
	p.send("Type HELP for commands.")

	gw.mu.Lock()
	cmdLook(p)
	gw.mu.Unlock()

	// Read-eval loop.
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

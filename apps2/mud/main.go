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

	"dragonsnshit/server/chat"
	"dragonsnshit/server/conquest"
	"dragonsnshit/server/craft"
	"dragonsnshit/server/enmity"
	"dragonsnshit/server/gear"
	"dragonsnshit/server/guild"
	"dragonsnshit/server/merit"
	"dragonsnshit/server/telecrystal"
	"dragonsnshit/server/worldcrisis"
	"dragonsnshit/server/market"
	"dragonsnshit/server/field"
	"dragonsnshit/server/gather"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/job"
	"dragonsnshit/server/loot"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/nm"
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

// recipeIngredients maps recipe ID → { itemID: quantity required }.
// The craft package handles success/HQ, but ingredient requirements live here.
var recipeIngredients = map[string]map[string]int{
	"iron-ingot":    {"earth-crystal": 2, "worm-sinew": 1},
	"herbal-remedy": {"slime-oil": 1, "water-crystal": 1},
}

// itemDisplayName maps item IDs to human-readable names.
var itemDisplayName = map[string]string{
	"worm-sinew":       "Worm Sinew",
	"earth-crystal":    "Earth Crystal",
	"leech-blood":      "Leech Blood",
	"water-crystal":    "Water Crystal",
	"slime-oil":        "Slime Oil",
	"lizard-tail":      "Lizard Tail",
	"fire-crystal":     "Fire Crystal",
	"king-sinew":       "King Worm Sinew",
	"nm-worm-shell":    "Royal Worm Shell",
	"earth-crystal-hq": "Earth Crystal (HQ)",
	"marsh-blood":      "Marsh Leech Blood",
	"nm-leech-fang":    "Marsh Leech Fang",
	"water-crystal-hq": "Water Crystal (HQ)",
	"gil-drop":         "100 Gil",
	"crisis-shard":     "Crisis Shard",
	"iron-ingot":       "Iron Ingot",
	"iron-ingot+1":     "Iron Ingot +1",
	"iron-ingot+2":     "Iron Ingot +2",
	"iron-ingot+3":     "Iron Ingot +3",
	"herbal-remedy":    "Herbal Remedy",
	"herbal-remedy+1":  "Herbal Remedy +1",
	"herbal-remedy+2":  "Herbal Remedy +2",
	"herbal-remedy+3":  "Herbal Remedy +3",
}

func itemName(id string) string {
	if n, ok := itemDisplayName[id]; ok {
		return n
	}
	return id
}

// itemCategory maps item IDs to AH categories for listing.
var itemCategory = map[string]market.Category{
	"worm-sinew":    market.CatMaterials,
	"earth-crystal": market.CatCrystals,
	"leech-blood":   market.CatMaterials,
	"water-crystal": market.CatCrystals,
	"slime-oil":     market.CatMaterials,
	"lizard-tail":   market.CatMaterials,
	"fire-crystal":  market.CatCrystals,
	"king-sinew":    market.CatMaterials,
	"nm-worm-shell": market.CatMaterials,
	"marsh-blood":   market.CatMaterials,
	"nm-leech-fang": market.CatMaterials,
	"crisis-shard":  market.CatMaterials,
	"iron-ingot":    market.CatCraftItems,
	"iron-ingot+1":  market.CatCraftItems,
	"iron-ingot+2":  market.CatCraftItems,
	"iron-ingot+3":  market.CatCraftItems,
	"herbal-remedy":   market.CatCraftItems,
	"herbal-remedy+1": market.CatCraftItems,
	"herbal-remedy+2": market.CatCraftItems,
	"herbal-remedy+3": market.CatCraftItems,
}

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

// activeLootPool is a pending treasure pool after a mob kill.
type activeLootPool struct {
	pool   *loot.Pool
	poolID string
	zoneID int
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
	lootPools      map[string]*activeLootPool // poolID → pool
	nmSpawns       map[int][]*nm.NMSpawn // zoneID → NM spawn definitions
	conquestMap    *conquest.Map
	ah             *market.AuctionHouse
	playerNation   map[string]conquest.Nation // slot → declared nation
	lastConquestTick time.Time
	mobEnmity      map[string]*enmity.Table // mobID → enmity table
	chatRouter     *chat.Router
	guildReg       *guild.Registry
	wcrisis        *worldcrisis.Crisis
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
	jobID       string // current job (job.JobID, default "WAR")
	inventory   map[string]int // itemID → quantity
	craftSkill  *craft.CraftSkill
	gil         int
	guildID      string // linkshell guild ID ("" = none)
	equip        *gear.Equipment
	isInvisible  bool
	invisExpires time.Time
	isSneaking   bool
	sneakExpires time.Time
	charJob      *job.CharJob // main+sub job pairing (nil until initialized)
	meritBank    *merit.MeritBank
	recastTracker *job.RecastTracker
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
		lootPools:      make(map[string]*activeLootPool),
		playerNation:   make(map[string]conquest.Nation),
		lastConquestTick: time.Now(),
		mobEnmity:      make(map[string]*enmity.Table),
		chatRouter:     chat.New(),
		guildReg:       guild.New(),
		wcrisis:        worldcrisis.New(),
		nmSpawns: map[int][]*nm.NMSpawn{
			0: nm.MeadowNMs(),
			3: nm.SwampNMs(),
		},
	}

	w.zoneMgr = zone.New(zone.DefaultZones())

	w.conquestMap = conquest.NewMap()
	for _, r := range conquest.DefaultRegions() {
		w.conquestMap.AddRegion(r)
	}

	w.ah = market.New()

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
		// At level cap: XP converts to merit points.
		gained, merr := p.meritBank.Earn(total)
		if gained > 0 {
			p.sendf("  Merit: +%d point(s) (%d/%d banked)",
				gained, p.meritBank.Points, merit.MeritCap)
		} else if merr == merit.ErrMeritCapReached {
			p.sendf("  XP: +%d (Level cap 99 + merit cap reached)", total)
		} else {
			p.sendf("  XP: +%d (Level cap 99)", total)
		}
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
		applyJobStats(p)
		p.sendf("  >>> LEVEL UP! You are now level %d! (HP: %d/%d  MP: %d/%d) <<<",
			p.charXP.Level, p.hp, p.maxHP, p.mp, p.maxMP)
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

	// Loot pool.
	openLootPool(p, killedMob, now)

	// Clear enmity table for this mob.
	if et := gw.mobEnmity[killedMob.ID]; et != nil {
		et.Clear()
	}
	delete(gw.mobEnmity, killedMob.ID)

	// Conquest: award kill points to the player's declared nation.
	if nation := gw.playerNation[p.slot]; nation != conquest.NationNeutral {
		pts := 1 + killedMob.MaxHP/50
		_ = gw.conquestMap.AddPoints(p.zoneID, nation, pts)
	}

	// NM placeholder check.
	for _, spawn := range gw.nmSpawns[p.zoneID] {
		if spawn.PlaceholderID == killedMob.ID {
			spawn.OnPlaceholderKilled(now)
			broadcastZoneNoLock(p.zoneID,
				fmt.Sprintf("[!!!] The slaying of %s has disturbed something nearby...", killedMob.Kind), "")
		}
	}

	// World Crisis: NM kills (ID prefix "nm-") contribute to Intercept objective.
	if strings.HasPrefix(killedMob.ID, "nm-") {
		_ = gw.wcrisis.CompleteObjective(worldcrisis.ObjectiveIntercept, 10, now)
		for _, cp := range gw.players {
			cp.sendf("\r\n[Crisis] %s vanquished! Intercept objective advanced. (LEY +10)", killedMob.Kind)
			cp.prompt()
		}
		// Chaos Elementals drop crisis-shards.
		if killedMob.Kind == "Chaos Elemental" {
			p.inventory["crisis-shard"]++
			p.sendf("[Crisis] You obtain: Crisis Shard.")
		}
	}
}

// applyJobStats recomputes maxHP/maxMP for p based on their current job + level
// and caps current HP/MP to the new max. Must be called with gw.mu held (or at login).
func applyJobStats(p *player) {
	if hp, err := job.HPAtLevel(p.jobID, p.charXP.Level); err == nil {
		p.maxHP = hp
		if p.hp > p.maxHP {
			p.hp = p.maxHP
		}
	}
	if mp, err := job.MPAtLevel(p.jobID, p.charXP.Level); err == nil {
		p.maxMP = mp
		if mp == 0 {
			p.maxMP = 0 // melee job — no MP pool
		}
		if p.mp > p.maxMP {
			p.mp = p.maxMP
		}
	}
}

// dropsForMob returns the loot items a mob of the given kind drops on death.
func dropsForMob(kind string) []loot.Item {
	switch strings.ToLower(kind) {
	case "worm":
		return []loot.Item{
			{ID: "worm-sinew", Name: "Worm Sinew"},
			{ID: "earth-crystal", Name: "Earth Crystal"},
		}
	case "leech":
		return []loot.Item{
			{ID: "leech-blood", Name: "Leech Blood"},
			{ID: "water-crystal", Name: "Water Crystal"},
		}
	case "slime":
		return []loot.Item{{ID: "slime-oil", Name: "Slime Oil"}}
	case "lizard":
		return []loot.Item{
			{ID: "lizard-tail", Name: "Lizard Tail"},
			{ID: "fire-crystal", Name: "Fire Crystal"},
		}
	case "king worm":
		return []loot.Item{
			{ID: "king-sinew", Name: "King Worm Sinew"},
			{ID: "nm-worm-shell", Name: "Royal Worm Shell"},
			{ID: "earth-crystal-hq", Name: "Earth Crystal (HQ)"},
		}
	case "marsh leech":
		return []loot.Item{
			{ID: "marsh-blood", Name: "Marsh Leech Blood"},
			{ID: "nm-leech-fang", Name: "Marsh Leech Fang"},
			{ID: "water-crystal-hq", Name: "Water Crystal (HQ)"},
		}
	default:
		return []loot.Item{{ID: "gil-drop", Name: "100 Gil"}}
	}
}

// nmMobFor returns a Mob template for the given NM ID.
func nmMobFor(nmID string) mob.Mob {
	switch nmID {
	case "nm-king-worm":
		return mob.Mob{
			ID: nmID, Kind: "King Worm",
			HP: 800, MaxHP: 800,
			AggroRange: 15, LeashRange: 40, MeleeRange: 3,
			MoveSpeed: 4, MeleeDamage: 60, SwingDelay: 2 * time.Second,
		}
	case "nm-marsh-leech":
		return mob.Mob{
			ID: nmID, Kind: "Marsh Leech",
			HP: 600, MaxHP: 600,
			AggroRange: 12, LeashRange: 35, MeleeRange: 3,
			MoveSpeed: 5, MeleeDamage: 50, SwingDelay: 2 * time.Second,
		}
	default:
		return mob.Mob{
			ID: nmID, Kind: nmID,
			HP: 500, MaxHP: 500,
			AggroRange: 10, LeashRange: 30, MeleeRange: 3,
			MoveSpeed: 4, MeleeDamage: 40,
		}
	}
}

// openLootPool creates a loot pool after a mob kill. For solo kills, drops are auto-awarded.
// Must be called with gw.mu held.
func openLootPool(killer *player, m *mob.Mob, now time.Time) {
	drops := dropsForMob(m.Kind)
	if len(drops) == 0 {
		return
	}

	// Determine eligible players (party members in same zone, or solo).
	var eligible []string
	if partyID, inParty := gw.playerParty[killer.slot]; inParty {
		if pt, ok := gw.parties[partyID]; ok {
			for _, slot := range pt.All() {
				if op, ok := gw.players[slot]; ok && op.zoneID == killer.zoneID {
					eligible = append(eligible, slot)
				}
			}
		}
	}

	if len(eligible) == 0 {
		// Solo: auto-award all drops to the killer.
		for _, it := range drops {
			if it.ID == "gil-drop" {
				killer.gil += 100
				killer.sendf("  You obtain: 100 Gil.")
			} else {
				killer.inventory[it.ID]++
				killer.sendf("  You obtain: %s.", it.Name)
			}
		}
		return
	}

	// Party: open a pool.
	poolID := fmt.Sprintf("%s-%d", m.ID, now.UnixNano())
	pool := loot.NewPool(m.ID, eligible, drops, now.UnixNano())
	gw.lootPools[poolID] = &activeLootPool{pool: pool, poolID: poolID, zoneID: killer.zoneID}

	// Announce to eligible players.
	for _, slot := range eligible {
		if op, ok := gw.players[slot]; ok {
			op.sendf("\r\n[Loot] Treasure pool from %s:", m.Kind)
			for i, it := range drops {
				op.sendf("  [%d] %s", i+1, it.Name)
			}
			op.send("  Use 'lot N' to roll or 'pass N' (or 'pass all') to decline.")
		}
	}
}

// resolvePool announces awards and removes the pool when all players have acted.
// Must be called with gw.mu held.
func resolvePool(alp *activeLootPool) {
	if !alp.pool.Ready() {
		return
	}
	awards, err := alp.pool.Resolve()
	if err != nil {
		return
	}
	// Find item names.
	itemNames := make(map[string]string)
	for _, it := range alp.pool.Items {
		itemNames[it.ID] = it.Name
	}
	// Announce to zone.
	for _, award := range awards {
		name := itemNames[award.ItemID]
		if award.Slot == "" {
			broadcastZoneNoLock(alp.zoneID, fmt.Sprintf("[Loot] %s — no one claimed it.", name), "")
		} else if op, ok := gw.players[award.Slot]; ok {
			if award.ItemID == "gil-drop" {
				op.gil += 100
			} else {
				op.inventory[award.ItemID]++
			}
			broadcastZoneNoLock(alp.zoneID,
				fmt.Sprintf("[Loot] %s obtains %s! (lot: %d)", op.name, name, award.Roll), "")
		}
	}
	delete(gw.lootPools, alp.poolID)
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
			// Invisible/Sneak aggro block: cancel aggro on invisible/sneaking players.
			if ev.Kind == mob.EvtMobAggro {
				if pp, ok := gw.players[ev.Slot]; ok {
					if (pp.isInvisible && pp.invisExpires.After(now)) ||
						(pp.isSneaking && pp.sneakExpires.After(now)) {
						if m, ok2 := reg.Get(ev.MobID); ok2 && m.AggroSlot == ev.Slot {
							m.AggroSlot = ""
						}
						continue // suppress this aggro event
					}
				}
			}
			broadcastMobEvent(zoneID, ev)
		}
	}

	for _, p := range gw.players {
		// Expire Invisible/Sneak.
		if p.isInvisible && now.After(p.invisExpires) {
			p.isInvisible = false
			p.sendf("\r\n[Invisible fades.]")
			p.prompt()
		}
		if p.isSneaking && now.After(p.sneakExpires) {
			p.isSneaking = false
			p.sendf("\r\n[Sneak fades.]")
			p.prompt()
		}
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
			// Enmity: damage generates CE.
			mobID := p.combat.TargetMobID
			if mobID != "" {
				if gw.mobEnmity[mobID] == nil {
					gw.mobEnmity[mobID] = enmity.NewTable()
				}
				gw.mobEnmity[mobID].Add(p.slot, res.Dealt)
				// Switch mob AggroSlot to highest enmity player.
				if topSlot, err := gw.mobEnmity[mobID].Top(); err == nil {
					if m, ok := reg.Get(mobID); ok && m.AggroSlot != topSlot {
						m.AggroSlot = topSlot
					}
				}
			}
			if res.Died {
				p.send("The creature collapses!")
				deadMobID := p.combat.TargetMobID
				p.combat.TargetMobID = ""
				delete(gw.mobChains, deadMobID)
				// Clear enmity on death.
				if et := gw.mobEnmity[deadMobID]; et != nil {
					et.Clear()
				}
				delete(gw.mobEnmity, deadMobID)
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

	// Conquest tick: evaluate weekly control once per minute (MUD-compressed to 1 min).
	if now.Sub(gw.lastConquestTick) >= time.Minute {
		gw.lastConquestTick = now
		results := gw.conquestMap.TickAll()
		for regionID, newController := range results {
			if r, ok := gw.conquestMap.Get(regionID); ok {
				msg := fmt.Sprintf("[Conquest] %s is now controlled by %s.", r.Name, conquest.NationName(newController))
				for _, cp := range gw.players {
					cp.sendf("\r\n%s", msg)
					cp.prompt()
				}
			}
		}
	}

	// World Crisis tick.
	if gw.wcrisis.Status().Phase == worldcrisis.PhaseIdle && len(gw.players) > 0 {
		_ = gw.wcrisis.Start(now, "")
		for _, cp := range gw.players {
			cp.sendf("\r\n[World Crisis] Ley energies stir... a World Crisis is beginning!")
			cp.prompt()
		}
	}
	if gw.wcrisis.Status().Phase != worldcrisis.PhaseIdle {
		phaseChanged, oldPhase, newPhase := gw.wcrisis.Tick(now)
		if phaseChanged {
			phaseMsgs := map[worldcrisis.Phase]string{
				worldcrisis.PhaseOmens:       "[Crisis] Omens spread across the land...",
				worldcrisis.PhaseBurrow:       "[Crisis] The threat burrows deeper — prepare yourselves!",
				worldcrisis.PhaseEmergence:    "[Crisis] Combat window open — strike now! Chaos Elementals emerge in the Swamp!",
				worldcrisis.PhaseSplitWar:     "[Crisis] The enemy splits — two fronts detected!",
				worldcrisis.PhaseFinalWindow:  "[Crisis] Final Window! Complete objectives NOW!",
				worldcrisis.PhaseResolution:   "",
			}
			_ = oldPhase
			if msg, ok := phaseMsgs[newPhase]; ok && msg != "" {
				for _, cp := range gw.players {
					cp.sendf("\r\n\033[1;31m%s\033[0m", msg)
					cp.prompt()
				}
			}
			// Spawn crisis NMs in Swamp on Emergence.
			if newPhase == worldcrisis.PhaseEmergence {
				reg3 := gw.mobRegs[3]
				if reg3 != nil {
					for i := 1; i <= 3; i++ {
						nmID := fmt.Sprintf("nm-chaos-elemental-%d", i)
						spawnPos := mob.Pos{X: float64(-15 + i*12), Y: 0, Z: float64(-40 + i*10)}
					_ = reg3.Spawn(mob.Mob{
						ID: nmID, Kind: "Chaos Elemental",
						HP: 1200, MaxHP: 1200,
						SceneID:     3,
						AggroRange:  18, LeashRange: 45, MeleeRange: 3,
						MoveSpeed:   6, MeleeDamage: 80, SwingDelay: 2 * time.Second,
						Pos:     spawnPos,
						HomePos: spawnPos,
					})
					}
				}
			}
			if newPhase == worldcrisis.PhaseResolution {
				st := gw.wcrisis.Status()
				outcome := "FAILURE"
				if st.Outcome == worldcrisis.OutcomeVictory {
					outcome = "VICTORY"
				}
				for _, cp := range gw.players {
					cp.sendf("\r\n\033[1;33m[Crisis] World Crisis resolved: %s! LEY integrity: %d/%d\033[0m",
						outcome, st.LeyIntegrity, worldcrisis.LeyMax)
					cp.prompt()
				}
				// Auto-restart after resolution.
				gw.wcrisis = worldcrisis.New()
			}
		}
	}

	// NM spawn checks.
	for zoneID, spawns := range gw.nmSpawns {
		for _, spawn := range spawns {
			if spawn.WindowExpired(now) {
				spawn.Reset()
				continue
			}
			if spawn.InWindow(now) && spawn.TrySpawn(now, gw.rng) {
				nmMob := nmMobFor(spawn.ID)
				_ = gw.mobRegs[zoneID].Spawn(nmMob)
				for _, p := range gw.players {
					if p.zoneID == zoneID {
						p.sendf("\r\n[!!!] %s has appeared in %s!", nmMob.Kind, zoneName(zoneID))
						p.prompt()
					}
				}
			}
		}
	}
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
	case "yell", "y":
		cmdYell(p, strings.Join(args, " "))
	case "guild", "g":
		cmdGuildChat(p, strings.Join(args, " "))
	case "ls-create":
		if len(args) < 2 {
			p.send("Usage: ls-create <name> <tag>")
			return
		}
		cmdLSCreate(p, args[0], args[1])
	case "ls-invite":
		if len(args) < 1 {
			p.send("Usage: ls-invite <player-name>")
			return
		}
		cmdLSInvite(p, args[0])
	case "ls-leave":
		cmdLSLeave(p)
	case "ls-info":
		cmdLSInfo(p)
	case "equip":
		if len(args) < 2 {
			p.send("Usage: equip <slot> <item-id>")
			return
		}
		cmdEquip(p, args[0], args[1])
	case "unequip":
		if len(args) < 1 {
			p.send("Usage: unequip <slot>")
			return
		}
		cmdUnequip(p, args[0])
	case "gear":
		cmdGear(p)
	case "setsubjob", "ssj":
		if len(args) < 1 {
			p.send("Usage: setsubjob <JOB>")
			return
		}
		cmdSetSubJob(p, strings.ToUpper(args[0]))
	case "subjob", "sj":
		cmdSubJob(p)
	case "merits":
		cmdMerits(p)
	case "merit-spend", "ms":
		if len(args) < 1 {
			p.send("Usage: merit-spend <category>")
			return
		}
		cmdMeritSpend(p, args[0])
	case "crystals":
		cmdCrystals(p)
	case "travel":
		if len(args) < 1 {
			p.send("Usage: travel <crystal-id>")
			return
		}
		cmdTravel(p, args[0])
	case "cast":
		if len(args) < 1 {
			p.send("Usage: cast <spell>  (spells: invisible, sneak)")
			return
		}
		cmdCast(p, strings.ToLower(args[0]))
	case "ja":
		if len(args) < 1 {
			p.send("Usage: ja <ability-id>  (e.g. ja provoke)")
			return
		}
		cmdJA(p, strings.ToLower(args[0]))
	case "recasts", "recast":
		cmdRecasts(p)
	case "crisis":
		cmdCrisis(p)
	case "crisis-ley":
		if len(args) < 1 {
			p.send("Usage: crisis-ley <amount>")
			return
		}
		amount := 0
		_, _ = fmt.Sscan(args[0], &amount)
		gw.wcrisis.LeyDecay(amount)
		p.sendf("LEY decayed by %d. New integrity: %d", amount, gw.wcrisis.Status().LeyIntegrity)
		p.prompt()
	case "touch":
		cmdTouchCrystal(p)
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
	case "ah":
		cmdAH(p, args)
	case "enmity", "en":
		cmdEnmity(p)
	case "conquest", "con":
		cmdConquest(p)
	case "declare":
		if len(args) == 0 {
			p.send("Usage: declare <sandoria|bastok|windurst|neutral>")
			p.prompt()
			return
		}
		cmdDeclare(p, strings.ToLower(args[0]))
	case "inv", "inventory", "i":
		cmdInventory(p)
	case "craft":
		if len(args) == 0 {
			p.send("Usage: craft <recipe-id>  (see 'recipes')")
			p.prompt()
			return
		}
		cmdCraft(p, args[0])
	case "recipes":
		cmdRecipes(p)
	case "craft-skills", "cs":
		cmdCraftSkills(p)
	case "lot":
		if len(args) == 0 {
			p.send("Usage: lot <item-number>  (e.g. 'lot 1')")
			p.prompt()
			return
		}
		cmdLot(p, args[0])
	case "pass":
		if len(args) == 0 {
			p.send("Usage: pass <item-number> | pass all")
			p.prompt()
			return
		}
		cmdPass(p, args[0])
	case "pool", "loot":
		cmdPool(p)
	case "setjob":
		if len(args) == 0 {
			p.sendf("Current job: %s  (use 'jobs' to list all, 'setjob <JOB>' to change)", p.jobID)
			p.prompt()
			return
		}
		cmdSetJob(p, strings.ToUpper(args[0]))
	case "jobs":
		cmdJobs(p)
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
	syncChatSession(p)

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
	// Enmity: WS damage CE.
	if err == nil && res.Dealt > 0 {
		if gw.mobEnmity[mobID] == nil {
			gw.mobEnmity[mobID] = enmity.NewTable()
		}
		gw.mobEnmity[mobID].Add(p.slot, res.Dealt)
		if topSlot, terr := gw.mobEnmity[mobID].Top(); terr == nil {
			if m, ok := reg.Get(mobID); ok && m.AggroSlot != topSlot {
				m.AggroSlot = topSlot
			}
		}
	}
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
		p.inventory[y.ItemID] += y.Qty
		p.miningSkill += 0.5
	} else {
		p.sendf("You mine %s from %s.", y.ItemName, pt.Name)
		p.inventory[y.ItemID] += y.Qty
		p.miningSkill += 0.1
	}
	if p.miningSkill > gather.SkillCap {
		p.miningSkill = gather.SkillCap
	}
	p.sendf("  (Mining skill: %.1f)", p.miningSkill)
	// Crisis: gathering contributes to Ritual objective.
	if y.ItemID != "" {
		if err2 := gw.wcrisis.CompleteObjective(worldcrisis.ObjectiveRitual, 5, time.Now()); err2 == nil {
			p.sendf("[Crisis] Gathering stabilizes the ley lines. (Ritual +5 LEY)")
		}
	}
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
	p.sendf("\r\n=== %s [%s] ===", p.name, p.jobID)
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

// syncChatSession updates the chat router with the player's current position/zone/guild.
func syncChatSession(p *player) {
	gw.chatRouter.Register(p.slot, chat.Session{
		Name:    p.name,
		SceneID: p.zoneID,
		GuildID: p.guildID,
		Pos:     chat.Pos{X: p.pos.X, Y: p.pos.Y, Z: float64(p.zoneID) * 1000},
	})
}

// deliverChat routes a chat message and writes to each recipient.
func deliverChat(fromSlot string, channel int, target, msg string) {
	deliveries := gw.chatRouter.Deliver(fromSlot, channel, target, msg, 50.0)
	for _, d := range deliveries {
		ch, _, body, ok := chat.ParseChatPacket(d.Packet)
		if !ok {
			continue
		}
		op, found := gw.players[d.To]
		if !found {
			continue
		}
		prefix := ""
		switch ch {
		case chat.ChatSay:
			prefix = fmt.Sprintf("[Say] %s: ", gw.players[fromSlot].name)
		case chat.ChatTell:
			prefix = fmt.Sprintf("[Tell → %s]: ", gw.players[fromSlot].name)
		case chat.ChatYell:
			prefix = fmt.Sprintf("[Yell] %s: ", gw.players[fromSlot].name)
		case chat.ChatGuild:
			prefix = fmt.Sprintf("[LS] %s: ", gw.players[fromSlot].name)
		}
		op.sendf("\r\n%s%s", prefix, body)
		op.prompt()
	}
}

func cmdSay(p *player, msg string) {
	if msg == "" {
		p.send("Say what?")
		p.prompt()
		return
	}
	syncChatSession(p)
	deliverChat(p.slot, chat.ChatSay, "", msg)
	p.prompt()
}

func cmdTell(p *player, target, msg string) {
	syncChatSession(p)
	deliverChat(p.slot, chat.ChatTell, target, msg)
	p.prompt()
}

func cmdYell(p *player, msg string) {
	if msg == "" {
		p.send("Yell what?")
		p.prompt()
		return
	}
	syncChatSession(p)
	deliverChat(p.slot, chat.ChatYell, "", msg)
	p.prompt()
}

func cmdGuildChat(p *player, msg string) {
	if msg == "" {
		p.send("Guild chat what?")
		p.prompt()
		return
	}
	if p.guildID == "" {
		p.send("You are not in a linkshell.")
		p.prompt()
		return
	}
	if !gw.guildReg.CanChat(p.guildID, p.slot) {
		p.send("Your linkshell feather has been revoked.")
		p.prompt()
		return
	}
	syncChatSession(p)
	deliverChat(p.slot, chat.ChatGuild, "", msg)
	p.prompt()
}

func cmdLSCreate(p *player, name, tag string) {
	if p.guildID != "" {
		p.sendf("You are already in linkshell [%s]. Leave first.", p.guildID)
		p.prompt()
		return
	}
	guildID := strings.ToLower(strings.ReplaceAll(name, " ", "-")) + "-" + p.slot[:4]
	_, _, err := gw.guildReg.CreateGuild(guildID, name, tag, p.slot)
	if err != nil {
		p.sendf("Failed to create linkshell: %v", err)
		p.prompt()
		return
	}
	p.guildID = guildID
	syncChatSession(p)
	p.sendf("Linkshell [%s] <%s> founded. You hold the Feather Sack.", name, tag)
	p.prompt()
}

func cmdLSInvite(p *player, targetName string) {
	if p.guildID == "" {
		p.send("You are not in a linkshell.")
		p.prompt()
		return
	}
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
	if target.guildID != "" {
		p.sendf("%s is already in a linkshell.", target.name)
		p.prompt()
		return
	}
	_, err := gw.guildReg.ForgeFeather(p.guildID, p.slot, target.slot)
	if err != nil {
		p.sendf("Cannot invite: %v", err)
		p.prompt()
		return
	}
	target.guildID = p.guildID
	syncChatSession(target)
	target.sendf("[Linkshell] %s has invited you to join. You now hold a Feather.", p.name)
	p.sendf("[Linkshell] %s has joined.", target.name)
	p.prompt()
}

func cmdLSLeave(p *player) {
	if p.guildID == "" {
		p.send("You are not in a linkshell.")
		p.prompt()
		return
	}
	g, err := gw.guildReg.GetGuild(p.guildID)
	if err != nil {
		p.send("Linkshell not found.")
		p.guildID = ""
		p.prompt()
		return
	}
	// Find and revoke own item via leader/self — simplest: clear membership directly.
	// The guild Registry does not expose a self-leave API; use RevokeItem as leader, or
	// just drop the player from the guild's internal map via a dedicated approach.
	// Since RevokeItem requires an Officer+ issuer who is NOT the target, a player leaving
	// voluntarily needs a special path. We implement leave by finding an officer/leader to
	// auto-revoke, falling back to just clearing our local state if we're the last member.
	members := g.Members()
	var issuerSlot string
	for _, m := range members {
		if m.CharacterID != p.slot && m.Role >= 2 { // RoleOfficer=2, RoleLeader=3
			issuerSlot = m.CharacterID
			break
		}
	}
	lsName := g.Name
	if issuerSlot != "" {
		// Find our feather item ID
		for _, m := range members {
			if m.CharacterID == p.slot {
				_ = gw.guildReg.RevokeItem(p.guildID, issuerSlot, m.FeatherID)
				break
			}
		}
	}
	// Regardless, clear local guildID.
	p.guildID = ""
	syncChatSession(p)
	p.sendf("You have left linkshell [%s].", lsName)
	p.prompt()
}

func cmdLSInfo(p *player) {
	if p.guildID == "" {
		p.send("You are not in a linkshell.")
		p.prompt()
		return
	}
	g, err := gw.guildReg.GetGuild(p.guildID)
	if err != nil {
		p.send("Linkshell not found.")
		p.prompt()
		return
	}
	p.sendf("\r\n=== Linkshell [%s] <%s> ===", g.Name, g.Tag)
	members := g.Members()
	roleLabel := []string{"", "Member", "Officer", "Leader"}
	for _, m := range members {
		online := ""
		if _, ok := gw.players[m.CharacterID]; ok {
			online = " (online)"
		}
		label := "Member"
		if int(m.Role) < len(roleLabel) {
			label = roleLabel[m.Role]
		}
		p.sendf("  %s — %s%s", m.CharacterID, label, online)
	}
	p.prompt()
}

// itemIL maps known equippable item IDs to their item level.
var itemIL = map[string]int{
	"bronze-sword":    1,
	"iron-sword":      10,
	"bronze-shield":   1,
	"iron-shield":     10,
	"leather-helm":    5,
	"leather-body":    5,
	"leather-legs":    5,
	"leather-feet":    5,
	"leather-hands":   5,
	"bone-earring":    3,
	"iron-earring":    8,
	"cotton-cape":     4,
	"leather-belt":    4,
	"bronze-ring":     2,
}

func cmdEquip(p *player, slotName, itemID string) {
	// Validate slot.
	validSlot := false
	for _, s := range gear.AllSlots {
		if s == slotName {
			validSlot = true
			break
		}
	}
	if !validSlot {
		p.sendf("Unknown slot %q. Valid slots: %s", slotName, strings.Join(gear.AllSlots, ", "))
		p.prompt()
		return
	}
	// Must have item in inventory.
	if p.inventory[itemID] <= 0 {
		p.sendf("You don't have %s in your inventory.", itemDisplayName[itemID])
		p.prompt()
		return
	}
	il, ok := itemIL[itemID]
	if !ok {
		il = 1
	}
	// Move old item back to inventory if slot occupied.
	if old, err := p.equip.Unequip(slotName); err == nil {
		p.inventory[old.ItemID]++
	}
	_ = p.equip.Equip(slotName, gear.ItemEntry{ItemID: itemID, IL: il})
	p.inventory[itemID]--
	if p.inventory[itemID] == 0 {
		delete(p.inventory, itemID)
	}
	dispName := itemID
	if dn, ok := itemDisplayName[itemID]; ok {
		dispName = dn
	}
	p.sendf("Equipped %s in %s slot.", dispName, slotName)
	if eIL, err := p.equip.EffectiveIL(); err == nil {
		p.sendf("  Effective IL: %d", eIL)
	}
	p.prompt()
}

func cmdUnequip(p *player, slotName string) {
	item, err := p.equip.Unequip(slotName)
	if err != nil {
		p.sendf("Cannot unequip %s: %v", slotName, err)
		p.prompt()
		return
	}
	p.inventory[item.ItemID]++
	dispName := item.ItemID
	if dn, ok := itemDisplayName[item.ItemID]; ok {
		dispName = dn
	}
	p.sendf("Unequipped %s from %s.", dispName, slotName)
	p.prompt()
}

func cmdGear(p *player) {
	p.sendf("\r\n=== Equipment ===")
	for _, slotName := range gear.AllSlots {
		item, err := p.equip.ItemAt(slotName)
		if err != nil {
			p.sendf("  %-8s: (empty)", slotName)
			continue
		}
		dispName := item.ItemID
		if dn, ok := itemDisplayName[item.ItemID]; ok {
			dispName = dn
		}
		p.sendf("  %-8s: %s (IL %d)", slotName, dispName, item.IL)
	}
	if eIL, err := p.equip.EffectiveIL(); err == nil {
		p.sendf("  Effective IL: %d", eIL)
	}
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

// findActivePool returns the first active loot pool in p's zone that p is eligible for.
func findActivePool(p *player) *activeLootPool {
	for _, alp := range gw.lootPools {
		if alp.zoneID != p.zoneID {
			continue
		}
		for _, slot := range alp.pool.Eligible {
			if slot == p.slot {
				return alp
			}
		}
	}
	return nil
}

func cmdPool(p *player) {
	alp := findActivePool(p)
	if alp == nil {
		p.send("No active loot pool in this zone.")
		p.prompt()
		return
	}
	p.send("\r\n[Loot Pool]")
	for i, it := range alp.pool.Items {
		results, _ := alp.pool.LotStatus(it.ID)
		actStr := ""
		for _, lr := range results {
			if lr.Slot == p.slot {
				if lr.Roll == 0 {
					actStr = " (you: pass)"
				} else {
					actStr = fmt.Sprintf(" (you: %d)", lr.Roll)
				}
			}
		}
		p.sendf("  [%d] %s%s", i+1, it.Name, actStr)
	}
	p.prompt()
}

func cmdLot(p *player, numStr string) {
	alp := findActivePool(p)
	if alp == nil {
		p.send("No active loot pool to lot in.")
		p.prompt()
		return
	}
	var idx int
	if _, err := fmt.Sscanf(numStr, "%d", &idx); err != nil || idx < 1 || idx > len(alp.pool.Items) {
		p.sendf("Invalid item number. Pool has %d item(s).", len(alp.pool.Items))
		p.prompt()
		return
	}
	itemID := alp.pool.Items[idx-1].ID
	roll, err := alp.pool.Lot(p.slot, itemID)
	if err != nil {
		p.sendf("Lot failed: %v", err)
		p.prompt()
		return
	}
	itemName := alp.pool.Items[idx-1].Name
	broadcastZoneNoLock(alp.zoneID,
		fmt.Sprintf("[Loot] %s lots %s: %d", p.name, itemName, roll), "")
	resolvePool(alp)
	p.prompt()
}

func cmdPass(p *player, what string) {
	alp := findActivePool(p)
	if alp == nil {
		p.send("No active loot pool to pass on.")
		p.prompt()
		return
	}
	if strings.EqualFold(what, "all") {
		for _, it := range alp.pool.Items {
			_ = alp.pool.Pass(p.slot, it.ID)
		}
		broadcastZoneNoLock(alp.zoneID, fmt.Sprintf("[Loot] %s passes on all items.", p.name), "")
		resolvePool(alp)
		p.prompt()
		return
	}
	var idx int
	if _, err := fmt.Sscanf(what, "%d", &idx); err != nil || idx < 1 || idx > len(alp.pool.Items) {
		p.sendf("Invalid item number. Use 'pool' to see the list, or 'pass all' to decline everything.")
		p.prompt()
		return
	}
	itemID := alp.pool.Items[idx-1].ID
	if err := alp.pool.Pass(p.slot, itemID); err != nil {
		p.sendf("Pass failed: %v", err)
		p.prompt()
		return
	}
	broadcastZoneNoLock(alp.zoneID,
		fmt.Sprintf("[Loot] %s passes on %s.", p.name, alp.pool.Items[idx-1].Name), "")
	resolvePool(alp)
	p.prompt()
}

func cmdEnmity(p *player) {
	mobID := p.combat.TargetMobID
	if mobID == "" {
		p.send("No target. Attack something first.")
		p.prompt()
		return
	}
	et := gw.mobEnmity[mobID]
	if et == nil || et.Len() == 0 {
		p.send("No enmity data for this mob yet.")
		p.prompt()
		return
	}
	reg := gw.mobRegs[p.zoneID]
	m, _ := reg.Get(mobID)
	mobKind := mobID
	if m != nil {
		mobKind = m.Kind
	}
	p.sendf("\r\n=== Enmity — %s ===", mobKind)
	topSlot, _ := et.Top()
	for _, slot := range et.Slots() {
		score, _ := et.Score(slot)
		bar := score * 20 / enmity.EnmityCap
		marker := ""
		if slot == topSlot {
			marker = " ◄ TOP"
		}
		pp := gw.players[slot]
		name := slot
		if pp != nil {
			name = pp.name
		}
		p.sendf("  %-12s %5d/%-5d [%-20s]%s", name, score, enmity.EnmityCap,
			strings.Repeat("█", bar)+strings.Repeat("░", 20-bar), marker)
	}
	p.prompt()
}

func cmdAH(p *player, args []string) {
	if len(args) == 0 {
		p.send("\r\nAuction House — subcommands: browse [category], sell <item-id> <price>, buy <listing-id>, history <item-id>, status, cancel <listing-id>")
		p.prompt()
		return
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "browse":
		ahBrowse(p, args[1:])
	case "sell":
		if len(args) < 3 {
			p.send("Usage: ah sell <item-id> <price>")
			p.prompt()
			return
		}
		price := int64(0)
		fmt.Sscanf(args[2], "%d", &price)
		ahSell(p, args[1], price)
	case "buy":
		if len(args) < 2 {
			p.send("Usage: ah buy <listing-id>")
			p.prompt()
			return
		}
		ahBuy(p, args[1])
	case "history":
		if len(args) < 2 {
			p.send("Usage: ah history <item-id>")
			p.prompt()
			return
		}
		ahHistory(p, args[1])
	case "status":
		ahStatus(p)
	case "cancel":
		if len(args) < 2 {
			p.send("Usage: ah cancel <listing-id>")
			p.prompt()
			return
		}
		ahCancel(p, args[1])
	default:
		p.sendf("Unknown ah subcommand %q. Try: browse, sell, buy, history, status, cancel.", sub)
		p.prompt()
	}
}

func ahBrowse(p *player, args []string) {
	if len(args) == 0 {
		p.send("\r\n=== Auction House — Categories ===")
		for _, cat := range market.AllCategories() {
			items := gw.ah.BrowseCategory(cat)
			p.sendf("  [%d] %-16s  %d listing(s)", int(cat), market.CategoryName(cat), len(items))
		}
		p.send("  Use: ah browse <number>")
		p.prompt()
		return
	}
	catNum := 0
	fmt.Sscanf(args[0], "%d", &catNum)
	cat := market.Category(catNum)
	items := gw.ah.BrowseCategory(cat)
	p.sendf("\r\n=== AH: %s ===", market.CategoryName(cat))
	if len(items) == 0 {
		p.send("  (no listings)")
	}
	for _, it := range items {
		p.sendf("  %-24s  x%d listing(s)  lowest: %d gil  last: %d gil",
			it.ItemName, it.ListingCount, it.LowestPrice, it.LastPrice)
	}
	p.prompt()
}

func ahSell(p *player, itemID string, price int64) {
	if p.inventory[itemID] == 0 {
		p.sendf("You don't have any %s.", itemName(itemID))
		p.prompt()
		return
	}
	cat, ok := itemCategory[itemID]
	if !ok {
		cat = market.CatMisc
	}
	l, err := gw.ah.List(p.slot, itemID, itemName(itemID), cat, price, 1)
	if err != nil {
		p.sendf("AH error: %v", err)
		p.prompt()
		return
	}
	p.inventory[itemID]--
	if p.inventory[itemID] == 0 {
		delete(p.inventory, itemID)
	}
	p.sendf("Listed %s for %d gil. (ID: %s)", itemName(itemID), price, l.ID)
	p.prompt()
}

func ahBuy(p *player, listingID string) {
	// We need to find which itemID this listing belongs to — scan active listings.
	// Instead, expose via ItemPage indirectly: iterate categories to find it.
	var targetItemID string
	for _, cat := range market.AllCategories() {
		for _, summary := range gw.ah.BrowseCategory(cat) {
			pg := gw.ah.ItemPage(summary.ItemID)
			for _, l := range pg.Listings {
				if l.ID == listingID {
					targetItemID = l.ItemID
					break
				}
			}
			if targetItemID != "" {
				break
			}
		}
		if targetItemID != "" {
			break
		}
	}
	if targetItemID == "" {
		p.send("Listing not found.")
		p.prompt()
		return
	}
	pg := gw.ah.ItemPage(targetItemID)
	var listing *market.Listing
	for i := range pg.Listings {
		if pg.Listings[i].ID == listingID {
			listing = &pg.Listings[i]
			break
		}
	}
	if listing == nil {
		p.send("Listing not found.")
		p.prompt()
		return
	}
	if int64(p.gil) < listing.Price {
		p.sendf("Not enough gil. Need %d, have %d.", listing.Price, p.gil)
		p.prompt()
		return
	}
	rec, err := gw.ah.Buy(p.slot, targetItemID)
	if err != nil {
		p.sendf("Purchase failed: %v", err)
		p.prompt()
		return
	}
	p.gil -= int(rec.Price)
	p.inventory[rec.ItemID] += rec.Qty
	p.sendf("You purchase %dx %s for %d gil. (gil remaining: %d)", rec.Qty, rec.ItemName, rec.Price, p.gil)
	p.prompt()
}

func ahHistory(p *player, itemID string) {
	history := gw.ah.HistoryFor(itemID)
	p.sendf("\r\n=== AH History: %s ===", itemName(itemID))
	if len(history) == 0 {
		p.send("  (no sales recorded)")
		p.prompt()
		return
	}
	for _, r := range history {
		p.sendf("  %s  %dx %s  %d gil", r.SoldAt.Format("01/02 15:04"), r.Qty, r.ItemName, r.Price)
	}
	p.prompt()
}

func ahStatus(p *player) {
	listings := gw.ah.SellerListings(p.slot)
	p.sendf("\r\n=== Your AH Listings (gil: %d) ===", p.gil)
	if len(listings) == 0 {
		p.send("  (none)")
	}
	for _, l := range listings {
		p.sendf("  [%s]  %-24s  %d gil  (listed %s)", l.ID, l.ItemName, l.Price, l.ListedAt.Format("01/02 15:04"))
	}
	p.prompt()
}

func ahCancel(p *player, listingID string) {
	l, err := gw.ah.CancelListing(p.slot, listingID)
	if err != nil {
		p.sendf("Cannot cancel: %v", err)
		p.prompt()
		return
	}
	p.inventory[l.ItemID] += l.Qty
	p.sendf("Listing cancelled. %dx %s returned to your inventory.", l.Qty, l.ItemName)
	p.prompt()
}

func cmdConquest(p *player) {
	p.send("\r\n=== Conquest — Region Control ===")
	for _, r := range conquest.DefaultRegions() {
		live, ok := gw.conquestMap.Get(r.ID)
		if !ok {
			continue
		}
		ctrl := conquest.NationName(live.Controller)
		pts := live.Points
		p.sendf("  %-18s  Controller: %-12s  [S:%d B:%d W:%d]",
			live.Name, ctrl,
			pts[conquest.NationSandoria], pts[conquest.NationBastok], pts[conquest.NationWindurst])
	}
	p.send("")
	counts := gw.conquestMap.RegionCount()
	totals := gw.conquestMap.Scoreboard()
	p.send("  Nation totals:")
	for _, n := range conquest.AllNations() {
		p.sendf("    %-12s  regions: %d  points: %d", conquest.NationName(n), counts[n], totals[n])
	}
	myNation := gw.playerNation[p.slot]
	p.sendf("\r\n  Your allegiance: %s", conquest.NationName(myNation))
	p.prompt()
}

func cmdDeclare(p *player, input string) {
	var n conquest.Nation
	switch input {
	case "sandoria", "san":
		n = conquest.NationSandoria
	case "bastok", "bas":
		n = conquest.NationBastok
	case "windurst", "win":
		n = conquest.NationWindurst
	case "neutral", "none":
		n = conquest.NationNeutral
	default:
		p.sendf("Unknown nation %q. Choose: sandoria, bastok, windurst, neutral.", input)
		p.prompt()
		return
	}
	gw.playerNation[p.slot] = n
	p.sendf("You have declared allegiance to %s.", conquest.NationName(n))
	p.prompt()
}

func cmdInventory(p *player) {
	p.sendf("\r\n=== Inventory (Gil: %d) ===", p.gil)
	if len(p.inventory) == 0 {
		p.send("  (empty)")
		p.prompt()
		return
	}
	ids := make([]string, 0, len(p.inventory))
	for id := range p.inventory {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		p.sendf("  %-30s  x%d", itemName(id), p.inventory[id])
	}
	p.prompt()
}

func cmdRecipes(p *player) {
	p.send("\r\n=== Known Recipes ===")
	for recipeID, ingredients := range recipeIngredients {
		recipe, err := craft.LookupRecipe(recipeID)
		if err != nil {
			continue
		}
		skillLvl, _ := p.craftSkill.Level(recipe.CraftType)
		pct := craft.SuccessChance(skillLvl, recipe.Difficulty) * 100
		p.sendf("  %-20s  [%s lv%.0f]  success: %.0f%%", recipeID, recipe.CraftType, recipe.Difficulty, pct)
		for ingID, qty := range ingredients {
			have := p.inventory[ingID]
			haveStr := fmt.Sprintf("(have %d)", have)
			if have < qty {
				haveStr = fmt.Sprintf("(MISSING: need %d, have %d)", qty, have)
			}
			p.sendf("    %dx %s %s", qty, itemName(ingID), haveStr)
		}
		p.sendf("    → %s", itemName(recipe.ItemID))
	}
	p.prompt()
}

func cmdCraftSkills(p *player) {
	p.send("\r\n=== Craft Skills ===")
	for _, ct := range craft.AllCrafts {
		lvl, _ := p.craftSkill.Level(ct)
		p.sendf("  %-14s  %.1f / %.0f", ct, lvl, craft.SkillCap)
	}
	p.prompt()
}

func cmdCraft(p *player, recipeID string) {
	if p.homePoint.IsKO {
		p.send("You are KO'd.")
		p.prompt()
		return
	}
	recipe, err := craft.LookupRecipe(recipeID)
	if err != nil {
		p.sendf("Unknown recipe %q. Use 'recipes' to see available recipes.", recipeID)
		p.prompt()
		return
	}
	ingredients, ok := recipeIngredients[recipeID]
	if !ok {
		p.sendf("No ingredient table for recipe %q.", recipeID)
		p.prompt()
		return
	}

	// Check inventory has all ingredients.
	for ingID, qty := range ingredients {
		if p.inventory[ingID] < qty {
			p.sendf("Missing ingredient: %dx %s (have %d).", qty, itemName(ingID), p.inventory[ingID])
			p.prompt()
			return
		}
	}

	// Consume ingredients.
	for ingID, qty := range ingredients {
		p.inventory[ingID] -= qty
		if p.inventory[ingID] == 0 {
			delete(p.inventory, ingID)
		}
	}

	skillLvl, _ := p.craftSkill.Level(recipe.CraftType)
	result, err := craft.Attempt(recipe, skillLvl, gw.rng)
	if err == craft.ErrBreak {
		p.send(">>> BREAK! <<< Your synthesis failed catastrophically — ingredients destroyed.")
		p.sendf("  (craft skill: %s %.1f)", recipe.CraftType, skillLvl)
		p.prompt()
		return
	}
	if !result.Success {
		// Failure: ingredients are gone (already consumed), no output.
		p.sendf("Your synthesis failed. (craft skill: %s %.1f  difficulty: %.0f)",
			recipe.CraftType, skillLvl, recipe.Difficulty)
		// Small skill gain on failure.
		_ = p.craftSkill.SetLevel(recipe.CraftType, skillLvl+0.05)
		p.prompt()
		return
	}

	// Success.
	p.inventory[result.ItemID]++
	_ = p.craftSkill.SetLevel(recipe.CraftType, skillLvl+0.1)
	newSkill, _ := p.craftSkill.Level(recipe.CraftType)
	if result.HQTier > 0 {
		p.sendf(">>> HQ%d! <<< You synthesise %s! (craft skill: %.1f)", result.HQTier, itemName(result.ItemID), newSkill)
	} else {
		p.sendf("Synthesis complete: %s. (craft skill: %.1f)", itemName(result.ItemID), newSkill)
	}
	p.prompt()
}

func cmdSetJob(p *player, jobID string) {
	if p.homePoint.IsKO {
		p.send("You cannot change jobs while KO'd.")
		p.prompt()
		return
	}
	s, err := job.StatsFor(jobID)
	if err != nil {
		p.sendf("Unknown job %q. Use 'jobs' to list all 22 jobs.", jobID)
		p.prompt()
		return
	}
	p.jobID = jobID
	p.recastTracker = job.NewRecastTracker(abilitiesForJob(jobID))
	applyJobStats(p)
	// Restore HP to full on job change (FFXI-style rest at moogle).
	p.hp = p.maxHP
	p.mp = p.maxMP
	mpStr := fmt.Sprintf("MP: %d", p.maxMP)
	if s.BaseMP == 0 {
		mpStr = "MP: --  (melee job)"
	}
	p.sendf("Job changed to %s. HP: %d  %s", jobID, p.maxHP, mpStr)
	p.prompt()
}

func cmdSetSubJob(p *player, subJobID string) {
	if subJobID == "NONE" || subJobID == "" {
		if p.charJob != nil {
			p.charJob.Sub = ""
			p.charJob.SubLvl = 0
		}
		p.send("Sub-job cleared.")
		p.prompt()
		return
	}
	if _, err := job.StatsFor(subJobID); err != nil {
		p.sendf("Unknown job %q. Use 'jobs' to list all.", subJobID)
		p.prompt()
		return
	}
	if subJobID == p.jobID {
		p.send("Sub-job cannot match main job.")
		p.prompt()
		return
	}
	subLvl := p.charXP.Level / 2
	cj, err := job.NewCharJob(p.jobID, subJobID, p.charXP.Level, subLvl)
	if err != nil {
		p.sendf("Cannot set sub-job: %v", err)
		p.prompt()
		return
	}
	p.charJob = cj
	combined, _ := cj.CombinedStats()
	p.sendf("Sub-job set to %s (effective level %d). STR: %d VIT: %d INT: %d",
		subJobID, cj.EffectiveSubLevel(), combined.STR, combined.VIT, combined.INT)
	p.prompt()
}

func cmdSubJob(p *player) {
	if p.charJob == nil || p.charJob.Sub == "" {
		p.send("No sub-job set. Use 'setsubjob <JOB>' to set one.")
		p.prompt()
		return
	}
	cj := p.charJob
	combined, err := cj.CombinedStats()
	if err != nil {
		p.sendf("Sub-job info error: %v", err)
		p.prompt()
		return
	}
	p.sendf("\r\n=== Job ===")
	p.sendf("  Main: %s Lv%d", cj.Main, cj.MainLvl)
	p.sendf("  Sub:  %s Lv%d (effective: Lv%d)", cj.Sub, cj.SubLvl, cj.EffectiveSubLevel())
	p.sendf("  Combined STR:%d DEX:%d VIT:%d AGI:%d INT:%d MND:%d CHR:%d",
		combined.STR, combined.DEX, combined.VIT, combined.AGI,
		combined.INT, combined.MND, combined.CHR)
	p.prompt()
}

func cmdMerits(p *player) {
	mb := p.meritBank
	p.sendf("\r\n=== Merit Points: %d/%d ===", mb.Points, merit.MeritCap)
	p.sendf("  %-10s  %s  %s", "Category", "Tier", "Next cost")
	for _, cat := range merit.AllCategories {
		tier := mb.TierOf(cat)
		nextCost := "maxed"
		if tier < merit.MaxTierPerCategory {
			nextCost = fmt.Sprintf("%d mp", tier+1)
		}
		p.sendf("  %-10s  %d/5  %s", cat, tier, nextCost)
	}
	p.prompt()
}

func cmdMeritSpend(p *player, category string) {
	err := p.meritBank.Spend(category)
	switch err {
	case nil:
		tier := p.meritBank.TierOf(category)
		p.sendf("Merit spent. %s is now tier %d. (Points remaining: %d)",
			category, tier, p.meritBank.Points)
	case merit.ErrInvalidCategory:
		p.sendf("Unknown category %q. Valid: %s", category, strings.Join(merit.AllCategories, ", "))
	case merit.ErrTierCapped:
		p.sendf("%s is already at maximum tier (5).", category)
	case merit.ErrInsufficientMerits:
		cost := p.meritBank.TierOf(category) + 1
		p.sendf("Need %d merit point(s) for %s. You have %d.", cost, category, p.meritBank.Points)
	default:
		p.sendf("Merit spend error: %v", err)
	}
	p.prompt()
}

func crystalPos(p *player) telecrystal.Vec3 {
	return telecrystal.Vec3{X: p.pos.X, Y: p.pos.Y, Z: p.pos.Z}
}

func cmdCrystals(p *player) {
	nearby := telecrystal.InScene(p.zoneID)
	if len(nearby) == 0 {
		p.send("No telecrystals in this zone.")
		p.prompt()
		return
	}
	p.sendf("\r\n=== Telecrystals in %s ===", zoneName(p.zoneID))
	ppos := crystalPos(p)
	for _, c := range nearby {
		dist := c.Dist2D(ppos)
		inRange := ""
		if c.InRange(ppos) {
			inRange = " [IN RANGE]"
		}
		p.sendf("  %-40s → %-12s  dist: %.0f  cost: %d Gil%s",
			c.ID, c.TargetName, dist, c.CastCost, inRange)
	}
	p.prompt()
}

func cmdTravel(p *player, crystalID string) {
	c, err := telecrystal.Validate(crystalID, p.zoneID, crystalPos(p), p.gil)
	switch err {
	case telecrystal.ErrUnknownCrystal:
		p.sendf("Unknown crystal %q.", crystalID)
		p.prompt()
		return
	case telecrystal.ErrWrongScene:
		p.sendf("Crystal %q is not in this zone.", crystalID)
		p.prompt()
		return
	case telecrystal.ErrInsufficientGold:
		p.sendf("Need %d Gil to travel. You have %d.", c.CastCost, p.gil)
		p.prompt()
		return
	}
	if err != nil {
		p.sendf("Travel failed: %v", err)
		p.prompt()
		return
	}
	// Deduct cost and teleport.
	p.gil -= c.CastCost
	p.sendf("The crystal resonates... you are transported to %s! (-%d Gil)", c.TargetName, c.CastCost)
	gw.mu.Lock()
	broadcastZoneNoLock(p.zoneID, fmt.Sprintf("%s vanishes into a telecrystal.", p.name), p.slot)
	_ = gw.zoneMgr.Transfer(p.slot, c.TargetScene)
	p.combat.TargetMobID = ""
	p.zoneID = c.TargetScene
	p.pos = mob.Pos{X: c.SpawnPos.X, Y: c.SpawnPos.Y, Z: c.SpawnPos.Z}
	syncChatSession(p)
	broadcastZoneNoLock(p.zoneID, fmt.Sprintf("%s arrives via telecrystal.", p.name), p.slot)
	gw.mu.Unlock()
	cmdLook(p)
}

func cmdTouchCrystal(p *player) {
	ppos := crystalPos(p)
	nearby := telecrystal.InScene(p.zoneID)
	var touched *telecrystal.Crystal
	for i := range nearby {
		if nearby[i].InRange(ppos) {
			touched = &nearby[i]
			break
		}
	}
	if touched == nil {
		p.send("No telecrystal within range. Move closer.")
		p.prompt()
		return
	}
	p.sendf("You touch the crystal. [%s] (→ %s, cost %d Gil)", touched.ID, touched.TargetName, touched.CastCost)
	p.sendf("Use 'travel %s' to teleport.", touched.ID)
	p.prompt()
}

// abilitiesForJob returns the known abilities for a job ID.
func abilitiesForJob(jobID string) []job.Ability {
	switch jobID {
	case job.WAR:
		return job.WarriorAbilities()
	case job.WHM:
		return job.WhiteMageAbilities()
	default:
		return nil
	}
}

func cmdJA(p *player, abilityID string) {
	now := time.Now()
	err := p.recastTracker.Use(abilityID, now, p.charXP.Level)
	switch err {
	case nil:
		// Apply ability effect.
		switch abilityID {
		case "provoke":
			// Add enmity on current target.
			mobID := p.combat.TargetMobID
			if mobID != "" {
				if gw.mobEnmity[mobID] == nil {
					gw.mobEnmity[mobID] = enmity.NewTable()
				}
				gw.mobEnmity[mobID].Add(p.slot, 2000)
				reg := gw.mobRegs[p.zoneID]
				if topSlot, terr := gw.mobEnmity[mobID].Top(); terr == nil {
					if m, ok := reg.Get(mobID); ok && m.AggroSlot != topSlot {
						m.AggroSlot = topSlot
					}
				}
				p.sendf("You shout Provoke! (Enmity +2000 on target)")
			} else {
				p.send("No target for Provoke.")
			}
		case "berserk":
			p.sendf("You enter Berserk! (Haste +30%% for 3 minutes)")
		case "warcry":
			broadcastZone(p.zoneID, fmt.Sprintf("%s lets out a fierce battle cry!", p.name), "")
		case "benediction":
			p.hp = p.maxHP
			p.mp = p.maxMP
			p.sendf("Benediction! HP and MP fully restored.")
		default:
			p.sendf("You use %s.", abilityID)
		}
	case job.ErrAbilityOnRecast:
		remaining, _ := p.recastTracker.RecastRemaining(abilityID, now)
		p.sendf("%s is on recast. (%s remaining)", abilityID, remaining.Round(time.Second))
	case job.ErrAbilityLevelGated:
		p.sendf("You need a higher level to use %s.", abilityID)
	case job.ErrAbilityUnknown:
		p.sendf("Unknown ability %q. Use 'recasts' to see your abilities.", abilityID)
	default:
		p.sendf("Cannot use %s: %v", abilityID, err)
	}
	p.prompt()
}

func cmdRecasts(p *player) {
	now := time.Now()
	abilities := abilitiesForJob(p.jobID)
	if len(abilities) == 0 {
		p.sendf("No abilities registered for job %s.", p.jobID)
		p.prompt()
		return
	}
	p.sendf("\r\n=== Job Abilities [%s] ===", p.jobID)
	for _, a := range abilities {
		if p.charXP.Level < a.MinLevel {
			p.sendf("  %-18s (requires Lv%d)", a.ID, a.MinLevel)
			continue
		}
		remaining, _ := p.recastTracker.RecastRemaining(a.ID, now)
		status := "ready"
		if remaining > 0 {
			status = remaining.Round(time.Second).String()
		}
		p.sendf("  %-18s %s", a.ID, status)
	}
	p.prompt()
}

func cmdCast(p *player, spell string) {
	if p.statFX.IsSilenced() {
		p.send("You are silenced and cannot cast spells.")
		p.prompt()
		return
	}
	const mpCostInvis = 50
	const mpCostSneak = 50
	const duration = 60 * time.Second
	now := time.Now()
	switch spell {
	case "invisible", "invis":
		if p.mp < mpCostInvis {
			p.sendf("Not enough MP. (need %d, have %d)", mpCostInvis, p.mp)
			p.prompt()
			return
		}
		p.mp -= mpCostInvis
		p.isInvisible = true
		p.invisExpires = now.Add(duration)
		p.sendf("You cast Invisible. (60s — sight aggro blocked. MP: %d)", p.mp)
	case "sneak":
		if p.mp < mpCostSneak {
			p.sendf("Not enough MP. (need %d, have %d)", mpCostSneak, p.mp)
			p.prompt()
			return
		}
		p.mp -= mpCostSneak
		p.isSneaking = true
		p.sneakExpires = now.Add(duration)
		p.sendf("You cast Sneak. (60s — sound aggro blocked. MP: %d)", p.mp)
	default:
		p.sendf("Unknown spell %q. Available: invisible, sneak", spell)
	}
	p.prompt()
}

func cmdCrisis(p *player) {
	st := gw.wcrisis.Status()
	if st.Phase == worldcrisis.PhaseIdle {
		p.send("No active World Crisis.")
		p.prompt()
		return
	}
	p.sendf("\r\n=== World Crisis ===")
	p.sendf("  Phase:         %s", st.Phase)
	p.sendf("  LEY Integrity: %d/%d", st.LeyIntegrity, worldcrisis.LeyMax)
	if st.Outcome != worldcrisis.OutcomeNone {
		p.sendf("  Outcome:       %s", st.Outcome)
	}
	if !st.PhaseDeadline.IsZero() {
		remaining := time.Until(st.PhaseDeadline).Round(time.Second)
		if remaining > 0 {
			p.sendf("  Phase ends in: %s", remaining)
		}
	}
	p.sendf("  Objectives completed:")
	for obj, t := range st.Objectives {
		p.sendf("    %s — %s", obj, t.Format("15:04:05"))
	}
	if len(st.Objectives) == 0 {
		p.send("    (none yet)")
	}
	p.sendf("  Objectives needed: intercept + anchor + ritual (concurrent within 5 min)")
	p.prompt()
}

func cmdJobs(p *player) {
	p.send("\r\n=== Jobs (22) — FFXI-parity ===")
	for _, j := range job.AllJobs {
		s, _ := job.StatsFor(j)
		hpStr := fmt.Sprintf("HP+%d/lv", s.HPPerLevel)
		mpStr := "no MP"
		if s.BaseMP > 0 {
			mpStr = fmt.Sprintf("MP+%d/lv", s.MPPerLevel)
		}
		cur := ""
		if j == p.jobID {
			cur = " <--"
		}
		p.sendf("  %-4s  %-12s %-12s  STR:%d DEX:%d VIT:%d AGI:%d INT:%d MND:%d CHR:%d%s",
			j, hpStr, mpStr, s.STR, s.DEX, s.VIT, s.AGI, s.INT, s.MND, s.CHR, cur)
	}
	p.send("Use 'setjob <ABBR>' to change your job (restores HP/MP).")
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
  setjob <JOB>        — change your job (WAR/WHM/BLM/RDM/THF/PLD/DRK/… 22 total)
  jobs                — list all 22 jobs with HP/MP growth and base stats
  pool / loot         — show active treasure pool in this zone
  lot <N>             — roll on item N in the loot pool
  pass <N> / pass all — decline item(s) in the loot pool
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

	startHP, _ := job.HPAtLevel(job.WAR, 1)
	startMP, _ := job.MPAtLevel(job.WAR, 1)
	if startHP == 0 {
		startHP = defaultHP
	}
	p := &player{
		slot:        slot,
		name:        name,
		zoneID:      0,
		pos:         mob.Pos{X: 0, Y: 2, Z: 0},
		hp:          startHP,
		maxHP:       startHP,
		mp:          startMP,
		maxMP:       startMP,
		tp:          &combatTp.TPState{},
		statFX:      status.New(),
		combat:      &mob.PlayerCombat{BaseDamage: playerDamage, MeleeRange: playerMeleeRng},
		miningSkill: 0,
		charXP:      xp.NewCharXP(),
		homePoint:   homepoint.NewState(0),
		wsSkill:    "Fast Blade",
		jobID:      job.WAR,
		charJob:       func() *job.CharJob { cj, _ := job.NewCharJob(job.WAR, "", 1, 0); return cj }(),
		meritBank:     merit.NewMeritBank(),
		recastTracker: job.NewRecastTracker(job.WarriorAbilities()),
		inventory:  make(map[string]int),
		craftSkill: craft.NewCraftSkill(),
		gil:        500, // starting gil
		equip:      gear.NewEquipment(),
		conn:       conn,
		w:           w,
	}

	gw.mu.Lock()
	gw.players[slot] = p
	_ = gw.zoneMgr.Enter(slot, name, 0)
	gw.chatRouter.Register(slot, chat.Session{Name: name, SceneID: 0, Pos: chat.Pos{X: 0, Y: 2, Z: 0}})
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
		gw.chatRouter.Unregister(slot)
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

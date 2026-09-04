package main

import (
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"sync"

	"dragonsnshit/packages2/common"
	"dragonsnshit/server/attention"
	"dragonsnshit/server/chat"
	combatTp "dragonsnshit/server/combat"
	"dragonsnshit/server/craft"
	"dragonsnshit/server/fieldoffice"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/idunaauth"
	"dragonsnshit/server/idunaclient"
	"dragonsnshit/server/integrity"
	jobpkg "dragonsnshit/server/job"
	"dragonsnshit/server/ledger"
	"dragonsnshit/server/player"
	"dragonsnshit/server/skillchain"
	"dragonsnshit/server/store"
	"dragonsnshit/server/system"
	"dragonsnshit/server/techpressure"
	"dragonsnshit/server/telecrystal"
	"dragonsnshit/server/trapxapi"
	"dragonsnshit/server/worldapi"
	"dragonsnshit/server/worldcrisis"
)

// gameWorld implements player.RaycastWorld with real entity (player) hit detection --
// backend-unification, 2026-08-03: ported from SHANKPIT's own sibling repo's real, tested
// gameWorld.RayTrace (this exact ray-vs-sphere math, hitboxRadius, chest-height approximation),
// not reinvented. Before this, GoblinFoxDragon's own RayTrace was a permanent stub always
// returning "no hit" -- hitscan shooting (BtnAttack, HandleShankFire) has never actually been
// able to hit anything in this backend, ever, not just "no wall collision." This fixes that for
// player-vs-player hits specifically. Dropped SHANKPIT's own sceneID cross-scene guard -- this
// backend has no per-client scene tracking at all yet (unchanged, named in
// DRAGONSNSHIT_TWO_BACKENDS_AUDIT.md), so every connected client is implicitly the same scene.
// No mutex needed (unlike SHANKPIT's own real version, which takes one): every real call site
// here runs HandleShankFire synchronously inside the main read loop's own PacketUserCmd case,
// the same single-threaded context that already safely owns `clients` -- adding a mutex here
// would be redundant, not safer.
type gameWorld struct {
	clients   map[string]clientInfo
	shooterID uint8
}

// gameEntityHit is the result when the ray intersects another connected client's real position.
type gameEntityHit struct {
	pos      system.Vec3
	clientID uint8
	slot     string // the hit client's own key in gw.clients -- carried through so Entity() can apply real damage
	clients  map[string]clientInfo
}

func (h gameEntityHit) Position() system.Vec3       { return h.pos }
func (h gameEntityHit) Entity() player.LivingEntity { return realEntity{clients: h.clients, slot: h.slot} }

// realEntity applies real damage to a real connected client's hpState (backend-unification,
// 2026-08-03, founder: "for damage we want to make it match up") -- replaces the previous
// nopEntity no-op now that gameWorld has a real client to hit in the first place. h.clients is
// the SAME map the main loop owns (Go maps are reference types), so this mutation is visible
// there immediately after HandleShankFire returns -- no separate write-back plumbing needed.
// Mirrors PacketWSCast's own already-real damage application exactly (same
// targetInfo.hpState.TakeDamage(...) call, same defensive fallback for a client that somehow
// has no hpState yet), just reached via a hitscan hit instead of an explicit weapon-skill cast.
type realEntity struct {
	clients map[string]clientInfo
	slot    string
}

func (e realEntity) Hurt(amount float64, _ player.DamageSource) {
	info, ok := e.clients[e.slot]
	if !ok {
		return
	}
	if info.hpState == nil {
		fallbackHP, _ := jobpkg.HPAtLevel(jobpkg.WAR, 1)
		info.hpState = combatTp.NewHPState(fallbackHP)
	}
	info.hpState.TakeDamage(int(amount))
	e.clients[e.slot] = info
}

const hitboxRadius = 0.4 // matches SHANKPIT's own real value, a real tuned constant, not a guess

func (gw *gameWorld) RayTrace(start, end system.Vec3) (player.RaycastResult, bool) {
	dir := end.Sub(start)
	maxDist := dir.Len()
	if maxDist < 1e-6 {
		return nil, false
	}
	dirN := dir.Mul(1.0 / maxDist)

	bestT := maxDist + 1
	var bestHit *gameEntityHit
	for slot, c := range gw.clients {
		if c.id == gw.shooterID {
			continue
		}
		// Approximate each player as a sphere centered at chest height, same as SHANKPIT's own
		// real version.
		center := system.Vec3{X: c.pos.X, Y: c.pos.Y + 0.9, Z: c.pos.Z}
		w := center.Sub(start)
		t := w.Dot(dirN)
		if t < 0 || t > maxDist {
			continue
		}
		closest := start.Add(dirN.Mul(t))
		if center.Sub(closest).Len() < hitboxRadius && t < bestT {
			bestT = t
			h := gameEntityHit{pos: closest, clientID: c.id, slot: slot, clients: gw.clients}
			bestHit = &h
		}
	}
	if bestHit != nil {
		return *bestHit, true
	}
	return nil, false
}

type shankPlayer struct {
	pos       system.Vec3
	eyeHeight float64
	world     player.RaycastWorld
}

func (p *shankPlayer) Position() system.Vec3 { return p.pos }
func (p *shankPlayer) EyeHeight() float64    { return p.eyeHeight }
func (p *shankPlayer) World() player.RaycastWorld {
	return p.world
}
func (p *shankPlayer) SendSound(name string, pos system.Vec3) {
	fmt.Printf("[sound] %s at %.2f %.2f %.2f\n", name, pos.X, pos.Y, pos.Z)
}

type voxelBlock struct {
	x       uint8
	y       uint8
	z       uint8
	blockID uint16
}

type chunkCoord struct {
	x int
	z int
}

type clientInfo struct {
	id            uint8
	playerID      string // IDUNA subject ("sub") from JWT — empty until authenticated
	lastVoxelSent time.Time
	chunkIndex    int
	tp            *combatTp.TPState // backend-unification Sprint 3: real TP, apps2/mud's own combat.TPState
	// backend-unification follow-up (2026-07-31): real job/level fetched from IDUNA on connect
	// (jobpkg.WAR/level 1 fallback if IDUNA has no character row yet). hpState uses
	// combatTp.HPState (server/combat's own real KO state machine, NewHPState/TakeDamage/IsKO)
	// rather than raw ints -- apps2/mud itself doesn't call this type directly (it drives KO
	// through its own separate homepoint.State.IsKO field instead), but the mechanics are the
	// same shape and reusing the tested type here avoids re-deriving KO/damage-floor logic by
	// hand. In-memory only, same as apps2/mud's own p.hp -- not persisted to IDUNA (a life's
	// current HP isn't durable character state, matching every MMORPG's own convention).
	jobMain string
	level   int
	hpState *combatTp.HPState
	// currentXP is fetched from IDUNA's real Character.CurrentXP on connect and mutated locally
	// on respawn (see PacketRespawn's own handler) -- not written back to IDUNA yet, a further
	// gap named there, not silently hidden.
	currentXP int
	// pos/yaw (backend-unification, 2026-08-03): real server-authoritative position, integrated
	// from raw UserCmd input every tick by integrateMovement (snapshot.go) -- picked deliberately
	// over trusting a client-reported position, a real cheat vector for an MMO. No collision
	// against world geometry yet (world.RayTrace is a real stub, always returns false -- a
	// pre-existing gap, not introduced here); this tracks *where input says a player went*, not
	// yet *where they were physically allowed to go*. Broadcast to other clients via
	// buildSnapshotPacket/PacketSnapshot, confirmed-unused until now.
	pos system.Vec3
	yaw float32
}

// wsChainState mirrors apps2/mud's own gw.mobChains, keyed by the TARGET client's slot
// (remote.String()) instead of a mob ID -- apps2/server-go has no mob registry, so weapon
// skills here are PvP-shaped: chain windows are per-victim, same "does the next WS on this
// target land inside the resonance window" logic, just scored against another player instead
// of a mob.
type wsChainState struct {
	Attrs []skillchain.Resonance
	At    time.Time
}

// TRAPX city simulation state (shared across server tick and trapxapi HTTP handler).
var (
	cityFOReg     = fieldoffice.NewRegistry()
	cityAttnReg   = attention.NewRegistry()
	cityIntReg    = integrity.NewRegistry()
	cityTechClock = techpressure.NewClock()
	cityLedger    = ledger.NewLedger()
)

func initCityState() {
	for _, fo := range fieldoffice.DefaultFieldOffices(nil) {
		cityFOReg.Add(fo)
	}
	for _, id := range []string{"district-residential", "district-commercial", "district-industrial", "district-underground", "district-abandoned"} {
		cityIntReg.GetOrCreate(id)
	}
}

func main() {
	worldapiPort := flag.Int("worldapi-port", 7070, "HTTP port for the worldapi /chunks endpoint (0 = disabled)")
	trapxPort := flag.Int("trapx-port", 7071, "HTTP port for the TRAPX city-state API (0 = disabled)")
	udpPort := flag.Int("udp-port", 6969, "UDP port for the real game protocol -- was hardcoded to "+
		"6969, which SHANKPIT's own shank_server already occupies on this box (2026-08-03, founder: "+
		"\"we need to get the dragonfly server seeded with a world... to debug\"); made configurable "+
		"so a debug instance can run alongside SHANKPIT without a port conflict, default unchanged "+
		"for anyone who wants the original behavior")
	flag.Parse()

	initCityState()

	// Dungeon instancing (DUNGEON_NORTHSTAR.md Milestone 1, real v0, 2026-09-04): shared between
	// the worldapi chunk-generator closure below and the UDP PacketDungeonEnter handler further
	// down -- see server/worldapi/dungeon_instance.go's own doc comment for why this lives here
	// (this persistent process, not a REDGARDEN-style per-match spawned server) rather than
	// wiring through REDGARDEN's matchmaker.
	dungeonRegistry := worldapi.NewDungeonInstanceRegistry(208)

	// Start the worldapi HTTP server — SHANKPIT connects here with --dragonfly-url http://localhost:7070
	if *worldapiPort > 0 {
		gen := worldapi.NewDragonflyChunkGenerator(func(sceneID, chunkX, chunkZ int) []worldapi.WorldBlock {
			if blocks, ok := dungeonRegistry.BlocksForChunk(sceneID, chunkX, chunkZ); ok {
				return blocks
			}
			return worldapi.ProceduralWorldStore(sceneID, chunkX, chunkZ)
		})
		srv := worldapi.New(gen)
		go func() {
			addr := fmt.Sprintf(":%d", *worldapiPort)
			fmt.Printf("worldapi listening on %s (scene 0=meadow, 1=hills, 2=caves)\n", addr)
			if err := http.ListenAndServe(addr, srv); err != nil {
				fmt.Printf("worldapi: %v\n", err)
			}
		}()
	}

	// Start the TRAPX city-state API — Emily Prime Dragon reads this each RSI cycle.
	if *trapxPort > 0 {
		tsrv := trapxapi.New(cityFOReg, cityAttnReg, cityIntReg, cityTechClock, cityLedger)
		go func() {
			addr := fmt.Sprintf(":%d", *trapxPort)
			fmt.Printf("trapxapi listening on %s (GET /api/v1/trapx/city-state, POST /api/v1/trapx/events)\n", addr)
			if err := http.ListenAndServe(addr, tsrv); err != nil {
				fmt.Printf("trapxapi: %v\n", err)
			}
		}()
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", *udpPort))
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("Go backend listening on :%d\n", *udpPort)
	authVerifier := idunaauth.NewVerifier()
	idunaClient := idunaclient.New()
	buf := make([]byte, 2048)
	clientStore := store.NewMemoryClientStore()
	clients := make(map[string]clientInfo)
	nextClientID := uint8(0)
	chatRouter := chat.New()
	clientAddrs := make(map[string]*net.UDPAddr) // slot → addr for chat delivery
	wsChains := make(map[string]wsChainState)    // target slot → last weapon skill landed on them (backend-unification Sprint 3)

	// World Crisis state machine (S76-05).
	crisis := worldcrisis.New()
	broadcastCh := make(chan []byte, 64) // server goroutine → broadcast goroutine

	// Broadcast goroutine: sends packets from broadcastCh to all connected clients.
	go func() {
		var addrsMu sync.RWMutex
		_ = addrsMu // clients/clientAddrs are accessed from main loop; broadcast via channel
		for pkt := range broadcastCh {
			for slot, addr := range clientAddrs {
				if _, ok := clients[slot]; ok {
					conn.WriteToUDP(pkt, addr)
				}
			}
		}
	}()

	// World Crisis tick goroutine: drives phase machine + periodic meter broadcast.
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		lastPhase := worldcrisis.PhaseIdle
		for t := range ticker.C {
			changed, oldP, newP := crisis.Tick(t)
			s := crisis.Status()
			// Always broadcast the world crisis update every 5s (for meter sync).
			// On phase change: broadcast immediately + PATCH IDUNA.
			if changed {
				fmt.Printf("[worldcrisis] %s → %s (ley=%d)\n", oldP, newP, s.LeyIntegrity)
				if s.EventID != "" {
					_ = idunaClient.PatchWorldEvent(s.EventID, string(newP), s.LeyIntegrity)
				}
				lastPhase = newP
			}
			_ = lastPhase
			pkt := buildWorldCrisisPacket(s)
			select {
			case broadcastCh <- pkt:
			default:
			}
		}
	}()

	// S171-04 chat bridge, receive side: polls IDUNA for EINHORN_SURVIVAL-
	// origin messages and broadcasts them to every connected client via the
	// existing broadcastCh mechanism (same pattern as the World Crisis
	// ticker above) -- doesn't need clientAddrs directly, so it doesn't
	// need that map lifted to package scope after all, contrary to what
	// CHAT_BRIDGE_TO_EINHORN_SURVIVAL_SPEC.md originally assumed before
	// broadcastCh was found. Starts from the current high-water mark (first
	// tick only records lastSeenID, doesn't broadcast) so a restart doesn't
	// replay EINHORN_SURVIVAL's whole chat history into the game.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		lastSeenID := int64(-1)
		for range ticker.C {
			msgs, err := idunaClient.GetChatMessages(max(lastSeenID, 0), 50)
			if err != nil {
				fmt.Printf("[chat-bridge] poll failed: %v\n", err)
				continue
			}
			if lastSeenID < 0 {
				for _, m := range msgs {
					if m.ID > lastSeenID {
						lastSeenID = m.ID
					}
				}
				if lastSeenID < 0 {
					lastSeenID = 0
				}
				continue
			}
			for _, m := range msgs {
				if m.ID > lastSeenID {
					lastSeenID = m.ID
				}
				if m.SenderSource != "einhorn_survival" {
					continue
				}
				pkt := chat.EncodeChat(chat.ChatYell, "[Paper] "+m.SenderName, m.Body)
				select {
				case broadcastCh <- pkt:
				default:
				}
			}
		}
	}()

	lastSnapshotBroadcast := time.Now()
	const snapshotInterval = 250 * time.Millisecond // matches the read-loop's own natural poll cadence below -- see its own doc comment for why this isn't the higher SHANKPIT-sibling rate (33ms/30Hz)

	for {
		// Backend-unification, 2026-08-03: real player-position broadcast (PacketSnapshot,
		// confirmed unused until now). Deliberately built and sent from THIS single-threaded
		// main loop, not a new ticker goroutine like the World Crisis one above -- `clients`/
		// `clientAddrs` have no real mutex protecting them today (the existing broadcast
		// goroutine already reads them unsynchronized, a pre-existing, named, not-fixed-here
		// race; see its own comment, "clients/clientAddrs are accessed from main loop; broadcast
		// via channel" -- an intent that was only half-wired). Adding a THIRD unsynchronized
		// accessor would be a real, new Go crash risk (concurrent map read/write panics), not
		// just a style nit. Doing it here instead means it only ever runs where `clients` is
		// already safely, singly owned. Real, honest tradeoff: ~4Hz (driven by this loop's own
        // 250ms read-timeout cadence) instead of SHANKPIT sibling's own 30Hz -- a named,
		// follow-on-able limitation, not silently accepted as good enough forever.
		if time.Since(lastSnapshotBroadcast) >= snapshotInterval {
			lastSnapshotBroadcast = time.Now()
			all := make([]snapshotPeer, 0, len(clients))
			for slot, info := range clients {
				if _, ok := clientAddrs[slot]; !ok {
					continue
				}
				peer := snapshotPeer{id: info.id, pos: info.pos, yaw: info.yaw}
				if info.hpState != nil {
					peer.health, peer.maxHP, peer.isKO = info.hpState.Current, info.hpState.Max, info.hpState.IsKO
				}
				all = append(all, peer)
			}
			for slot, info := range clients {
				addr, ok := clientAddrs[slot]
				if !ok {
					continue
				}
				peers := make([]snapshotPeer, 0, len(all))
				for _, peer := range all {
					if peer.id != info.id {
						peers = append(peers, peer)
					}
				}
				if len(peers) == 0 {
					continue
				}
				conn.WriteToUDP(buildSnapshotPacket(info.id, peers), addr)
			}
		}

		conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			fmt.Printf("read error: %v\n", err)
			continue
		}
		if n < 1 {
			continue
		}
		const netHeaderSize = 12
		const userCmdSize = 36
		switch buf[0] {
		case common.PacketConnect:
			// Payload layout: buf[0] = PacketConnect; buf[1:n] = null-terminated JWT string.
			// Validate JWT against IDUNA before accepting the connection.
			slot := remote.String()
			if _, already := clients[slot]; !already {
				jwtStr := ""
				if n > 1 {
					jwtStr = strings.TrimRight(string(buf[1:n]), "\x00")
				}
				claims, authErr := authVerifier.Verify(jwtStr)
				if authErr != nil {
					fmt.Printf("[auth] reject %s: %v\n", slot, authErr)
					sendAuthReject(conn, remote)
					continue
				}
				jobMain, level, maxHP, currentXP := fetchCharacterCombatStats(idunaClient, claims.Subject)
				info := clientInfo{
					id: nextClientID, playerID: claims.Subject, tp: &combatTp.TPState{},
					jobMain: jobMain, level: level, hpState: combatTp.NewHPState(maxHP), currentXP: currentXP,
				}
				if nextClientID < 255 {
					nextClientID++
				}
				clients[slot] = info
				clientAddrs[slot] = remote
				chatRouter.Register(slot, chat.Session{
					Name:    fmt.Sprintf("Player%d", info.id),
					SceneID: 1,
					GuildID: "",
					Pos:     chat.Pos{},
				})
				fmt.Printf("[auth] accept %s playerID=%s id=%d job=%s lvl=%d hp=%d\n", slot, claims.Subject, info.id, jobMain, level, maxHP)
			}
			info := clients[slot]
			sendWelcome(conn, remote, info.id)
			sendVoxelPacket(conn, remote, info)
		case common.PacketUserCmd:
			if n < netHeaderSize+1+userCmdSize {
				continue
			}
			slot := remote.String()
			info, ok := clients[slot]
			if !ok {
				info = clientInfo{id: nextClientID}
				if nextClientID < 255 {
					nextClientID++
				}
			}
			if info.tp == nil {
				info.tp = &combatTp.TPState{}
			}
			info = sendVoxelPacket(conn, remote, info)
			clients[slot] = info
			count := int(buf[netHeaderSize])
			if count < 1 {
				continue
			}
			cmd := parseUserCmd(buf, netHeaderSize+1)
			clientStore.Upsert(slot, cmd)
			// Backend-unification, 2026-08-03: real server-authoritative position, integrated
			// from this UserCmd's own raw input (see integrateMovement's own doc comment,
			// snapshot.go, for the real yaw/forward convention this defines -- no existing
			// on-foot movement precedent anywhere in this codebase family to match against).
			info.pos = integrateMovement(info.pos, cmd, float64(cmd.Msec)/1000.0)
			info.pos = groundClampY(info.pos) // real Y-axis ground collision, see its own doc comment (snapshot.go)
			info.yaw = cmd.Yaw
			clients[slot] = info
			if cmd.Buttons&common.BtnAttack != 0 {
				// Backend-unification, 2026-08-03: real per-shooter player + real gameWorld
				// (entity hit detection, see its own doc comment) -- replaces a single shared
				// stub `p` that every client's shots used to fire through, whose world.RayTrace
				// always returned "no hit" regardless of shooter or target. Real damage-on-hit
				// is NOT applied here -- same acknowledged, named gap SHANKPIT's own real
				// version has ("real damage routing is handled by the caller," not yet built
				// there either); this fixes hit DETECTION, a separate, real, and previously
				// completely broken concern (hitscan could never hit anything at all, on
				// anyone's shots, not just "no wall collision").
				shooter := &shankPlayer{pos: info.pos, eyeHeight: 1.62, world: &gameWorld{clients: clients, shooterID: info.id}}
				hit, pos, hitEntity := player.HandleShankFire(shooter, float64(cmd.Yaw), float64(cmd.Pitch), int(cmd.WeaponIdx))
				if hit {
					sendImpact(conn, remote, pos, hitEntity, 0)
				}
				// Backend-unification Sprint 3: every attack also feeds real TP, same
				// server/combat.TPState apps2/mud's own auto-attack loop uses -- alongside
				// HandleShankFire's existing hitscan, not replacing it. Flat 1H-sword delay
				// assumed for this slice (no real weapon/gear system wired into
				// apps2/server-go yet, unlike apps2/mud's own gear.Equipment).
				info.tp.AddTP(combatTp.Delay1HSword, 0)
			}

		case common.PacketTelecrystalUse:
			// Payload: buf[1:n] — null-terminated crystal ID string.
			slot := remote.String()
			info, ok := clients[slot]
			if !ok || info.playerID == "" {
				// Reject unauthenticated or unknown clients.
				conn.WriteToUDP([]byte{common.PacketTelecrystalErr, 1}, remote)
				continue
			}
			crystalID := strings.TrimRight(string(buf[1:n]), "\x00")
			// Fetch character from IDUNA to get current scene + gold.
			ch, err := idunaClient.GetCharacter(info.playerID)
			if err != nil {
				fmt.Printf("[telecrystal] GetCharacter %s: %v\n", info.playerID, err)
				conn.WriteToUDP([]byte{common.PacketTelecrystalErr, 2}, remote)
				continue
			}
			playerPos := telecrystal.Vec3{X: ch.PosX, Y: ch.PosY, Z: ch.PosZ}
			crystal, valErr := telecrystal.Validate(crystalID, ch.SceneID, playerPos, ch.GoldBalance)
			if valErr != nil {
				fmt.Printf("[telecrystal] validate %s: %v\n", crystalID, valErr)
				conn.WriteToUDP([]byte{common.PacketTelecrystalErr, 3}, remote)
				continue
			}
			// Deduct gold + update scene/pos atomically via IDUNA.
			if err := idunaClient.TravelTelecrystal(info.playerID, crystal.CastCost,
				crystal.TargetScene, crystal.SpawnPos.X, crystal.SpawnPos.Y, crystal.SpawnPos.Z); err != nil {
				fmt.Printf("[telecrystal] travel %s: %v\n", crystalID, err)
				conn.WriteToUDP([]byte{common.PacketTelecrystalErr, 4}, remote)
				continue
			}
			// Send PacketTelecrystalAck = same wire layout as PacketSceneChange.
			ack := make([]byte, 14)
			ack[0] = common.PacketTelecrystalAck
			ack[1] = uint8(crystal.TargetScene)
			binary.LittleEndian.PutUint32(ack[2:], math.Float32bits(float32(crystal.SpawnPos.X)))
			binary.LittleEndian.PutUint32(ack[6:], math.Float32bits(float32(crystal.SpawnPos.Y)))
			binary.LittleEndian.PutUint32(ack[10:], math.Float32bits(float32(crystal.SpawnPos.Z)))
			conn.WriteToUDP(ack, remote)
			fmt.Printf("[telecrystal] %s → scene=%d spawn=(%.1f,%.1f,%.1f)\n",
				info.playerID, crystal.TargetScene,
				crystal.SpawnPos.X, crystal.SpawnPos.Y, crystal.SpawnPos.Z)

		case common.PacketDungeonEnter:
			// DUNGEON_NORTHSTAR.md Milestone 1, real v0 (2026-09-04): payload = dungeon_index
			// uint8 (not yet used to pick a per-dungeon boss/elite roster -- see
			// GenerateDungeonSpawns' own real signature for that follow-up wiring). Reuses the
			// exact same travel mechanism as PacketTelecrystalUse just above (allocate a
			// destination, update scene/pos via IDUNA, ack with a PacketSceneChange-shaped
			// body) at zero Flow/gold cost -- a dungeon entrance isn't a paid telecrystal.
			if n < 2 {
				continue
			}
			slot := remote.String()
			info, ok := clients[slot]
			if !ok || info.playerID == "" {
				conn.WriteToUDP([]byte{common.PacketDungeonEnterErr, 1}, remote)
				continue
			}
			dungeonIndex := int(buf[1])
			seed := time.Now().UnixNano() // real, fresh per-request seed -- this v0 always
			// allocates a brand-new solo instance per request (no party-roster passthrough or
			// instance-sharing yet, see dungeon_instance.go's own doc comment).
			sceneID, allocated := dungeonRegistry.Allocate(dungeonIndex, seed)
			if !allocated {
				fmt.Printf("[dungeon] enter %s: no free instance slots\n", info.playerID)
				conn.WriteToUDP([]byte{common.PacketDungeonEnterErr, 2}, remote)
				continue
			}
			spawnX, spawnY, spawnZ, _ := dungeonRegistry.EntrySpawn(sceneID) // always ok right
			// after a successful Allocate for this same sceneID -- no separate error path needed.
			if err := idunaClient.TravelTelecrystal(info.playerID, 0, sceneID, spawnX, spawnY, spawnZ); err != nil {
				fmt.Printf("[dungeon] enter %s: %v\n", info.playerID, err)
				conn.WriteToUDP([]byte{common.PacketDungeonEnterErr, 3}, remote)
				continue
			}
			// Send PacketDungeonEnterAck = same wire layout as PacketSceneChange/PacketTelecrystalAck.
			ack := make([]byte, 14)
			ack[0] = common.PacketDungeonEnterAck
			ack[1] = uint8(sceneID)
			binary.LittleEndian.PutUint32(ack[2:], math.Float32bits(float32(spawnX)))
			binary.LittleEndian.PutUint32(ack[6:], math.Float32bits(float32(spawnY)))
			binary.LittleEndian.PutUint32(ack[10:], math.Float32bits(float32(spawnZ)))
			conn.WriteToUDP(ack, remote)
			fmt.Printf("[dungeon] %s → scene=%d (dungeon_index=%d, seed=%d) spawn=(%.1f,%.1f,%.1f)\n",
				info.playerID, sceneID, dungeonIndex, seed, spawnX, spawnY, spawnZ)

		case common.PacketCraftRequest:
			// Payload: JSON {"recipe_id":"...","character_id":"...","reagent_ids":["...","..."]}
			// Server: look up recipe, validate reagents exist in IDUNA, run craft.Attempt,
			// create item + destroy reagents in IDUNA, reply PacketCraftResult JSON.
			slot := remote.String()
			info, ok := clients[slot]
			if !ok || info.playerID == "" {
				sendCraftResult(conn, remote, false, 0, "", "", "unauthenticated")
				continue
			}
			if n < 2 {
				sendCraftResult(conn, remote, false, 0, "", "", "empty payload")
				continue
			}
			var craftReq struct {
				RecipeID    string   `json:"recipe_id"`
				CharacterID string   `json:"character_id"`
				ReagentIDs  []string `json:"reagent_ids"`
			}
			if err := json.Unmarshal(buf[1:n], &craftReq); err != nil {
				sendCraftResult(conn, remote, false, 0, "", "", "invalid JSON")
				continue
			}
			// Look up recipe by ID.
			recipe, recipeErr := craft.LookupRecipe(craftReq.RecipeID)
			if recipeErr != nil {
				sendCraftResult(conn, remote, false, 0, "", "", "unknown recipe")
				continue
			}
			// Validate reagents exist in IDUNA and are owned by the character.
			items, itemsErr := idunaClient.ListItems(craftReq.CharacterID)
			if itemsErr != nil {
				sendCraftResult(conn, remote, false, 0, "", "", "could not load inventory")
				continue
			}
			ownedIDs := map[string]bool{}
			for _, it := range items {
				ownedIDs[it.ItemID] = true
			}
			allOwned := true
			for _, rid := range craftReq.ReagentIDs {
				if !ownedIDs[rid] {
					allOwned = false
					break
				}
			}
			if !allOwned {
				sendCraftResult(conn, remote, false, 0, "", "", "reagent not owned by character")
				continue
			}
			// Fetch character skill for this craft type.
			ch, chErr := idunaClient.GetCharacter(craftReq.CharacterID)
			if chErr != nil {
				sendCraftResult(conn, remote, false, 0, "", "", "character not found")
				continue
			}
			// For now use a fixed skill of 0; skill XP is wired in S76-06.
			_ = ch
			rng := craftRNG()
			result, craftErr := craft.Attempt(recipe, 0.0, rng)
			if craftErr == craft.ErrBreak {
				// Reagents consumed even on break.
				for _, rid := range craftReq.ReagentIDs {
					idunaClient.DestroyItem(rid)
				}
				sendCraftResult(conn, remote, false, 0, "", "", "break")
				continue
			}
			if !result.Success {
				sendCraftResult(conn, remote, false, 0, "", "", "failed")
				continue
			}
			// Remove reagents and create the crafted item.
			for _, rid := range craftReq.ReagentIDs {
				idunaClient.DestroyItem(rid)
			}
			newItemID, createErr := idunaClient.CreateItem(
				craftReq.CharacterID, craftReq.CharacterID,
				recipe.CraftType, result.ItemID, 0)
			if createErr != nil {
				sendCraftResult(conn, remote, false, 0, "", "", "item create failed")
				continue
			}
			sendCraftResult(conn, remote, true, result.HQTier, newItemID, result.ItemID, "")
			fmt.Printf("[craft] %s crafted %s (HQ%d) → %s\n",
				craftReq.CharacterID, result.ItemID, result.HQTier, newItemID)

		case common.PacketSkillXP:
			// Payload: JSON {"character_id":"...","skill_name":"...","delta":N}
			// Server validates the claim is plausible (delta > 0, delta <= maxGrant per tick)
			// then calls IDUNA to increment. Never trust client-reported XP magnitude.
			slot := remote.String()
			info, ok := clients[slot]
			if !ok || info.playerID == "" {
				continue
			}
			if n < 2 {
				continue
			}
			var xpReq struct {
				CharacterID string  `json:"character_id"`
				SkillName   string  `json:"skill_name"`
				Delta       float64 `json:"delta"`
			}
			if err := json.Unmarshal(buf[1:n], &xpReq); err != nil {
				continue
			}
			// Cap per-packet XP grant to prevent inflation (max 1.0 per action).
			const maxXPGrant = 1.0
			if xpReq.Delta <= 0 || xpReq.SkillName == "" {
				continue
			}
			if xpReq.Delta > maxXPGrant {
				xpReq.Delta = maxXPGrant
			}
			go func(cid, skill string, delta float64) {
				if err := idunaClient.IncrementSkill(cid, skill, delta); err != nil {
					fmt.Printf("[skill-xp] IncrementSkill %s/%s: %v\n", cid, skill, err)
				}
			}(xpReq.CharacterID, xpReq.SkillName, xpReq.Delta)

		case common.PacketObjectiveComplete:
			// Payload: JSON {"character_id":"...","objective_type":"anchor|ritual|intercept","stabilize_amount":N}
			slot := remote.String()
			if _, ok := clients[slot]; !ok {
				continue
			}
			if n < 2 {
				continue
			}
			var objReq struct {
				CharacterID     string `json:"character_id"`
				ObjectiveType   string `json:"objective_type"`
				StabilizeAmount int    `json:"stabilize_amount"`
			}
			if err := json.Unmarshal(buf[1:n], &objReq); err != nil {
				continue
			}
			objType := worldcrisis.ObjectiveType(objReq.ObjectiveType)
			if err := crisis.CompleteObjective(objType, objReq.StabilizeAmount, time.Now()); err != nil {
				fmt.Printf("[worldcrisis] objective %s rejected: %v\n", objType, err)
				continue
			}
			s := crisis.Status()
			fmt.Printf("[worldcrisis] objective %s completed (ley=%d)\n", objType, s.LeyIntegrity)

		case common.PacketWSCast:
			// Backend-unification Sprint 3 (EMILY/BACKLOG.md, 2026-07-31): real weapon-skill
			// casting + skillchain resonance, using the same apps2/mud-tested server/combat and
			// server/skillchain packages, wired into apps2/server-go's own UDP loop. PvP-shaped
			// (targets another connected client) rather than PvE -- this backend has no mob
			// registry the way apps2/mud does.
			slot := remote.String()
			info, ok := clients[slot]
			if !ok || info.playerID == "" {
				sendWSResult(conn, remote, wsResultPayload{Error: "unauthenticated"})
				continue
			}
			if info.tp == nil {
				info.tp = &combatTp.TPState{}
			}
			if info.hpState != nil && info.hpState.IsKO {
				sendWSResult(conn, remote, wsResultPayload{Error: "you are KO'd"})
				continue
			}
			if n < 2 {
				continue
			}
			var wsReq struct {
				WSName         string `json:"ws_name"`
				TargetClientID int    `json:"target_client_id"`
			}
			if err := json.Unmarshal(buf[1:n], &wsReq); err != nil {
				sendWSResult(conn, remote, wsResultPayload{Error: "invalid JSON"})
				continue
			}
			if !info.tp.CanWeaponSkill() {
				sendWSResult(conn, remote, wsResultPayload{Error: "not enough TP"})
				continue
			}
			targetSlot := ""
			for s, ci := range clients {
				if int(ci.id) == wsReq.TargetClientID {
					targetSlot = s
					break
				}
			}
			if targetSlot == "" {
				sendWSResult(conn, remote, wsResultPayload{Error: "target not found"})
				continue
			}
			targetInfo := clients[targetSlot]
			if targetInfo.hpState != nil && targetInfo.hpState.IsKO {
				sendWSResult(conn, remote, wsResultPayload{Error: "target is already KO'd"})
				continue
			}
			result, newChainState, ok2 := resolveWSCast(wsReq.WSName, wsChains, targetSlot, int(info.id), wsReq.TargetClientID, time.Now())
			if !ok2 {
				sendWSResult(conn, remote, result)
				continue
			}
			info.tp.UseWeaponSkill()
			clients[slot] = info
			wsChains[targetSlot] = newChainState

			// Apply the damage to the target's real HPState (backend-unification follow-up,
			// 2026-07-31) -- Sprint 3 only ever reported a damage number without touching
			// anything; this is the first real slice where a weapon skill actually hurts
			// someone. TakeKO is intentionally not followed by any respawn/home-point flow here
			// -- apps2/mud's own knockOut() leaves a KO'd player waiting until they actively
			// type 'home' (8% XP penalty) or find another Raise; a real respawn flow for this
			// backend is a separate, larger follow-up, not attempted in this slice. A KO'd
			// player just can't act (see the caster/target IsKO guards above) until something
			// external (not built yet) revives them.
			if targetInfo.hpState == nil {
				// Defensive fallback -- every real client gets a real hpState via
				// PacketConnect's own fetchCharacterCombatStats; this only guards against a
				// theoretical WSCast reaching a client record that skipped that path.
				fallbackHP, _ := jobpkg.HPAtLevel(jobpkg.WAR, 1)
				targetInfo.hpState = combatTp.NewHPState(fallbackHP)
			}
			killed, _ := targetInfo.hpState.TakeDamage(result.Damage)
			result.Killed = killed
			clients[targetSlot] = targetInfo
			result.TargetHP = targetInfo.hpState.Current
			result.TargetMaxHP = targetInfo.hpState.Max

			payload, _ := json.Marshal(result)
			pkt := append([]byte{common.PacketWSResult}, payload...)
			conn.WriteToUDP(pkt, remote)
			if targetAddr, ok := clientAddrs[targetSlot]; ok {
				conn.WriteToUDP(pkt, targetAddr)
			}
			fmt.Printf("[weaponskill] %s -> %s: %s dmg=%d chained=%v\n", slot, targetSlot, result.WSName, result.Damage, result.Chained)

		case common.PacketRespawn:
			// Backend-unification follow-up (2026-07-31): the only way back from KO on this
			// backend so far. Correction, same day: this originally called
			// HPState.RaiseDefault, which applies combat.DefaultRaisePenaltyPct (10%) -- checked
			// against apps2/mud's own actual live behavior and that's the wrong number.
			// apps2/mud's real cmdHome hand-computes an 8% penalty (homepoint.
			// DefaultXPPenaltyPct) and doesn't call HPState.Raise at all; server/homepoint's own
			// ReturnHome() implements that exact 8% mechanic too but apps2/mud's cmdHome
			// duplicates it by hand instead of calling it (pre-existing, unrelated to this
			// change). Fixed here by passing homepoint.DefaultXPPenaltyPct explicitly to
			// HPState.Raise (which does accept an arbitrary pct) instead of trusting its own
			// unrelated 10% default -- still reuses Raise's real HP-reset/IsKO-clear behavior,
			// just with the correct, actually-live percentage. Real per-player XP now comes from
			// IDUNA (currentXP, fetched on connect) instead of a hardcoded 0.
			slot := remote.String()
			info, ok := clients[slot]
			if !ok || info.playerID == "" {
				sendRespawnResult(conn, remote, respawnResultPayload{Error: "unauthenticated"})
				continue
			}
			if info.hpState == nil || !info.hpState.IsKO {
				sendRespawnResult(conn, remote, respawnResultPayload{Error: "not KO'd"})
				continue
			}
			penalty, err := info.hpState.Raise(info.currentXP, float64(homepoint.DefaultXPPenaltyPct))
			if err != nil {
				sendRespawnResult(conn, remote, respawnResultPayload{Error: err.Error()})
				continue
			}
			info.currentXP -= penalty
			if info.currentXP < 0 {
				info.currentXP = 0
			}
			clients[slot] = info

			// Persist the penalty back to IDUNA -- same fire-and-forget, log-on-error pattern
			// PacketSkillXP's own IncrementSkill call already uses just above, not blocking the
			// UDP read loop on an HTTP round trip. Closes the "XP earned isn't written back to
			// IDUNA yet" gap named in EMILY/BACKLOG.md's own unification item.
			go func(cid string, level, currentXP int) {
				if err := idunaClient.UpdateCharacterLevel(cid, level, currentXP); err != nil {
					fmt.Printf("[respawn] UpdateCharacterLevel %s: %v\n", cid, err)
				}
			}(info.playerID, info.level, info.currentXP)

			sendRespawnResult(conn, remote, respawnResultPayload{
				HP: info.hpState.Current, MaxHP: info.hpState.Max, XPPenalty: penalty,
			})
			fmt.Printf("[respawn] %s revived at %d/%d HP\n", slot, info.hpState.Current, info.hpState.Max)

		case common.PacketChat:
			if n < 4 {
				continue
			}
			slot := remote.String()
			channel, target, msg, ok := chat.ParseChatPacket(buf[:n])
			if !ok {
				continue
			}
			deliveries := chatRouter.Deliver(slot, channel, target, msg, common.SayRadius)
			for _, d := range deliveries {
				if addr, ok := clientAddrs[d.To]; ok {
					_, _ = conn.WriteToUDP(d.Packet, addr)
				}
			}
			fmt.Printf("[chat/%s] %s: %s\n", channelName(channel), slot, msg)

			// S171-04 chat bridge: only ChatYell relays to EINHORN_SURVIVAL --
			// GFD's own zone-wide broadcast is the one channel public enough
			// to bridge (see GoblinFoxDragon/docs2/
			// CHAT_BRIDGE_TO_EINHORN_SURVIVAL_SPEC.md's Design Decisions).
			// Async: a slow/failed IDUNA call must never stall the packet loop.
			if channel == chat.ChatYell {
				if sess, ok := chatRouter.GetSession(slot); ok {
					go func(name, body string) {
						if err := idunaClient.PostChatMessageAs("yell", name, "gfd_server", body); err != nil {
							fmt.Printf("[chat-bridge] post failed: %v\n", err)
						}
					}(sess.Name, msg)
				}
			}
		}
	}
}

var craftSeed = rand.NewSource(time.Now().UnixNano())
var craftRandMu sync.Mutex

func craftRNG() *rand.Rand {
	craftRandMu.Lock()
	defer craftRandMu.Unlock()
	return rand.New(rand.NewSource(craftSeed.Int63()))
}

func sendCraftResult(conn *net.UDPConn, remote *net.UDPAddr, success bool, hqTier int, itemID, itemName, errMsg string) {
	payload := map[string]interface{}{
		"success": success, "hq_tier": hqTier,
		"item_id": itemID, "item_name": itemName,
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	b, _ := json.Marshal(payload)
	pkt := append([]byte{common.PacketCraftResult}, b...)
	conn.WriteToUDP(pkt, remote)
}

// wsResultPayload is PacketWSResult's JSON body (backend-unification Sprint 3).
type wsResultPayload struct {
	CasterID    int    `json:"caster_id,omitempty"`
	TargetID    int    `json:"target_id,omitempty"`
	WSName      string `json:"ws_name,omitempty"`
	Damage      int    `json:"damage,omitempty"`
	Chained     bool   `json:"chained,omitempty"`
	Resonance   string `json:"resonance,omitempty"`
	Tier        int    `json:"tier,omitempty"`
	TargetHP    int    `json:"target_hp,omitempty"`
	TargetMaxHP int    `json:"target_max_hp,omitempty"`
	Killed      bool   `json:"killed,omitempty"`
	Error       string `json:"error,omitempty"`
}

func sendWSResult(conn *net.UDPConn, remote *net.UDPAddr, result wsResultPayload) {
	b, _ := json.Marshal(result)
	pkt := append([]byte{common.PacketWSResult}, b...)
	conn.WriteToUDP(pkt, remote)
}

// respawnResultPayload is PacketRespawnResult's JSON body (backend-unification follow-up).
type respawnResultPayload struct {
	HP        int    `json:"hp,omitempty"`
	MaxHP     int    `json:"max_hp,omitempty"`
	XPPenalty int    `json:"xp_penalty,omitempty"`
	Error     string `json:"error,omitempty"`
}

func sendRespawnResult(conn *net.UDPConn, remote *net.UDPAddr, result respawnResultPayload) {
	b, _ := json.Marshal(result)
	pkt := append([]byte{common.PacketRespawnResult}, b...)
	conn.WriteToUDP(pkt, remote)
}

// placeholderPlayerDamage matches apps2/mud's own cmdWS shape (baseDamage := playerDamage*3) --
// real HP/death tracking for apps2/server-go's own connected players doesn't exist yet at all
// (clientInfo has no HP field), a separate, larger follow-up not attempted in this slice.
const placeholderPlayerDamage = 10

// resolveWSCast is the pure decision core of PacketWSCast handling, split out from the switch
// case (same "extract for testability" reasoning main_test.go's own TestParseUserCmd already
// established for parseUserCmd): validates wsName against server/skillchain's real weapon-skill
// registry, computes damage, and checks whether it closes a skillchain against whatever last
// landed on targetSlot (wsChains is read-only here -- the caller commits the returned newState
// to the map, and only on a successful cast).
//
// Returns (result, newState, true) on a real cast; (result{Error: ...}, wsChainState{}, false)
// if wsName is unknown -- the caller is responsible for every other precondition (auth, TP,
// target existence) before calling this.
// fetchCharacterCombatStats resolves the connecting player's real job/level from IDUNA and
// computes their starting HP via jobpkg.HPAtLevel -- the same formula apps2/mud's own character
// sheet already uses (server/job's own HPAtLevel, not reinvented here). Falls back to WAR/level
// 1 if IDUNA has no character row yet (a legitimately new player) or the fetch fails outright --
// a missing character shouldn't hard-reject the connection, matching PacketTelecrystalUse's own
// best-effort tone toward IDUNA lookups elsewhere in this file.
func fetchCharacterCombatStats(idunaClient *idunaclient.Client, characterID string) (jobMain string, level, maxHP, currentXP int) {
	jobMain, level = jobpkg.WAR, 1
	if ch, err := idunaClient.GetCharacter(characterID); err == nil && ch.JobMain != "" {
		jobMain = ch.JobMain
		if ch.Level >= 1 {
			level = ch.Level
		}
		if ch.CurrentXP >= 0 {
			currentXP = ch.CurrentXP
		}
	}
	hp, err := jobpkg.HPAtLevel(jobMain, level)
	if err != nil {
		jobMain, level = jobpkg.WAR, 1
		hp, _ = jobpkg.HPAtLevel(jobpkg.WAR, 1)
	}
	return jobMain, level, hp, currentXP
}

func resolveWSCast(wsName string, wsChains map[string]wsChainState, targetSlot string, casterID, targetID int, now time.Time) (wsResultPayload, wsChainState, bool) {
	ws, ok := skillchain.CanonicalWeaponSkills[wsName]
	if !ok {
		return wsResultPayload{Error: "unknown weapon skill"}, wsChainState{}, false
	}

	damage := placeholderPlayerDamage * 3
	result := wsResultPayload{CasterID: casterID, TargetID: targetID, WSName: ws.Name}

	if prev, exists := wsChains[targetSlot]; exists {
		if r, formed := skillchain.Chain(prev.Attrs, ws.Attrs, now.Sub(prev.At), skillchain.DefaultChainWindow); formed {
			damage += int(float64(damage) * r.Multiplier)
			result.Chained = true
			result.Resonance = r.Resonance.String()
			result.Tier = int(r.Tier)
		}
	}
	result.Damage = damage

	return result, wsChainState{Attrs: ws.Attrs, At: now}, true
}

func buildWorldCrisisPacket(s worldcrisis.Status) []byte {
	payload := map[string]interface{}{
		"phase":               string(s.Phase),
		"ley_integrity":       s.LeyIntegrity,
		"phase_deadline_unix": s.PhaseDeadline.Unix(),
		"outcome":             string(s.Outcome),
	}
	b, _ := json.Marshal(payload)
	return append([]byte{common.PacketWorldCrisisUpdate}, b...)
}

func sendAuthReject(conn *net.UDPConn, remote *net.UDPAddr) {
	_, _ = conn.WriteToUDP([]byte{common.PacketAuthReject}, remote)
}

func sendWelcome(conn *net.UDPConn, remote *net.UDPAddr, id uint8) {
	payload := make([]byte, 12)
	payload[0] = common.PacketWelcome
	payload[1] = id
	binary.LittleEndian.PutUint16(payload[2:], 0)
	binary.LittleEndian.PutUint32(payload[4:], uint32(time.Now().UnixMilli()))
	payload[8] = 0
	_, _ = conn.WriteToUDP(payload, remote)
}

func sendVoxelPacket(conn *net.UDPConn, remote *net.UDPAddr, info clientInfo) clientInfo {
	now := time.Now()
	if now.Sub(info.lastVoxelSent) < 500*time.Millisecond {
		return info
	}
	chunks := nearbyChunks(0, 0, 1)
	if len(chunks) == 0 {
		return info
	}
	chunk := chunks[info.chunkIndex%len(chunks)]
	blocks := scanChunkForVoxelBlocks(chunk.x, chunk.z)

	headerSize := 16
	blockSize := 6
	payload := make([]byte, headerSize+len(blocks)*blockSize)
	payload[0] = common.PacketVoxelData
	binary.LittleEndian.PutUint32(payload[4:], uint32(chunk.x))
	binary.LittleEndian.PutUint32(payload[8:], uint32(chunk.z))
	binary.LittleEndian.PutUint16(payload[12:], uint16(len(blocks)))

	offset := headerSize
	for _, blk := range blocks {
		payload[offset] = blk.x
		payload[offset+1] = blk.y
		payload[offset+2] = blk.z
		payload[offset+3] = 0
		binary.LittleEndian.PutUint16(payload[offset+4:], blk.blockID)
		offset += blockSize
	}
	_, _ = conn.WriteToUDP(payload, remote)
	info.lastVoxelSent = now
	info.chunkIndex++
	return info
}

func sendImpact(conn *net.UDPConn, remote *net.UDPAddr, pos system.Vec3, hitEntity bool, blockID uint16) {
	payload := make([]byte, 20)
	payload[0] = common.PacketImpact
	if hitEntity {
		payload[1] = 1
	}
	binary.LittleEndian.PutUint32(payload[4:], math.Float32bits(float32(pos.X)))
	binary.LittleEndian.PutUint32(payload[8:], math.Float32bits(float32(pos.Y)))
	binary.LittleEndian.PutUint32(payload[12:], math.Float32bits(float32(pos.Z)))
	binary.LittleEndian.PutUint16(payload[16:], blockID)
	_, _ = conn.WriteToUDP(payload, remote)
}

func parseUserCmd(data []byte, offset int) common.UserCmd {
	off := offset
	cmd := common.UserCmd{}
	cmd.Sequence = binary.LittleEndian.Uint32(data[off:])
	off += 4
	cmd.Timestamp = binary.LittleEndian.Uint32(data[off:])
	off += 4
	cmd.Msec = binary.LittleEndian.Uint16(data[off:])
	off += 4
	cmd.Fwd = math.Float32frombits(binary.LittleEndian.Uint32(data[off:]))
	off += 4
	cmd.Str = math.Float32frombits(binary.LittleEndian.Uint32(data[off:]))
	off += 4
	cmd.Yaw = math.Float32frombits(binary.LittleEndian.Uint32(data[off:]))
	off += 4
	cmd.Pitch = math.Float32frombits(binary.LittleEndian.Uint32(data[off:]))
	off += 4
	cmd.Buttons = binary.LittleEndian.Uint32(data[off:])
	off += 4
	cmd.WeaponIdx = int32(binary.LittleEndian.Uint32(data[off:]))
	return cmd
}

const (
	chunkSize   = 16
	chunkHeight = 16
	logBlockID  = 17
	leafBlockID = 18
)

func nearbyChunks(centerX, centerZ, radius int) []chunkCoord {
	if radius < 0 {
		return nil
	}
	chunks := make([]chunkCoord, 0, (radius*2+1)*(radius*2+1))
	for dz := -radius; dz <= radius; dz++ {
		for dx := -radius; dx <= radius; dx++ {
			chunks = append(chunks, chunkCoord{x: centerX + dx, z: centerZ + dz})
		}
	}
	return chunks
}

func scanChunkForVoxelBlocks(chunkX, chunkZ int) []voxelBlock {
	blocks := make([]uint16, chunkSize*chunkHeight*chunkSize)
	for _, tree := range treeSeedsForChunk(chunkX, chunkZ) {
		placeTree(blocks, tree.x, tree.z, tree.baseY)
	}

	results := make([]voxelBlock, 0, 64)
	for y := 0; y < chunkHeight; y++ {
		for z := 0; z < chunkSize; z++ {
			for x := 0; x < chunkSize; x++ {
				blockID := blocks[chunkIndex(x, y, z)]
				if blockID != logBlockID && blockID != leafBlockID {
					continue
				}
				results = append(results, voxelBlock{
					x:       uint8(x),
					y:       uint8(y),
					z:       uint8(z),
					blockID: blockID,
				})
			}
		}
	}
	return results
}

type treeSeed struct {
	x     int
	z     int
	baseY int
}

func treeSeedsForChunk(chunkX, chunkZ int) []treeSeed {
	switch {
	case chunkX == 0 && chunkZ == 0:
		return []treeSeed{
			{x: 8, z: 8, baseY: 0},
			{x: 3, z: 12, baseY: 0},
		}
	case chunkX == 1 && chunkZ == 0:
		return []treeSeed{
			{x: 6, z: 5, baseY: 0},
		}
	case chunkX == -1 && chunkZ == -1:
		return []treeSeed{
			{x: 11, z: 4, baseY: 0},
		}
	default:
		return nil
	}
}

func placeTree(blocks []uint16, trunkX, trunkZ, baseY int) {
	for y := 0; y < 4; y++ {
		setBlock(blocks, trunkX, baseY+y, trunkZ, logBlockID)
	}
	leafY := baseY + 4
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			setBlock(blocks, trunkX+dx, leafY, trunkZ+dz, leafBlockID)
		}
	}
	setBlock(blocks, trunkX, leafY+1, trunkZ, leafBlockID)
}

func setBlock(blocks []uint16, x, y, z int, blockID uint16) {
	if x < 0 || x >= chunkSize || y < 0 || y >= chunkHeight || z < 0 || z >= chunkSize {
		return
	}
	blocks[chunkIndex(x, y, z)] = blockID
}

func chunkIndex(x, y, z int) int {
	return (y*chunkSize+z)*chunkSize + x
}

func channelName(ch int) string {
	switch ch {
	case common.ChatSay:
		return "say"
	case common.ChatTell:
		return "tell"
	case common.ChatYell:
		return "yell"
	case common.ChatGuild:
		return "guild"
	default:
		return "unknown"
	}
}

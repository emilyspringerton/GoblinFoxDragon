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

type world struct{}

type rayResult struct {
	pos system.Vec3
}

func (r rayResult) Position() system.Vec3 { return r.pos }

func (w world) RayTrace(start, end system.Vec3) (player.RaycastResult, bool) {
	return rayResult{}, false
}

type shankPlayer struct {
	pos       system.Vec3
	eyeHeight float64
	world     world
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
	flag.Parse()

	initCityState()

	// Start the worldapi HTTP server — SHANKPIT connects here with --dragonfly-url http://localhost:7070
	if *worldapiPort > 0 {
		gen := worldapi.NewDragonflyChunkGenerator(worldapi.ProceduralWorldStore)
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

	addr, err := net.ResolveUDPAddr("udp", ":6969")
	if err != nil {
		panic(err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Go backend listening on :6969")
	authVerifier := idunaauth.NewVerifier()
	idunaClient := idunaclient.New()
	buf := make([]byte, 2048)
	p := &shankPlayer{pos: system.Vec3{}, eyeHeight: 1.62, world: world{}}
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

	for {
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
				jobMain, level, maxHP := fetchCharacterCombatStats(idunaClient, claims.Subject)
				info := clientInfo{
					id: nextClientID, playerID: claims.Subject, tp: &combatTp.TPState{},
					jobMain: jobMain, level: level, hpState: combatTp.NewHPState(maxHP),
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
			if cmd.Buttons&common.BtnAttack != 0 {
				hit, pos, hitEntity := player.HandleShankFire(p, float64(cmd.Yaw), float64(cmd.Pitch), int(cmd.WeaponIdx))
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
			// backend so far -- apps2/mud's own real "type home" flow (knockOut() +
			// HPState.Raise, 8% XP penalty) reduced to its core mechanic. Real per-player XP
			// tracking doesn't exist in apps2/server-go yet (unlike apps2/mud's own
			// p.charXP.CurrentXP), so the penalty computed here is always against 0 XP --
			// RaiseDefault(0) is a real, already-tested degenerate case (server/combat's own
			// TestRaise_ZeroXPNoPanic), not a crash risk, just an honestly-incomplete number
			// until real XP tracking lands for this backend too.
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
			penalty, err := info.hpState.RaiseDefault(0)
			if err != nil {
				sendRespawnResult(conn, remote, respawnResultPayload{Error: err.Error()})
				continue
			}
			clients[slot] = info
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
func fetchCharacterCombatStats(idunaClient *idunaclient.Client, characterID string) (jobMain string, level, maxHP int) {
	jobMain, level = jobpkg.WAR, 1
	if ch, err := idunaClient.GetCharacter(characterID); err == nil && ch.JobMain != "" {
		jobMain = ch.JobMain
		if ch.Level >= 1 {
			level = ch.Level
		}
	}
	hp, err := jobpkg.HPAtLevel(jobMain, level)
	if err != nil {
		jobMain, level = jobpkg.WAR, 1
		hp, _ = jobpkg.HPAtLevel(jobpkg.WAR, 1)
	}
	return jobMain, level, hp
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

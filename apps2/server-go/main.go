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
	"dragonsnshit/server/chat"
	"dragonsnshit/server/craft"
	"dragonsnshit/server/idunaclient"
	"dragonsnshit/server/idunaauth"
	"dragonsnshit/server/player"
	"dragonsnshit/server/store"
	"dragonsnshit/server/system"
	"dragonsnshit/server/telecrystal"
	"dragonsnshit/server/worldcrisis"
	"dragonsnshit/server/worldapi"
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
}

func main() {
	worldapiPort := flag.Int("worldapi-port", 7070, "HTTP port for the worldapi /chunks endpoint (0 = disabled)")
	flag.Parse()

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
				info := clientInfo{id: nextClientID, playerID: claims.Subject}
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
				fmt.Printf("[auth] accept %s playerID=%s id=%d\n", slot, claims.Subject, info.id)
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

func buildWorldCrisisPacket(s worldcrisis.Status) []byte {
	payload := map[string]interface{}{
		"phase":                string(s.Phase),
		"ley_integrity":        s.LeyIntegrity,
		"phase_deadline_unix":  s.PhaseDeadline.Unix(),
		"outcome":              string(s.Outcome),
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

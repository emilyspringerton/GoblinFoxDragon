package main

// voyage.go -- the Heikegani cargo ship OPEN BALANCE, which sails between Handington Docks (zone 19)
// and Open Balance Quay (zone 20) on the Kuroshio Reach. `sail` costs flow and puts you on the deck
// (zone 32) for the crossing. If you log out at sea you are put back on the home docks on your next
// tick, so nobody can be stranded aboard.

import (
	"fmt"
	"time"

	"dragonsnshit/server/mob"
)

const (
	zoneDocks  = 19
	zoneQuay   = 20
	zoneDeck   = 32
	voyageFare = 400
	voyageTime = 25 * time.Second
	reachMinLv = 35
)

var voyageDestName = map[int]string{zoneQuay: "the Kuroshio Reach", zoneDocks: "New Handington"}

// movePlayerToZone relocates p the same way walking through an exit does.
func movePlayerToZone(p *player, dest int) {
	broadcastZoneNoLock(p.zoneID, fmt.Sprintf("%s leaves.", p.name), p.slot)
	_ = gw.zoneMgr.Transfer(p.slot, dest)
	p.combat.TargetMobID = ""
	p.zoneID = dest
	p.atlas.Visit(dest)
	if z, ok := gw.zoneMgr.Get(dest); ok {
		p.pos = mob.Pos{X: z.SpawnX, Y: z.SpawnY, Z: z.SpawnZ}
	}
	syncChatSession(p)
	broadcastZoneNoLock(dest, fmt.Sprintf("%s arrives.", p.name), p.slot)
}

func cmdSail(p *player) {
	if p.homePoint.IsKO {
		p.send("You are KO'd -- type 'home' first.")
		p.prompt()
		return
	}
	var dest int
	switch p.zoneID {
	case zoneDocks:
		dest = zoneQuay
	case zoneQuay:
		dest = zoneDocks
	case zoneDeck:
		left := time.Until(p.voyageAt).Round(time.Second)
		if p.voyageTo == 0 || left <= 0 {
			p.send("The ship is nearly in. Hold on.")
		} else {
			p.sendf("The Open Balance is already underway to %s. (%s to go)", voyageDestName[p.voyageTo], left)
		}
		p.prompt()
		return
	default:
		p.send("There is no ship here. The Open Balance sails from Handington Docks (east of New Handington) and from Open Balance Quay.")
		p.prompt()
		return
	}
	if p.flow < voyageFare {
		p.sendf("The purser wants %d flow for passage. You have %d.", voyageFare, p.flow)
		p.prompt()
		return
	}
	if dest == zoneQuay && p.charXP.Level < reachMinLv && !p.sailWarned {
		p.sendf("The purser squints at you. \"The Reach is for level %d and up, friend. Your ledger says level %d.\" (You may sail anyway -- type 'sail' again to confirm.)", reachMinLv, p.charXP.Level)
		p.sailWarned = true
		p.prompt()
		return
	}
	p.sailWarned = false
	p.flow -= voyageFare
	from := zoneName(p.zoneID)
	movePlayerToZone(p, zoneDeck)
	p.voyageTo = dest
	p.voyageAt = time.Now().Add(voyageTime)
	p.voyageHalf = false
	p.sendf("You pay %d flow and board the Open Balance. The lines are cast off from %s.", voyageFare, from)
	cmdLook(p)
}

// voyageTick advances every crossing: a mid-sea message at the halfway mark, arrival at the end,
// and a free ride back to the home docks for anyone found aboard with no voyage in progress.
func voyageTick(now time.Time) {
	for _, p := range gw.players {
		if p.zoneID != zoneDeck {
			continue
		}
		if p.voyageTo == 0 {
			movePlayerToZone(p, zoneDocks)
			p.send("\r\nA deckhand steers you gently back down the gangplank: \"No voyage on your ticket, friend.\"")
			cmdLook(p)
			continue
		}
		total := voyageTime
		if !p.voyageHalf && now.After(p.voyageAt.Add(-total/2)) {
			p.voyageHalf = true
			p.send("\r\n[At sea] The accountant on deck pauses. \"We are exactly halfway,\" she says, \"and the books agree.\" Somewhere below, something large turns over in its sleep.")
			p.prompt()
		}
		if now.Before(p.voyageAt) {
			continue
		}
		dest := p.voyageTo
		p.voyageTo = 0
		movePlayerToZone(p, dest)
		p.sendf("\r\nThe Open Balance noses into the berth at %s. A clerk stamps your arrival in triplicate.", zoneName(dest))
		cmdLook(p)
	}
}

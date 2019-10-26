package gossiper

import (
	"fmt"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
)

// THIS FILE FOR THE HOMEWORK2
func (g *Gossiper) updateRouteTable(newRumor *RumorMessage, senderAddrStr string) {
	// because we have checked the diff

	// lock
	g.routeTable.Mux.Lock()
	defer g.routeTable.Mux.Unlock()

	origin := newRumor.Origin
	prevAddr, present := g.routeTable.routeTable[origin]

	// OUTPUT
	if !present || prevAddr != senderAddrStr {
		g.routeTable.routeTable[origin] = senderAddrStr
		if newRumor.Text != "" {
			fmt.Printf("DSDV %s %s \n", origin, senderAddrStr)
		}
	}
}

func (g *Gossiper) RunRoutingMessage() {
	ticker := time.NewTicker(g.rtimer)
	defer ticker.Stop()

	// Here, we choose to broadcast
	g.SendRoutingMessage(nil)

	for {
		// wait
		<-ticker.C

		fmt.Println("Send routing rumor")

		peerSelect, present := g.SelectRandomNeighbor(nil)

		if present {
			g.SendRoutingMessage(GenerateStringSetSingleton(peerSelect).ToArray())
		}
	}

}

func (g *Gossiper) SendRoutingMessage(peers []string) {
	routeMessage := g.CreateRumorPacket(&Message{Text: ""})

	g.currentID++

	if peers == nil {
		for _, peer := range peers {
			g.SendGossipPacketStrAddr(&GossipPacket{Rumor: routeMessage}, peer)
		}

		return
	}

	for _, peer := range peers {
		g.SendGossipPacketStrAddr(&GossipPacket{Rumor: routeMessage}, peer)
	}

}

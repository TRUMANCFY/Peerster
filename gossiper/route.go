package gossiper

import (
	"fmt"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
)

const HOPLIMIT = 10

func (g *Gossiper) RunRoutingMessage() {
	ticker := time.NewTicker(g.rtimer)
	defer ticker.Stop()

	// Here, we choose to broadcast
	g.SendRoutingMessage(nil)

	for {
		// wait
		<-ticker.C

		if DEBUG {
			fmt.Println("Send routing rumor")
		}

		peerSelect, present := g.SelectRandomNeighbor(nil)

		if present {
			g.SendRoutingMessage(GenerateStringSetSingleton(peerSelect).ToArray())
		}
	}

}

func (g *Gossiper) updateRouteTable(newRumor *RumorMessage, senderAddrStr string) {
	// if the roumour is from my self should end
	if newRumor.Origin == g.name {
		return
	}

	// lock
	g.routeTable.Mux.Lock()
	defer g.routeTable.Mux.Unlock()

	origin := newRumor.Origin
	prevAddr, present := g.routeTable.routeTable[origin]

	if !present || prevAddr != senderAddrStr {
		g.routeTable.routeTable[origin] = senderAddrStr
		if newRumor.Text != "" {
			// OUTPUT
			fmt.Printf("DSDV %s %s \n", origin, senderAddrStr)
		}
	}
}

func (g *Gossiper) SendRoutingMessage(peers []string) {
	routeMessage := g.CreateRumorPacket(&Message{Text: ""})

	g.AcceptRumor(routeMessage)

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

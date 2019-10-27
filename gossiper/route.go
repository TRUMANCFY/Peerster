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

		fmt.Println("Send routing rumor")

		peerSelect, present := g.SelectRandomNeighbor(nil)

		if present {
			g.SendRoutingMessage(GenerateStringSetSingleton(peerSelect).ToArray())
		}
	}

}

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

func (g *Gossiper) HandleClientPrivate(cmw *ClientMessageWrapper) {
	privateMsg := g.CreatePrivateMessage(cmw)

	g.SendPrivateMessage(privateMsg)
}

func (g *Gossiper) CreatePrivateMessage(cmw *ClientMessageWrapper) *PrivateMessage {
	return &PrivateMessage{
		Origin:      g.name,
		ID:          0,
		Text:        cmw.msg.Text,
		Destination: *cmw.msg.Destination,
		HopLimit:    HOPLIMIT,
	}
}

func (g *Gossiper) SendPrivateMessage(privateMsg *PrivateMessage) {
	// check the route table first
	dest := privateMsg.Destination

	nextNode, present := g.routeTable.routeTable[dest]

	if !present {
		fmt.Printf("Destination %s does not exist in the table \n", dest)
		return
	}

	g.SendGossipPacketStrAddr(&GossipPacket{Private: privateMsg}, nextNode)
}

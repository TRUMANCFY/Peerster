package gossiper

import (
	"fmt"
	"net"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
)

const HOPLIMIT = 10

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

func (g *Gossiper) HandlePrivatePacket(privateMsg *PrivateMessage, sender *net.UDPAddr) {
	dest := privateMsg.Destination
	if dest == g.name {
		// add the private into the list
		g.addPrivateMessage(privateMsg)
		return
	}

	hopList := privateMsg.HopLimit

	if hopList == 0 {
		fmt.Println("Hop List has been ended")
		return
	}

	nextNode, present := g.routeTable.routeTable[dest]

	if !present {
		fmt.Printf("Destination %s does not exist in the table \n", dest)
		return
	}

	// update the privatemsg
	newPrivateMsg := PrivateMessage{
		Origin:      privateMsg.Origin,
		ID:          0,
		Text:        privateMsg.Text,
		Destination: privateMsg.Destination,
		HopLimit:    privateMsg.HopLimit - 1,
	}

	g.SendGossipPacketStrAddr(&GossipPacket{Private: &newPrivateMsg}, nextNode)

}

func (g *Gossiper) addPrivateMessage(privateMsg *PrivateMessage) {
	g.privateMessageList.Mux.Lock()
	defer g.privateMessageList.Mux.Unlock()

	origin := privateMsg.Origin
	content := privateMsg.Text
	hopLimit := privateMsg.HopLimit

	// OUTPUT
	fmt.Printf("PRIVATE origin %s hop-limit %d contents %s \n", origin, hopLimit, content)

	_, present := g.privateMessageList.privateMessageList[origin]

	if !present {
		g.privateMessageList.privateMessageList[origin] = make([]PrivateMessage, 0)
	}

	msgs := g.privateMessageList.privateMessageList[origin]

	msgs = append(msgs, *privateMsg)

	g.privateMessageList.privateMessageList[origin] = msgs

}

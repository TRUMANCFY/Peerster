package gossiper

import (
	"fmt"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	// . "github.com/TRUMANCFY/Peerster/util"
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

		if DEBUGROUTE {
			fmt.Println("Send routing rumor")
		}

		// peerSelect, present := g.SelectRandomNeighbor(nil)

		// if present {
		// 	g.SendRoutingMessage(GenerateStringSetSingleton(peerSelect).ToArray())
		// }
		g.SendRoutingMessage(nil)
	}

}

func (g *Gossiper) updateRouteTable(gp *GossipPacket, senderAddrStr string) {
	// if tlc message also need to update the tlc
	// if gp.TLCMessage != nil {
	// 	return
	// }

	var origin string
	var id uint32

	if gp.Rumor != nil {
		origin = gp.Rumor.Origin
		id = gp.Rumor.ID
	} else if gp.TLCMessage != nil {
		origin = gp.TLCMessage.Origin
		id = gp.TLCMessage.ID
	}

	// No update
	if origin == g.name {
		return
	}

	// lock
	g.routeTable.Mux.Lock()
	defer g.routeTable.Mux.Unlock()

	prevAddr, present := g.routeTable.routeTable[origin]

	if !present {
		g.routeTable.routeTable[origin] = senderAddrStr
		g.routeTable.IDTable[origin] = id
		if gp.Rumor != nil {
			if gp.Rumor.Text != "" {
				// OUTPUT
				fmt.Printf("DSDV %s %s \n", origin, senderAddrStr)
				// fmt.Printf("Text is %s \n", newRumor.Text)
			}
		}
		return
	}

	prev_id, _ := g.routeTable.IDTable[origin]

	if prev_id < id {
		if DEBUGROUTE {
			fmt.Printf("Origin: %s ID %d \n", origin, id)
		}
		g.routeTable.IDTable[origin] = id
		if prevAddr != senderAddrStr {
			g.routeTable.routeTable[origin] = senderAddrStr

			if DEBUGROUTE {
				g.PrintPeers()
				fmt.Printf("Route Rumor Origin: %s, ID: %d from %s \n", origin, id, senderAddrStr)
				fmt.Println(g.routeTable.routeTable)
			}

			if gp.Rumor != nil {
				if gp.Rumor.Text != "" {
					// OUTPUT
					fmt.Printf("DSDV %s %s \n", origin, senderAddrStr)
				}
			}

		}
	}
}

func (g *Gossiper) SendRoutingMessage(peers []string) {
	routeMessage := g.CreateRumorPacket(&Message{Text: ""})

	g.AcceptRumor(&GossipPacket{Rumor: routeMessage})

	if peers == nil {
		g.peersList.Mux.Lock()
		localPeers := g.peersList.PeersList.ToArray()

		for _, peer := range localPeers {
			g.SendGossipPacketStrAddr(&GossipPacket{Rumor: routeMessage}, peer)
		}

		g.peersList.Mux.Unlock()

		return
	}

	for _, peer := range peers {
		g.SendGossipPacketStrAddr(&GossipPacket{Rumor: routeMessage}, peer)
	}

}

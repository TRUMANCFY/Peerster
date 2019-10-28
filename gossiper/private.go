package gossiper

import (
	"fmt"
	"net"

	. "github.com/TRUMANCFY/Peerster/message"
)

func (g *Gossiper) HandlePrivatePacket(privateMsg *PrivateMessage, sender *net.UDPAddr) {
	dest := privateMsg.Destination
	if dest == g.name {
		// add the private into the list
		g.addPrivateMessage(privateMsg)
		return
	}

	hopLimit := privateMsg.HopLimit

	if hopLimit == 0 {
		fmt.Println("Hop Limit has been ended")
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

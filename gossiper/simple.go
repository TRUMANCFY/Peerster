package gossiper

import (
	// check the type of variable

	"fmt"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
)

func (g *Gossiper) HandleSimplePacket(s *SimpleMessage) {
	// fmt.Println("Deal with simple packet")
	gp := &GossipPacket{Simple: g.CreateForwardPacket(s)}

	// has already add as soon as receive the packet
	g.peersList.Mux.Lock()
	g.peersList.PeersList.Add(s.RelayPeerAddr)
	g.peersList.Mux.Unlock()

	// if g.CheckSimpleMsg(s) {
	// 	fmt.Println("Already have simple message")
	// 	return
	// }
	g.BroadcastPacket(gp, GenerateStringSetSingleton(s.RelayPeerAddr))
}

func (g *Gossiper) CheckSimpleMsg(s *SimpleMessage) bool {
	g.simpleListLock.Lock()
	defer g.simpleListLock.Unlock()

	origin := s.OriginalName
	content := s.Contents
	_, present := g.simpleList[origin]

	if !present {
		g.simpleList[origin] = make(map[string]bool)
		g.simpleList[origin][content] = true
		return false
	}

	contentMap := g.simpleList[origin]

	_, present = contentMap[content]

	if !present {
		g.simpleList[origin][content] = true
		return false
	}

	return true
}

func (g *Gossiper) BroadcastPacket(gp *GossipPacket, excludedPeers *StringSet) {
	g.peersList.Mux.Lock()
	defer g.peersList.Mux.Unlock()

	fmt.Println(g.peersList.PeersList.ToArray())

	for _, p := range g.peersList.PeersList.ToArray() {
		if excludedPeers == nil || !excludedPeers.Has(p) {
			fmt.Printf("Send to %s Origin: %s Message: %s \n", p, gp.Simple.OriginalName, gp.Simple.Contents)
			g.SendGossipPacketStrAddr(gp, p)
		}
	}
}

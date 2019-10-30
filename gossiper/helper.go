package gossiper

import (
	// check the type of variable

	"fmt"
	"math/rand"
	"net"
	"strings"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
	"github.com/dedis/protobuf"
)

func (g *Gossiper) SelectRandomNeighbor(excludedPeer *StringSet) (string, bool) {
	g.peersList.Mux.Lock()
	peers := g.peersList.PeersList.ToArray()
	g.peersList.Mux.Unlock()
	notExcluded := make([]string, 0)

	for _, peer := range peers {
		if excludedPeer == nil || !excludedPeer.Has(peer) {
			notExcluded = append(notExcluded, peer)
		}
	}

	if len(notExcluded) == 0 {
		return "", false
	}

	return notExcluded[rand.Intn(len(notExcluded))], true
}

func (g *Gossiper) SendClientAck(client *net.UDPAddr) {
	resp := &Message{Text: "Ok"}
	packetBytes, err := protobuf.Encode(resp)

	if err != nil {
		panic(err)
	}

	_, err = g.uiConn.WriteToUDP(packetBytes, client)
}

func (g *Gossiper) CreateForwardPacket(m *SimpleMessage) *SimpleMessage {
	return &SimpleMessage{
		OriginalName:  m.OriginalName,
		RelayPeerAddr: g.address.String(),
		Contents:      m.Contents,
	}
}

func (g *Gossiper) CreateClientPacket(m *Message) *SimpleMessage {
	return &SimpleMessage{
		OriginalName:  g.name,
		RelayPeerAddr: g.address.String(),
		Contents:      m.Text,
	}
}

func (g *Gossiper) CreateRumorPacket(m *Message) *RumorMessage {
	g.currentID.Mux.Lock()
	defer g.currentID.Mux.Unlock()

	res := &RumorMessage{
		Origin: g.name,
		ID:     g.currentID.currentID,
		Text:   m.Text,
	}

	g.currentID.currentID++

	return res
}

func (g *Gossiper) CreateStatusPacket() *GossipPacket {
	// TODO concurrent map iteration and map write

	wantSlice := make([]PeerStatus, 0)

	for _, ps := range g.peerStatuses {
		wantSlice = append(wantSlice, ps)
	}

	sp := &StatusPacket{Want: wantSlice}

	return sp.ToGossipPacket()
}

func (g *Gossiper) PrintPeers() {
	g.peersList.Mux.Lock()
	defer g.peersList.Mux.Unlock()
	fmt.Printf("PEERS %s\n", strings.Join(g.peersList.PeersList.ToArray(), ","))
}

func (g *Gossiper) AddPeer(p string) {
	g.peersList.Mux.Lock()
	g.peersList.PeersList.Add(p)
	g.peersList.Mux.Unlock()
}

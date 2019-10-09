package message

import (
	"fmt"
	"strings"
)

// N2N communication

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor  *RumorMessage
	Status *StatusPacket
}

type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

type PeerStatus struct {
	Identifier string
	NextID     uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

// client to node message
type Message struct {
	Text string
}

// String member function is actually overrided method
// please refer to https://stackoverflow.com/questions/13247644/tostring-function-in-go
func (sm *SimpleMessage) String() string {
	return fmt.Sprintf("SIMPLE MESSAGE origin %s from %s contents %s", sm.OriginalName, sm.RelayPeerAddr, sm.Contents)
}

func (m *Message) String() string {
	return fmt.Sprintf("CLIENT MESSAGE %s", m.Text)
}

func (rm *RumorMessage) SenderString(sender string) string {
	return fmt.Sprintf("RUMOR MESSAGE origin %s from %s ID %d contents %s", rm.Origin, sender, rm.ID, rm.Text)
}

func (sp *StatusPacket) SenderString(sender string) string {
	wantString := make([]string, 0)
	for _, ps := range sp.Want {
		wantString = append(wantString, ps.String())
	}
	return fmt.Sprintf("STATUS from %s, %s", sender, strings.Join(wantString, " "))
}

func (ps *PeerStatus) String() string {
	return fmt.Sprintf("peer %s next %d", ps.Identifier, ps.NextID)
}

// convert the member element to gossip packet
func (gp *GossipPacket) ToGossipPacket() *GossipPacket {
	return gp
}

func (sm *SimpleMessage) ToGossipPacket() *GossipPacket {
	return &GossipPacket{Simple: sm}
}

func (rm *RumorMessage) ToGossipPacket() *GossipPacket {
	return &GossipPacket{Rumor: rm}
}

func (sp *StatusPacket) ToGossipPacket() *GossipPacket {
	return &GossipPacket{Status: sp}
}

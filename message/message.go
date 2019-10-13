package message

import (
	"fmt"
	"strings"
)

// N2N communication

// Status Packet

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

// client to node message
type Message struct {
	Text string
}

const (
	antiEntropy   StatusPacketType = 0
	rumorResponse StatusPacketType = 1
)

type StatusPacket struct {
	Want []PeerStatus
	// spType StatusPacketType
}

type StatusPacketType int

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor  *RumorMessage
	Status *StatusPacket
}

// advanced data structure

// type GossipPacketWrapper struct {
// 	sender       *net.UDPAddr
// 	gossipPacket *GossipPacket
// }

// type ClientMessageWrapper struct {
// 	sender *net.UDPAddr
// 	msg    *Message
// }

// type MessageReceived struct {
// 	sender        *net.UDPAddr
// 	packetContent []byte
// }

// type RegisterMessageType int

// const (
// 	Register   RegisterMessageType = 0
// 	Unregister RegisterMessageType = 1
// )

// type PeerStatusObserver (chan<- PeerStatus)

// type RegisterMessage struct {
// 	observerChan    PeerStatusObserver
// 	tagger          StatusTagger
// 	registerMsgType RegisterMessageType
// }

// type StatusTagger struct {
// 	sender     string
// 	identifier string
// }

// type PeerStatusWrapper struct {
// 	sender       string
// 	peerStatuses []PeerStatus
// }

// type Dispatcher struct {
// 	statusListener   chan PeerStatusWrapper
// 	registerListener chan RegisterMessage
// }

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
	return fmt.Sprintf("STATUS from %s %s", sender, strings.Join(wantString, " "))
}

func (ps *PeerStatus) String() string {
	return fmt.Sprintf("peer %s nextID %d", ps.Identifier, ps.NextID)
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

package message

import (
	"crypto/sha256"
	"encoding/binary"
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
	Text        string
	Destination *string
	File        *string
	Request     *[]byte
	Keywords    *string
	Budget      *uint64
}

type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
}

type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}

// Actually, we donot have to be confused with the different types of status packet

// type StatusPacketType int

// const (
// 	antiEntropy   StatusPacketType = 0
// 	rumorResponse StatusPacketType = 1
// )

type StatusPacket struct {
	Want []PeerStatus
	// spType StatusPacketType
}

type SearchRequest struct {
	Origin   string
	Budget   uint64
	Keywords []string
}

type SearchReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	Results     []*SearchResult
}

type SearchResult struct {
	FileName     string
	MetafileHash []byte
	ChunkMap     []uint64
	ChunkCount   uint64
}

// hw3ex2
type TxPublish struct {
	Name         string
	Size         int64 // this is a size in bytes
	MetafileHash []byte
}

type BlockPublish struct {
	PrevHash    [32]byte
	Transaction TxPublish
}

type TLCMessage struct {
	Origin      string
	ID          uint32
	Confirmed   uint32
	TxBlock     BlockPublish
	VectorClock *StatusPacket
	Fitness     float32
}

type TLCAck PrivateMessage

type GossipPacket struct {
	Simple        *SimpleMessage
	Rumor         *RumorMessage
	Status        *StatusPacket
	Private       *PrivateMessage
	DataRequest   *DataRequest
	DataReply     *DataReply
	SearchRequest *SearchRequest
	SearchReply   *SearchReply
	TLCMessage    *TLCMessage
	Ack           *TLCAck
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

func (b *BlockPublish) Hash() (out [32]byte) {
	h := sha256.New()
	h.Write(b.PrevHash[:])
	th := b.Transaction.Hash()
	h.Write(th[:])
	copy(out[:], h.Sum(nil))
	return
}

func (t *TxPublish) Hash() (out [32]byte) {
	h := sha256.New()
	binary.Write(h, binary.LittleEndian, uint32(len(t.Name)))
	h.Write([]byte(t.Name))
	h.Write(t.MetafileHash)
	copy(out[:], h.Sum(nil))
	return
}

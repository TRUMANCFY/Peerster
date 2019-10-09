package main

import (
	"flag"
	"fmt" // check the type of variable
	"net"
	"strings"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
	"github.com/dedis/protobuf"
)

const UDP_DATAGRAM_MAX_SIZE = 1024
const CHANNEL_BUFFER_SIZE = 1024
const STATUS_MESSAGE_TIMEOUT = 10 * time.Second

// Memory arrangement
// Think about the process and what dataframe do we need
// 1. we need a list contain self peerstatus for sure, data format: map[string]PeerStatus peerName => PeerStatus
// 2. We need to record the peer status we have received from other peers, data format: map[string](map[string]PeerStatus) peerName => peerName => PeerStatus
// 3. Also, we need to record the rumour we are maintain data format: map[string](map[int]RumorMessage) peerName => sequential number => rumorMessage

// GoRoutine Handler
// 1. we need one routine to receive GossipPacket from peers: func ReceiveFromPeers()
// 2. we need one routine to receive Message from clients: func ReceiveFromClients()
// 3.

type PeerStatusObserver (chan<- PeerStatus)

type Gossiper struct {
	address      *net.UDPAddr
	conn         *net.UDPConn
	uiAddr       *net.UDPAddr
	uiConn       *net.UDPConn
	name         string
	peersList    *StringSet
	simple       bool
	peerStatuses map[string]PeerStatus
	peerWantList map[string](map[string]PeerStatus)
	rumorList    map[string](map[uint32]RumorMessage)
	simpleList   map[string]([]SimpleMessage)
	brain        *Handler
	currentID    uint32
}

type Handler struct {
	input chan PeerStatusObserver
}

// advance and combined data structure

type GossipPacketWrapper struct {
	sender       *net.UDPAddr
	gossipPacket *GossipPacket
}

type ClientMessageWrapper struct {
	sender *net.UDPAddr
	msg    *Message
}

type MessageReceived struct {
	sender        *net.UDPAddr
	packetContent []byte
}

var uiPort = flag.String("UIPort", "8001", "please provide UI Port")
var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "please provide gossip address")
var name = flag.String("name", "nodeA", "please provide the node name")
var peersStr = flag.String("peers", "127.0.0.1:5001, 10.1.1.7:5002", "please provide the peers")
var simple = flag.Bool("simple", false, "type")

func main() {

	fmt.Println("Gossiper")
	flag.Parse()
	fmt.Printf("UI port is %s \n", *uiPort)
	fmt.Printf("Gossip address is %s \n", *gossipAddr)
	fmt.Printf("Gossip name is %s \n", *name)
	fmt.Printf("Gossip peers are %s \n", *peersStr)
	fmt.Printf("Simple mode is %t \n", *simple)

	// Split the peers to list
	peersList := GenerateStringSet(strings.Split(*peersStr, ","))

	gossiper := NewGossiper(*gossipAddr, *uiPort, *name, peersList, *simple)

	go gossiper.ListenClientMsg()
	gossiper.ListenNodeMsg()

}

func NewGossiper(gossipAddr string, uiPort string, name string, peersStr *StringSet, simple bool) *Gossiper {
	// gossip
	udpAddr, err := net.ResolveUDPAddr("udp4", gossipAddr)

	if err != nil {
		panic(fmt.Sprintln("Error when creating udpAddr", err))
	}

	udpConn, err := net.ListenUDP("udp4", udpAddr)

	if err != nil {
		panic(fmt.Sprintln("Error when creating udpConn", err))
	}

	// ui
	uiAddr := fmt.Sprintf("127.0.0.1:%s", uiPort)
	udpUIAddr, err := net.ResolveUDPAddr("udp4", uiAddr)
	if err != nil {
		panic(fmt.Sprintln("Error when creating udpUIAddr", err))
	}
	udpUIConn, err := net.ListenUDP("udp4", udpUIAddr)
	if err != nil {
		panic(fmt.Sprintln("Error when creating udpUIConn", err))
	}

	fmt.Println("Created")

	return &Gossiper{
		address:      udpAddr,
		conn:         udpConn,
		uiAddr:       udpUIAddr,
		uiConn:       udpUIConn,
		name:         name,
		peersList:    peersStr,
		simple:       simple,
		peerStatuses: make(map[string]PeerStatus),
		peerWantList: make(map[string](map[string]PeerStatus)),
		rumorList:    make(map[string](map[uint32]RumorMessage)),
		simpleList:   make(map[string]([]SimpleMessage)),
		brain:        nil,
		currentID:    0,
	}
}

func (g *Gossiper) Run() {
	peerListener := g.ReceiveFromPeers()
	clientListener := g.ReceiveFromClients()

	g.Listen(peerListener, clientListener)
}

func (g *Gossiper) Listen(peerListener <-chan *GossipPacketWrapper, clientListener <-chan *ClientMessageWrapper) {
	for {
		select {
		case gpw := <-peerListener:
			g.HandlePeerMessage(gpw)
		case cmw := <-clientListener:
			g.HandleClientMessage(cmw)
		}

	}
}

func (g *Gossiper) HandlePeerMessage(gpw *GossipPacketWrapper) {
	packet := gpw.gossipPacket
	sender := gpw.sender

	g.AddPeer(sender.String())

	switch {
	case packet.Simple != nil:
		fmt.Println(packet.Simple)
		g.HandleSimplePacket(packet.Simple)
	case packet.Rumor != nil:
		fmt.Println(packet.Rumor.SenderString(sender.String()))
		g.HandleRumorPacket(packet.Rumor)
	case packet.Status != nil:
		fmt.Println(packet.Status.SenderString(sender.String()))
		g.HandleStatusPacket(packet.Status)
	}
}

func (g *Gossiper) HandleClientMsg(msg *Message) {
	fmt.Println(msg)
	g.PrintPeers()
	newMsg := g.CreateClientPacket(msg)
	newGossipPacket := &GossipPacket{Simple: newMsg}

	// we need to exclude the node sending info
	g.BroadcastPacket(newGossipPacket, nil)
}

func (g *Gossiper) HandleClientMessage(cmw *ClientMessageWrapper) {
	msg := cmw.msg
	fmt.Println(msg)

	if g.simple {
		newMsg := g.CreateClientPacket(msg)
		newGossipPacket := &GossipPacket{Simple: newMsg}

		g.BroadcastPacket(newGossipPacket, nil)
	} else {
		fmt.Println("Create Rumor")
		newRumorMsg := g.CreateRumorPacket(msg)
		g.currentID++
		g.HandleRumorPacket(newRumorMsg, g.address)

	}
}

func (g *Gossiper) HandleSimplePacket(s *SimpleMessage) {

}

func (g *Gossiper) HandleRumorPacket(r *RumorMessage, senderAddr *net.UDPAddr) {
	// diff == 0 => accept the rumor as rumor is rightly updated
	// diff > 0 => drop it (should we keep this for the future reference???)
	// diff < 0 => drop it
	// if sender is self, broadcast (mongering) the rumor
	diff := g.RumorStatusCheck(r)

	fmt.Printf("The difference between the comming rumor and current peerstatus is %d \n", diff)

	switch {
	case diff == 0:
		// accept the rumor
		g.AcceptRumor(r)
	case diff > 0:
		fmt.Println("The rumor is ahead of our record")
	case diff < 0:
		fmt.Println("The rumor is behind our record")
	}
	// broadcast
	if senderAddr == g.address {

	}
}

func (g *Gossiper) HandleStatusPacket(s *StatusPacket) {

}

func (g *Gossiper) RumorStatusCheck(r *RumorMessage) int {
	// To Check the status of the rumor message
	// If diff == 0, rightly updated, therefore
	// If diff > 0, the rumorMessage is head of the record peerStatus
	// if diff < 0, the rumorMessage is behind the record peerStatus
	peerStatus, ok := g.peerStatuses[r.Origin]

	if !ok {
		peerStatus := PeerStatus{
			Identifier: r.Origin,
			NextID:     0,
		}
		g.peerStatuses[r.Origin] = peerStatus
	}

	return int(peerStatus.NextID) - int(r.ID)

}

func (g *Gossiper) AcceptRumor(r *RumorMessage) {
	// 1. put the rumor in the list
	// 2. update the peer status
}

func (g *Gossiper) ReceiveFromPeers() <-chan *GossipPacketWrapper {
	res := make(chan *GossipPacketWrapper, CHANNEL_BUFFER_SIZE)
	messageReceiver := MessageReceive(g.conn)

	go func() {
		for {
			var packetReceived GossipPacket
			msg := <-messageReceiver
			protobuf.Decode(msg.packetContent, packetReceived)
			res <- &GossipPacketWrapper{sender: msg.sender, gossipPacket: &packetReceived}
		}
	}()

	return res
}

func (g *Gossiper) ReceiveFromClients() <-chan *ClientMessageWrapper {
	res := make(chan *ClientMessageWrapper, CHANNEL_BUFFER_SIZE)
	messageReceiver := MessageReceive(g.uiConn)

	go func() {
		for {
			var packetReceived Message
			msg := <-messageReceiver
			protobuf.Decode(msg.packetContent, packetReceived)
			res <- &ClientMessageWrapper{sender: msg.sender, msg: &packetReceived}
		}
	}()
	return res
}

func MessageReceive(conn *net.UDPConn) <-chan *MessageReceived {
	res := make(chan *MessageReceived, CHANNEL_BUFFER_SIZE)
	go func() {
		packageBytes := make([]byte, UDP_DATAGRAM_MAX_SIZE)
		_, sender, _ := conn.ReadFromUDP(packageBytes)
		res <- &MessageReceived{sender: sender, packetContent: packageBytes}
	}()

	return res
}

func (g *Gossiper) HandleNodeMsg(gossipPacket *GossipPacket) {
	fmt.Println(gossipPacket)
	g.PrintPeers()
	newMsg := g.CreateForwardPacket(gossipPacket.Simple)
	newGossipPacket := &GossipPacket{Simple: newMsg}

	// add to the peers
	g.AddPeer(gossipPacket.Simple.RelayPeerAddr)

	g.BroadcastPacket(newGossipPacket, GenerateStringSetSingleton(gossipPacket.Simple.RelayPeerAddr))
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
	return &RumorMessage{
		Origin: g.name,
		ID:     g.currentID,
		Text:   m.Text,
	}
}

func (g *Gossiper) BroadcastPacket(gp *GossipPacket, excludedPeers *StringSet) {
	for _, p := range g.peersList.ToArray() {
		if excludedPeers == nil || !excludedPeers.Has(p) {
			g.SendPacket(gp, p)
		}
	}
}

func (g *Gossiper) SendPacket(gp *GossipPacket, peerAddr string) {
	packetBytes, err := protobuf.Encode(gp)

	if err != nil {
		fmt.Println(err)
	}

	conn, err := net.Dial("udp4", peerAddr)

	if err != nil {
		fmt.Println(err)
	}

	conn.Write(packetBytes)
}

func (g *Gossiper) PrintPeers() {
	fmt.Printf("PEERS %s\n", strings.Join(g.peersList.ToArray(), ","))
}

func (g *Gossiper) AddPeer(p string) {
	g.peersList.Add(p)
}

// backup code let's see

func (g *Gossiper) ListenClientMsg() {
	msgBytes := make([]byte, UDP_DATAGRAM_MAX_SIZE)
	// init the data structure
	var msg Message

	for {
		g.uiConn.ReadFromUDP(msgBytes)
		protobuf.Decode(msgBytes, &msg)
		g.HandleClientMsg(&msg)
	}
}

func (g *Gossiper) ListenNodeMsg() {
	gossipBytes := make([]byte, UDP_DATAGRAM_MAX_SIZE)
	// init the data structure
	var gossipPacket GossipPacket

	for {
		g.conn.ReadFromUDP(gossipBytes)
		protobuf.Decode(gossipBytes, &gossipPacket)
		g.HandleNodeMsg(&gossipPacket)
	}
}

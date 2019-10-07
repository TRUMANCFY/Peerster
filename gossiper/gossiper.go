package main

import (
	"flag"
	"fmt" // check the type of variable
	"net"
	"strings"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
	"github.com/dedis/protobuf"
)

var uiPort = flag.String("UIPort", "8001", "please provide UI Port")
var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "please provide gossip address")
var name = flag.String("name", "nodeA", "please provide the node name")
var peersStr = flag.String("peers", "127.0.0.1:5001,10.1.1.7:5002", "please provide the peers")
var simple = flag.Bool("simple", false, "type")

const UDP_DATAGRAM_MAX_SIZE = 1024

type Gossiper struct {
	address   *net.UDPAddr
	conn      *net.UDPConn
	uiAddr    *net.UDPAddr
	uiConn    *net.UDPConn
	name      string
	peersList *StringSet
	simple    bool
}

func main() {
	flag.Parse()
	fmt.Printf("UI port is %s \n", *uiPort)
	fmt.Printf("Gossip address is %s \n", *gossipAddr)
	fmt.Printf("Gossip name is %s \n", *name)
	fmt.Printf("Gossip peers are %s \n", *peersStr)
	fmt.Printf("Simple mode is %t \n", *simple)

	// Split the peers to list
	peersList := GenerateStringSet(strings.Split(*peersStr, ","))

	tester := NewGossiper(*gossipAddr, *uiPort, *name, peersList, *simple)

	newPackageBypes := make([]byte, UDP_DATAGRAM_MAX_SIZE)

	tester.conn.ReadFromUDP(newPackageBypes)

	var newMsg Message

	protobuf.Decode(newPackageBypes, &newMsg)

	fmt.Println("Receive new")
	fmt.Println(newMsg)

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

	return &Gossiper{
		address:   udpAddr,
		conn:      udpConn,
		uiAddr:    udpUIAddr,
		uiConn:    udpUIConn,
		name:      name,
		peersList: peersStr,
		simple:    simple,
	}
}

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

func (g *Gossiper) HandleClientMsg(msg *Message) {
	fmt.Println(msg)
	newMsg := g.CreateClientPacket(msg)
	newGossipPacket := &GossipPacket{Simple: newMsg}

	// we need to exclude the node sending info
	g.BroadcastPacket(newGossipPacket, nil)
}

func (g *Gossiper) HandleNodeMsg(gossipPacket *GossipPacket) {
	fmt.Println(gossipPacket)
	newMsg := g.CreateForwardPacket(gossipPacket.Simple)
	newGossipPacket := &GossipPacket{Simple: newMsg}

	// add to the peers
	g.peersList.Add(gossipPacket.Simple.RelayPeerAddr)

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

func (g *Gossiper) BroadcastPacket(gp *GossipPacket, excludedPeers *StringSet) {
	for _, p := range g.peersList.ToArray() {
		if excludedPeers == nil || excludedPeers.Has(p) {
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

package main

import (
	"flag"
	"fmt" // check the type of variable
	"math/rand"
	"net"
	"strings"
	"sync"
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

// GoRoutine Dispatcher
// 1. we need one routine to receive GossipPacket from peers: func ReceiveFromPeers()
// 2. we need one routine to receive Message from clients: func ReceiveFromClients()
// 3.
type Gossiper struct {
	address          *net.UDPAddr
	conn             *net.UDPConn
	uiAddr           *net.UDPAddr
	uiConn           *net.UDPConn
	name             string
	peersList        *StringSet
	simple           bool
	peerStatuses     map[string]PeerStatus
	peerWantList     map[string](map[string]PeerStatus)
	peerWantListLock *sync.RWMutex
	rumorList        map[string](map[uint32]RumorMessage)
	rumorListLock    *sync.RWMutex
	simpleList       map[string]([]SimpleMessage)
	dispatcher       *Dispatcher
	currentID        uint32
	toSendChan       chan *GossipPacketWrapper
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

type RegisterMessageType int

const (
	Register   RegisterMessageType = 0
	Unregister RegisterMessageType = 1
)

type PeerStatusObserver (chan<- PeerStatus)

type RegisterMessage struct {
	observerChan    PeerStatusObserver
	tagger          StatusTagger
	registerMsgType RegisterMessageType
}

type StatusTagger struct {
	sender     string
	identifier string
}

type PeerStatusWrapper struct {
	sender       string
	peerStatuses []PeerStatus
}

type Dispatcher struct {
	statusListener   chan PeerStatusWrapper
	registerListener chan RegisterMessage
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

	gossiper.Run()
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
		address:          udpAddr,
		conn:             udpConn,
		uiAddr:           udpUIAddr,
		uiConn:           udpUIConn,
		name:             name,
		peersList:        peersStr,
		simple:           simple,
		peerStatuses:     make(map[string]PeerStatus),
		peerWantList:     make(map[string](map[string]PeerStatus)),
		peerWantListLock: &sync.RWMutex{},
		rumorList:        make(map[string](map[uint32]RumorMessage)),
		rumorListLock:    &sync.RWMutex{},
		simpleList:       make(map[string]([]SimpleMessage)),
		currentID:        0,
		dispatcher:       nil,
		toSendChan:       make(chan *GossipPacketWrapper, CHANNEL_BUFFER_SIZE),
	}
}

func (g *Gossiper) Run() {
	peerListener := g.ReceiveFromPeers()
	clientListener := g.ReceiveFromClients()

	g.dispatcher = StartPeerStatusDispatcher()
	g.Listen(peerListener, clientListener)
}

func (g *Gossiper) Listen(peerListener <-chan *GossipPacketWrapper, clientListener <-chan *ClientMessageWrapper) {
	for {
		select {
		case gpw := <-peerListener:
			g.HandlePeerMessage(gpw)
		case cmw := <-clientListener:
			g.HandleClientMessage(cmw)
		case gpw := <-g.toSendChan:
			gp := gpw.gossipPacket
			peerUDPAddr := gpw.sender
			packetBytes, err := protobuf.Encode(gp)

			if err != nil {
				fmt.Println(err)
			}

			_, err = g.conn.WriteToUDP(packetBytes, peerUDPAddr)

			if err != nil {
				fmt.Println(err)
			}

		}
	}
}

func StartPeerStatusDispatcher() *Dispatcher {
	// TODO: whether the statusListener and registerListener should be buffered or unbuffered
	sListener := make(chan PeerStatusWrapper, CHANNEL_BUFFER_SIZE)
	rListener := make(chan RegisterMessage, CHANNEL_BUFFER_SIZE)

	dispatch := &Dispatcher{statusListener: sListener, registerListener: rListener}
	go dispatch.Run()
	return dispatch
}

func (d *Dispatcher) Run() {
	taggers := make(map[StatusTagger](map[PeerStatusObserver]bool))
	observers := make(map[PeerStatusObserver]StatusTagger)

	for {
		select {
		case registerMsg := <-d.registerListener:
			switch registerMsg.registerMsgType {
			case Register:
				tagger := registerMsg.tagger

				psoMap, present := taggers[tagger]

				if !present {
					taggers[tagger] = make(map[PeerStatusObserver]bool)
				}

				// the boolean value "true" is useless for this case
				psoMap[registerMsg.observerChan] = true
				observers[registerMsg.observerChan] = tagger

			case Unregister:
				// Close the channel
				toClosedObserver := registerMsg.observerChan
				tagger, present := observers[toClosedObserver]

				if present {
					// remove tagger from the local mapping
					delete(taggers[tagger], toClosedObserver)
					delete(observers, toClosedObserver)

					// close the channel
					close(toClosedObserver)
				} else {
					panic(fmt.Sprintf("Origin: %s; Sender: %s NOT REGISTERED", registerMsg.tagger.identifier, registerMsg.tagger.sender))

				}

			}
		case comingPeerStatusWrapper := <-d.statusListener:
			// peerStatusWrapper : {sender string; peerStatuses []PeerStatus}
			// the peerstatus of another peer
			sender := comingPeerStatusWrapper.sender
			comingPeerStatuses := comingPeerStatusWrapper.peerStatuses

			for _, peerStatus := range comingPeerStatuses {
				tagger := StatusTagger{sender: sender, identifier: peerStatus.Identifier}
				// taggers: map[StatusTagger](map[PeerStatusObserver]bool)

				// it will returns the channel which the target is sender, and the origin is identifier
				activeChans, present := taggers[tagger]

				if !present {
					continue
				}

				// activeChans is map[PeerStatusObserver]bool
				// add the coming peerStauts into the activeChan
				for activeChan, _ := range activeChans {
					// observerChan is renamed as activeChan here
					activeChan <- peerStatus
				}
			}
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
		g.HandleSimplePacket(packet.Simple, sender.String())
	case packet.Rumor != nil:
		fmt.Println(packet.Rumor.SenderString(sender.String()))
		g.HandleRumorPacket(packet.Rumor, sender)
	case packet.Status != nil:
		fmt.Println(packet.Status.SenderString(sender.String()))
		g.HandleStatusPacket(packet.Status, sender)
	}
}

// func (g *Gossiper) HandleClientMsg(msg *Message) {
// 	fmt.Println(msg)
// 	g.PrintPeers()
// 	newMsg := g.CreateClientPacket(msg)
// 	newGossipPacket := &GossipPacket{Simple: newMsg}

// 	// we need to exclude the node sending info
// 	g.BroadcastPacket(newGossipPacket, nil)
// }

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
		// ID count increase
		g.currentID++
		g.HandleRumorPacket(newRumorMsg, g.address)
	}
}

func (g *Gossiper) HandleSimplePacket(s *SimpleMessage, sender string) {
	fmt.Println("Deal with simple packet")
	gp := &GossipPacket{Simple: s}

	// has already add as soon as receive the packet
	g.peersList.Add(sender)

	g.BroadcastPacket(gp, GenerateStringSetSingleton(sender))
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
		// TODO: as soon as received, mongering should start
		// TODO: update the table, maybe have been done in acceptrumor function
		g.AcceptRumor(r)
	case diff > 0:
		fmt.Println("The rumor is ahead of our record")
	case diff < 0:
		fmt.Println("The rumor is behind our record")
	}
	// broadcast to other peer
	if senderAddr != g.address {
		// TODO send the status message to sender
	}
}

func (g *Gossiper) RumorMongeringPrepare(rumor *RumorMessage, excludedPeers *StringSet) (string, bool) {
	randomNeighbor, present := g.SelectRandomNeighbor(excludedPeers)

	if present {
		g.RumorMongeringAddrStr(rumor, randomNeighbor)
	}

	return randomNeighbor, present
}

func (g *Gossiper) RumorMongeringAddrStr(rumor *RumorMessage, peerStr string) {
	peerAddr, _ := net.ResolveUDPAddr("udp4", peerStr)
	g.RumorMongering(rumor, peerAddr)
}

func (g *Gossiper) RumorMongering(rumor *RumorMessage, peerAddr *net.UDPAddr) {
	go func() {
		// TODO: monitor the ack from the receiver
		observerChan := make(chan PeerStatus, CHANNEL_BUFFER_SIZE)
		timer := time.NewTimer(STATUS_MESSAGE_TIMEOUT)
		peerStr := peerAddr.String()

		// Register First
		g.dispatcher.registerListener <- RegisterMessage{
			observerChan: observerChan,
			tagger: StatusTagger{
				sender:     peerStr,
				identifier: rumor.Origin,
			},
			registerMsgType: Register,
		}

		unregister := func() {
			timer.Stop()
			g.dispatcher.registerListener <- RegisterMessage{
				observerChan:    observerChan,
				registerMsgType: Unregister,
			}
		}

		for {
			select {
			case peerStatus, ok := <-observerChan:
				// the channel has been closed by dispatcher
				if !ok {
					return
				}
				// this means that the peer has received the rumor (in this case, ps.nextID=rumor.ID+1)
				// or it already contains more advanced

				if peerStatus.NextID > rumor.ID {
					g.updatePeerStatusList(peerStr, peerStatus)
					// check whether the peer has been synced
					if g.syncWithPeer(peerStr) {
						// flip a coin to choose the next one
						g.flipCoinRumorMongering(rumor, GenerateStringSetSingleton(peerStr))
					}

					// channel exit
					unregister()
				}
			}
		}

	}()

	fmt.Println("MONGERING WITH PEER ", peerAddr)
	g.SendGossipPacket(&GossipPacket{Rumor: rumor}, peerAddr)
}

func (g *Gossiper) updatePeerStatusList(peerStr string, peerStatus PeerStatus) bool {
	// add the new peerStatus in the local peer
	g.peerWantListLock.RLock()
	defer g.peerWantListLock.RUnlock()

	_, present := g.peerWantList[peerStr]

	if !present {
		g.peerWantList[peerStr] = make(map[string]PeerStatus)
	}

	// check whether there is oldValue
	previousPeerStatus, present := g.peerWantList[peerStr][peerStatus.Identifier]

	if !present {
		g.peerWantList[peerStr][peerStatus.Identifier] = peerStatus
		return true
	} else {
		// TODO: cornor state: whether equality should be put into the consideration of update
		if previousPeerStatus.NextID > peerStatus.NextID {
			return false
		} else {
			g.peerWantList[peerStr][peerStatus.Identifier] = peerStatus
			return true
		}
	}
}

func (g *Gossiper) syncWithPeer(peerStr string) bool {
	// peerWant is a map[origin]PeerStatus
	peerWant := g.getOwnPeerSlice(peerStr)
	rumorToSend, rumorToAsk := g.ComputePeerStatusDiff(peerWant)
	return len(rumorToSend) == 0 && len(rumorToAsk) == 0
}

func (g *Gossiper) ComputePeerStatusDiff(peerWant []PeerStatus) (rumorToSend, rumorToAsk []PeerStatus) {
	rumorToSend = make([]PeerStatus, 0)
	rumorToSend = make([]PeerStatus, 0)
	// record the ww
	peerOrigins := make([]string, 0)

	// local peerstatus : map[origin]PeerStatus

	for _, pw := range peerWant {
		peerOrigins = append(peerOrigins, pw.Identifier)

		localStatus, present := g.peerStatuses[pw.Identifier]

		if !present {
			rumorToAsk = append(rumorToAsk, PeerStatus{Identifier: pw.Identifier, NextID: 0})
		} else if localStatus.NextID < pw.NextID {
			// it means we can ask the peer what we want
			rumorToAsk = append(rumorToAsk, localStatus)
		} else if localStatus.NextID > pw.NextID {
			rumorToSend = append(rumorToSend, pw)
		}
	}
	// our StringSet has provided some useful api
	peerOriginsSet := GenerateStringSet(peerOrigins)

	for localPeer, _ := range g.peerStatuses {
		if !peerOriginsSet.Has(localPeer) {
			rumorToSend = append(rumorToSend, PeerStatus{Identifier: localPeer, NextID: 0})
		}
	}
	return
}

func (g *Gossiper) getOwnPeerSlice(peerStr string) []PeerStatus {
	// First, we are sure that our peerlist must contain this peer, no need to check
	// so we could directly find their list
	// map[string](map[string]PeerStatus)

	// [origin]PeerStatus
	peerStatusMap, present := g.peerWantList[peerStr]
	peerSlice := make([]PeerStatus, 0)

	if present {
		for _, ps := range peerStatusMap {
			peerSlice = append(peerSlice, ps)
		}
	}

	return peerSlice
}

func (g *Gossiper) flipCoinRumorMongering(rumor *RumorMessage, excludedPeers *StringSet) {
	// 50 - 50
	if rand.Intn(2) == 0 {
		neighborPeer, present := g.RumorMongeringPrepare(rumor, excludedPeers)

		if present {
			fmt.Println("FLIP COIN RumorMongering to ", neighborPeer)
		}
	}
}

func (g *Gossiper) SelectRandomNeighbor(excludedPeer *StringSet) (string, bool) {
	peers := g.peersList.ToArray()
	notExcluded := make([]string, 0)

	for _, peer := range peers {
		if excludedPeer == nil && !excludedPeer.Has(peer) {
			notExcluded = append(notExcluded, peer)
		}
	}

	if len(notExcluded) == 0 {
		return "", false
	}

	return notExcluded[rand.Intn(len(notExcluded))], true
}

func (g *Gossiper) HandleStatusPacket(s *StatusPacket, sender *net.UDPAddr) {
	// rumorToSend, and rumorToAsk is []PeerStatus
	rumorToSend, rumorToAsk := g.ComputePeerStatusDiff(s.Want)

	if len(rumorToSend) > 0 {
		firstPeerStatus := rumorToSend[0]
		g.rumorListLock.Lock()
		firstRumor := g.rumorList[firstPeerStatus.Identifier][firstPeerStatus.NextID]

		// put the peerstatus to the channel
		g.dispatcher.statusListener <- PeerStatusWrapper{
			sender:       sender.String(),
			peerStatuses: s.Want,
		}
		g.RumorMongering(&firstRumor, sender)
	} else if len(rumorToAsk) > 0 {
		// TODO deal with the case to send rumorToAsk
		// g.sendgossippacket
	} else {
		// TODO: print out sync
	}
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
	// 2. update the peer status:
	origin := r.Origin
	messageID := r.ID

	g.rumorListLock.Lock()
	defer g.rumorListLock.Unlock()

	_, ok := g.rumorList[origin]

	if ok {
		g.rumorList[origin][messageID] = *r
		g.peerStatuses[r.Origin] = PeerStatus{
			Identifier: r.Origin,
			NextID:     messageID + 1,
		}
	} else {
		// this has been done during the *RumorStatusCheck* for the case not ok
		g.rumorList[origin] = make(map[uint32]RumorMessage)
		g.rumorList[origin][messageID] = *r
	}
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

func (g *Gossiper) SendGossipPacket(gp *GossipPacket, target *net.UDPAddr) {
	g.toSendChan <- &GossipPacketWrapper{sender: target, gossipPacket: gp}
}

func (g *Gossiper) SendGossipPacketStrAddr(gp *GossipPacket, targetStr string) {
	targetUDPAddr, err := net.ResolveUDPAddr("udp4", targetStr)

	if err != nil {
		fmt.Println(err)
	}

	g.SendGossipPacket(gp, targetUDPAddr)
}

func (g *Gossiper) SendClientAck(client *net.UDPAddr) {
	resp := &Message{Text: "Ok"}
	packetBytes, err := protobuf.Encode(resp)

	if err != nil {
		fmt.Println(err)
	}

	_, err = g.uiConn.WriteToUDP(packetBytes, client)
}

// func (g *Gossiper) HandleNodeMsg(gossipPacket *GossipPacket) {
// 	fmt.Println(gossipPacket)
// 	g.PrintPeers()
// 	newMsg := g.CreateForwardPacket(gossipPacket.Simple)
// 	newGossipPacket := &GossipPacket{Simple: newMsg}

// 	// add to the peers
// 	g.AddPeer(gossipPacket.Simple.RelayPeerAddr)

// 	g.BroadcastPacket(newGossipPacket, GenerateStringSetSingleton(gossipPacket.Simple.RelayPeerAddr))
// }

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

func (g *Gossiper) CreateStatusPacket() *Rumor

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

// func (g *Gossiper) ListenClientMsg() {
// 	msgBytes := make([]byte, UDP_DATAGRAM_MAX_SIZE)
// 	// init the data structure
// 	var msg Message

// 	for {
// 		g.uiConn.ReadFromUDP(msgBytes)
// 		protobuf.Decode(msgBytes, &msg)
// 		g.HandleClientMsg(&msg)
// 	}
// }

// func (g *Gossiper) ListenNodeMsg() {
// 	gossipBytes := make([]byte, UDP_DATAGRAM_MAX_SIZE)
// 	// init the data structure
// 	var gossipPacket GossipPacket

// 	for {
// 		g.conn.ReadFromUDP(gossipBytes)
// 		protobuf.Decode(gossipBytes, &gossipPacket)
// 		g.HandleNodeMsg(&gossipPacket)
// 	}
// }

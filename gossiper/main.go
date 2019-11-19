package gossiper

import (
	"encoding/hex"
	"fmt" // check the type of variable
	"net"
	"sync"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
	"github.com/dedis/protobuf"
)

// ./Peerster -gossipAddr=127.0.0.1:5001 -gui -GUIPort=8081 -peers=127.0.0.1:5002 -name=A -UIPort=8001
// ./Peerster -gossipAddr=127.0.0.1:5002 -gui -GUIPort=8082 -peers=127.0.0.1:5003 -name=A -UIPort=8002
// ./Peerster -gossipAddr=127.0.0.1:5003 -gui -GUIPort=8083 -peers=127.0.0.1:5001 -name=A -UIPort=8003

// QishanWang metafileHash
// 469403655c3a182a6b7856052a2428ebd24fede9e39b6cb428c21b8a0c222cc4
// Shaokang.png
// 2571718c9d1d4bbe9807df21f0dd84209d36b418ea15ca350c258495cdbe474d

const UDP_DATAGRAM_MAX_SIZE = 16384
const CHANNEL_BUFFER_SIZE = 1024
const STATUS_MESSAGE_TIMEOUT = 10 * time.Second
const GUI_ADDR = "127.0.0.1:8080"

const DEBUG = false
const DEBUGFILE = false
const DEBUGROUTE = false
const DEBUGSEARCH = true

// Memory arrangement
// Think about the process and what dataframe do we need
// 1. we need a list contain self peerstatus for sure, data format: map[string]PeerStatus peerName => PeerStatus
// 2. We need to record the peer status we have received from other peers, data format: map[string](map[string]PeerStatus) peerName => peerName => PeerStatus
// 3. Also, we need to record the rumour we are maintain data format: map[string](map[int]RumorMessage) peerName => sequential number => rumorMessage

func NewGossiper(gossipAddr string, uiPort string, name string, peersStr *StringSet, rtimer int, simple bool, antiEntropy int, gui bool, guiPort string) *Gossiper {
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

	// guiAddrStrSlice := strings.Split(gossipAddr, "")
	// guiAddrStrSlice[10] = "8"
	// guiAddrStr := strings.Join(guiAddrStrSlice, "")

	guiAddr := fmt.Sprintf("127.0.0.1:%s", guiPort)
	fmt.Printf("GUI Port is %s \n", guiAddr)

	routeTable := make(map[string]string)
	idTable := make(map[string]uint32)

	routeTableObject := RouteTable{
		routeTable: routeTable,
		IDTable:    idTable,
		Mux:        &sync.Mutex{},
	}

	privateMessageList := PrivateMessageList{
		privateMessageList: make(map[string][]PrivateMessage),
		Mux:                &sync.Mutex{},
	}

	currentID := CurrentID{
		currentID: 1,
		Mux:       &sync.Mutex{},
	}

	return &Gossiper{
		address: udpAddr,
		conn:    udpConn,
		uiAddr:  udpUIAddr,
		uiConn:  udpUIConn,
		name:    name,
		peersList: &PeersList{
			PeersList: peersStr,
			Mux:       &sync.Mutex{},
		},
		simple:             simple,
		peerStatuses:       make(map[string]PeerStatus),
		peerStatusesLock:   &sync.Mutex{},
		peerWantList:       make(map[string](map[string]PeerStatus)),
		peerWantListLock:   &sync.RWMutex{},
		rumorList:          make(map[string](map[uint32]RumorMessage)),
		rumorListLock:      &sync.RWMutex{},
		simpleList:         make(map[string](map[string]bool)),
		simpleListLock:     &sync.RWMutex{},
		currentID:          &currentID,
		dispatcher:         nil,
		toSendChan:         make(chan *GossipPacketWrapper, CHANNEL_BUFFER_SIZE),
		antiEntropy:        antiEntropy,
		guiAddr:            guiAddr,
		gui:                gui,
		routeTable:         &routeTableObject,
		rtimer:             time.Duration(rtimer) * time.Second,
		privateMessageList: &privateMessageList,
	}
}

func (g *Gossiper) Run() {
	peerListener := g.ReceiveFromPeers()
	clientListener := g.ReceiveFromClients()

	if g.antiEntropy > 0 && !g.simple {
		go g.AntiEntropy()
	}

	// Here to run the server
	if g.gui {
		go g.ListenToGUI()
	}

	// send route message
	if g.rtimer > 0 {
		go g.RunRoutingMessage()
	}

	g.fileHandler = NewFileHandler(g.name)
	go g.RunFileSystem()

	g.dispatcher = StartPeerStatusDispatcher()
	g.Listen(peerListener, clientListener)
}

func (g *Gossiper) Listen(peerListener <-chan *GossipPacketWrapper, clientListener <-chan *ClientMessageWrapper) {
	for {
		select {
		case gpw := <-peerListener:
			go g.HandlePeerMessage(gpw)
		case cmw := <-clientListener:
			// fmt.Println("Handle Client Msg")
			go g.HandleClientMessage(cmw)
		case gpw := <-g.toSendChan:
			gp := gpw.gossipPacket

			peerAddr := gpw.sender
			packetBytes, err := protobuf.Encode(gp)

			if err != nil {
				panic(err)
			}

			_, err = g.conn.WriteToUDP(packetBytes, peerAddr)

			if err != nil {
				panic(err)
			}

		}
	}
}

func (g *Gossiper) AntiEntropy() {
	if DEBUG {
		fmt.Println("Start antiEntropy")
	}

	// time.Duration can convert int to time type

	go func() {
		if DEBUG {
			fmt.Println("The value of antientropy is ", g.antiEntropy)
		}

		ticker := time.NewTicker(time.Duration(g.antiEntropy) * time.Second)

		for t := range ticker.C {

			_ = t

			neighbor, present := g.SelectRandomNeighbor(nil)
			if present {
				if DEBUG {
					fmt.Println("Anti entropy " + neighbor)
				}
				g.SendGossipPacketStrAddr(g.CreateStatusPacket(), neighbor)
			}
		}

	}()
}

func (g *Gossiper) HandlePeerMessage(gpw *GossipPacketWrapper) {
	packet := gpw.gossipPacket
	sender := gpw.sender

	g.AddPeer(sender.String())

	// OUTPUT-HW1
	if DEBUG {
		g.PrintPeers()
	}

	switch {
	case packet.Simple != nil:
		// OUTPUT-HW1
		fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s \n",
			packet.Simple.OriginalName,
			packet.Simple.RelayPeerAddr,
			packet.Simple.Contents)

		g.HandleSimplePacket(packet.Simple)
	case packet.Rumor != nil:
		// OUTPUT-HW1
		fmt.Printf("RUMOR origin %s from %s ID %d contents %s \n",
			packet.Rumor.Origin,
			sender,
			packet.Rumor.ID,
			packet.Rumor.Text)
		if packet.Rumor.ID != 0 {
			g.HandleRumorPacket(packet.Rumor, sender)
		} else {
			// fmt.Println("EMPTY Packet")
			g.SendGossipPacket(g.CreateStatusPacket(), sender)
		}

	case packet.Status != nil:
		// OUTPUT-HW1
		if DEBUG {
			fmt.Println(packet.Status.SenderString(sender.String()))
		}
		g.HandleStatusPacket(packet.Status, sender)

	case packet.Private != nil:
		g.HandlePrivatePacket(packet.Private, sender)
	case packet.DataReply != nil:
		g.HandleDataReply(packet.DataReply, sender)

	case packet.DataRequest != nil:
		g.HandleDataRequest(packet.DataRequest, sender)
	case packet.SearchReply != nil:
		g.HandleSearchReply(packet.SearchReply, sender)
	case packet.SearchRequest != nil:
		g.HandleSearchRequest(packet.SearchRequest, sender)
	}
}

func (g *Gossiper) HandleClientMessage(cmw *ClientMessageWrapper) {
	msg := cmw.msg

	if msg.Text != "" && msg.Destination == nil {
		// Handle with rumor message
		if g.simple {
			newMsg := g.CreateClientPacket(msg)
			newGossipPacket := &GossipPacket{Simple: newMsg}
			g.BroadcastPacket(newGossipPacket, nil)
		} else {
			newRumorMsg := g.CreateRumorPacket(msg)

			// fmt.Println("CURRENTID is ", g.currentID)
			// the second arguement is the last-step source of the message
			// here include the case that we receive it from the client
			g.HandleRumorPacket(newRumorMsg, g.address)
		}
	} else if msg.Text != "" && msg.Destination != nil {
		// Handle with private message
		fmt.Printf("CLIENT MESSAGE %s dest %s \n", cmw.msg.Text, *cmw.msg.Destination)
		g.HandleClientPrivate(cmw)
	} else if msg.File != nil && msg.Destination != nil && msg.Request != nil {
		// Handle with file download
		fmt.Printf("Ask %s from dest %s meta %s \n", *msg.File, *msg.Destination, hex.EncodeToString(*msg.Request))
		g.HandleDownloadRequest(cmw)
	} else if msg.File != nil && msg.Request == nil {
		fmt.Printf("Index File %s \n", *msg.File)
		g.HandleFileIndexing(cmw)
	} else if msg.File != nil && msg.Request != nil {
		fmt.Printf("Download Searched file %s with meta %s \n", *msg.File, hex.EncodeToString(*msg.Request))
	} else if *msg.Keywords != "" {
		fmt.Printf("Start Query for the keywords %s with budget %d \n", *msg.Keywords, *msg.Budget)
		g.HandleClientSearch(cmw)
	}
}

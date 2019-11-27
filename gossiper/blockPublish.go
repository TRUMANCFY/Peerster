package gossiper

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
)

type BlockPublishHandler struct {
	name                   string
	blockPublishList       *BlockPublishList
	blockPublishDispatcher *BlockPublishDispatcher
	numNodes               int
}

type BlockPublishList struct {
	blockPublishList map[string](map[uint32]*TLCMessage)
	fileNames        []string
	Mux              *sync.Mutex
}

type BlockPublishWatcher struct {
	ackChan      chan<- *TLCAck
	receivedAcks []string
	id           uint32
	Mux          *sync.Mutex
}

func NewBlockPublishHandler(name string, numNodes int) *BlockPublishHandler {
	blockPublishList := make(map[string](map[uint32]*TLCMessage))
	fileNames := make([]string, 0)

	blockPublishObject := &BlockPublishList{
		blockPublishList: blockPublishList,
		fileNames:        fileNames,
		Mux:              &sync.Mutex{},
	}

	return &BlockPublishHandler{
		name:                   name,
		numNodes:               numNodes,
		blockPublishList:       blockPublishObject,
		blockPublishDispatcher: NewBlockPublishDispatcher(),
	}
}

func (g *Gossiper) AnnounceFile(f *File) {
	txPublish := TxPublish{
		Name:         f.Name,
		Size:         f.Size,
		MetafileHash: f.MetafileHash[:],
	}

	blockPublish := BlockPublish{
		PrevHash:    [32]byte{},
		Transaction: txPublish,
	}

	g.currentID.Mux.Lock()

	tlcMessage := &TLCMessage{
		Origin:      g.name,
		ID:          g.currentID.currentID,
		Confirmed:   false,
		TxBlock:     blockPublish,
		VectorClock: nil,
		Fitness:     0,
	}
	g.currentID.currentID++
	g.currentID.Mux.Unlock()

	g.SendTLCMessage(tlcMessage)
}

func (g *Gossiper) SendTLCMessage(tlcMessage *TLCMessage) {
	g.HandleRumorPacket(&GossipPacket{TLCMessage: tlcMessage}, g.address)
	observer := make(chan *TLCAck, CHANNEL_BUFFER_SIZE)
	receivedNodes := make([]string, 0)
	receivedNodes = append(receivedNodes, g.name)

	blockPublishWatcher := &BlockPublishWatcher{
		ackChan:      observer,
		receivedAcks: receivedNodes,
		id:           tlcMessage.ID,
		Mux:          &sync.Mutex{},
	}

	g.blockPublishHandler.RegisterBlockPublish(blockPublishWatcher)

	timer := time.NewTicker(time.Duration(g.stubbornTimeout) * time.Second)
	go func() {
		defer g.blockPublishHandler.UnregisterBlockPublish(blockPublishWatcher)
		defer timer.Stop()
		for {
			select {
			case ackReply := <-observer:
				if DEBUGTLC {
					fmt.Printf("ACK Received from %s with ID %d \n", ackReply.Origin, ackReply.ID)
				}

				newOrigin := ackReply.Origin
				// check whether it already exist in the list
				existence := false
				blockPublishWatcher.Mux.Lock()
				for _, preNode := range receivedNodes {
					if preNode == newOrigin {
						existence = true
					}
				}
				blockPublishWatcher.Mux.Unlock()

				if existence {
					continue
				}

				// put it to the nodes
				blockPublishWatcher.Mux.Lock()
				blockPublishWatcher.receivedAcks = append(blockPublishWatcher.receivedAcks, newOrigin)
				nodeLen := len(blockPublishWatcher.receivedAcks)
				blockPublishWatcher.Mux.Unlock()

				// check the length
				if nodeLen > g.numNodes/2 {
					if DEBUGTLC {
						fmt.Println("The block publish requirement has been reached!")
					}

					tlcMessage.Confirmed = true
					if HW3OUTPUT {
						fmt.Printf("RE-BROADCAST ID %d WITNESSES %s,etc", tlcMessage.ID, strings.Join(blockPublishWatcher.receivedAcks, ","))
					}

					// Resend the confirmed TLCMessage
					g.HandleRumorPacket(&GossipPacket{TLCMessage: tlcMessage}, g.address)
					return
				}
			case <-timer.C:
				g.HandleRumorPacket(&GossipPacket{TLCMessage: tlcMessage}, g.address)
			}
		}

	}()
}

func (g *Gossiper) HandleTLCMessage(gp *GossipPacket, senderAddr *net.UDPAddr) {
	fmt.Printf("Handle TLC Message from %s \n", gp.TLCMessage.Origin)
	if gp.TLCMessage.Origin == g.name {
		return
	}

	tlcMessage := gp.TLCMessage
	// check whether the file name already exist
	valid := g.blockPublishHandler.ContainFile(tlcMessage)

	if !valid {
		return
	}

	ack := g.GenerateAck(gp)

	if HW3OUTPUT {
		fmt.Printf("SENDING ACK origin %s ID %d \n", ack.Origin, ack.ID)
	}

	g.SendTLCAck(&GossipPacket{Ack: ack})
}

func (g *Gossiper) HandleTLCAck(gp *GossipPacket, sender *net.UDPAddr) {
	dest := gp.Ack.Destination

	if DEBUGTLC {
		fmt.Printf("Receive ACK From %s To %s with ID %d with HopLimit %d \n", gp.Ack.Destination, gp.Ack.Origin, gp.Ack.ID, gp.Ack.HopLimit)
	}

	if dest == g.name {
		g.blockPublishHandler.blockPublishDispatcher.tlcAckChan <- gp.Ack
		return
	}

	hopLimit := gp.Ack.HopLimit

	if hopLimit == 0 {
		fmt.Println("Hop Limit has been ended!")
		return
	}

	newAck := g.GenerateNewAck(gp)
	// forward to TLC
	g.SendTLCAck(&GossipPacket{Ack: newAck})
}

func (g *Gossiper) GenerateNewAck(gp *GossipPacket) *TLCAck {
	return &TLCAck{
		Origin:      gp.Ack.Origin,
		Destination: gp.Ack.Destination,
		ID:          gp.Ack.ID,
		Text:        gp.Ack.Text,
		HopLimit:    gp.Ack.HopLimit - 1,
	}
}

func (g *Gossiper) GenerateAck(gp *GossipPacket) *TLCAck {
	return &TLCAck{
		Origin:      g.name,
		ID:          gp.TLCMessage.ID,
		Text:        "",
		Destination: gp.TLCMessage.Origin,
		HopLimit:    HOPLIMIT,
	}
}

func (g *Gossiper) SendTLCAck(gp *GossipPacket) {
	// check the route table first
	dest := gp.Ack.Destination

	g.routeTable.Mux.Lock()

	nextNode, present := g.routeTable.routeTable[dest]

	g.routeTable.Mux.Unlock()

	if !present {
		fmt.Printf("Destination %s does not exist in the table \n", dest)
		return
	}

	if DEBUGTLC {
		fmt.Printf("Send TLCAck from %s to %s with %d \n", gp.Ack.Origin, gp.Ack.Destination, gp.Ack.ID)
	}

	g.SendGossipPacketStrAddr(gp, nextNode)
}

func (bp *BlockPublishHandler) ContainFile(tlcMessage *TLCMessage) bool {
	bp.blockPublishList.Mux.Lock()
	defer bp.blockPublishList.Mux.Unlock()

	if tlcMessage.Confirmed {
		// already confirmed
		if HW3OUTPUT {
			fmt.Printf("CONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %x \n",
				tlcMessage.Origin,
				tlcMessage.ID,
				tlcMessage.TxBlock.Transaction.Name,
				tlcMessage.TxBlock.Transaction.Size,
				hex.EncodeToString(tlcMessage.TxBlock.Transaction.MetafileHash))
		}

		return false
	}

	if HW3OUTPUT {
		fmt.Printf("UNCONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %x \n",
			tlcMessage.Origin,
			tlcMessage.ID,
			tlcMessage.TxBlock.Transaction.Name,
			tlcMessage.TxBlock.Transaction.Size,
			hex.EncodeToString(tlcMessage.TxBlock.Transaction.MetafileHash))
	}

	inputFileName := tlcMessage.TxBlock.Transaction.Name

	for _, fname := range bp.blockPublishList.fileNames {
		if fname == inputFileName {
			return false
		}
	}

	// put the new filename in
	bp.blockPublishList.fileNames = append(bp.blockPublishList.fileNames, inputFileName)

	// put the tlcMessage
	origin := tlcMessage.Origin
	id := tlcMessage.ID

	_, present := bp.blockPublishList.blockPublishList[origin]

	if !present {
		bp.blockPublishList.blockPublishList[origin] = make(map[uint32]*TLCMessage)
	}

	bp.blockPublishList.blockPublishList[origin][id] = tlcMessage

	return true
}

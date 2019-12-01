package gossiper

import (
	"fmt"
	"strings"
	"sync"

	. "github.com/TRUMANCFY/Peerster/message"
)

type RoundHandler struct {
	name            string
	my_time         *Round           // round information
	messageChan     chan *TLCMessage // both confirmed and unconfirmed tlcMessage
	roundCounter    *RoundCounter
	roundMessageMap *RoundMessageMap
	firstRound      bool
}

type Round struct {
	round uint32
	Mux   *sync.Mutex
}

type RoundMessageMap struct {
	roundMessageMap map[uint32]([]*TLCMessage) // round to confirmed message
	Mux             *sync.Mutex
}

type RoundCounter struct {
	roundCounter map[string]([]*TLCMessage) // origin to message (the length is the round)
	Mux          *sync.Mutex
}

func NewRoundHandler(name string) *RoundHandler {
	my_time := &Round{
		round: 0,
		Mux:   &sync.Mutex{},
	}
	// this channel should be blocking
	messageChan := make(chan *TLCMessage, CHANNEL_BUFFER_SIZE)

	roundCounter := &RoundCounter{
		roundCounter: make(map[string][]*TLCMessage),
		Mux:          &sync.Mutex{},
	}

	localRoundMessageMap := &RoundMessageMap{
		roundMessageMap: make(map[uint32][]*TLCMessage),
		Mux:             &sync.Mutex{},
	}

	return &RoundHandler{
		name:            name,
		my_time:         my_time,
		messageChan:     messageChan,
		roundCounter:    roundCounter,
		roundMessageMap: localRoundMessageMap,
		firstRound:      true,
	}
}

func (g *Gossiper) RunRound() {
	for {
		// TODO: the channel should be blocked???
		msg := <-g.roundHandler.messageChan
		if msg.Confirmed == 0 {
			// this is an unconfirmed message, when comes it would increase
			// the reason why we do not need check duplicate here
			// because the input of the channel will put inside the `AcceptRumor` function
			// which is not duplicate

			if DEBUGROUND {
				fmt.Printf("Round Unconfirmed TLCMessage Origin: %s ID: %d, Name: %s \n", msg.Origin, msg.ID, msg.TxBlock.Transaction.Name)
			}

			g.roundHandler.roundCounter.Mux.Lock()
			_, present := g.roundHandler.roundCounter.roundCounter[msg.Origin]

			if !present {
				g.roundHandler.roundCounter.roundCounter[msg.Origin] = make([]*TLCMessage, 0)
			}

			g.roundHandler.roundCounter.roundCounter[msg.Origin] = append(g.roundHandler.roundCounter.roundCounter[msg.Origin], msg)
			g.roundHandler.roundCounter.Mux.Unlock()

		} else {
			// this is a confirmed message which has already been accepted
			// get the current round first

			// get the message round
			// this one must exist because the uncomfirmed has already come
			// when there is just one message uncomfirmed, the round should be 0
			if DEBUGROUND {
				fmt.Printf("Round Confirmed TLCMessage Origin: %s ID: %d, Name: %s \n", msg.Origin, msg.ID, msg.TxBlock.Transaction.Name)
			}
			g.roundHandler.roundCounter.Mux.Lock()
			roundNum := uint32(len(g.roundHandler.roundCounter.roundCounter[msg.Origin])) - 1
			g.roundHandler.roundCounter.Mux.Unlock()

			g.roundHandler.roundMessageMap.Mux.Lock()
			_, present := g.roundHandler.roundMessageMap.roundMessageMap[roundNum]
			if !present {
				g.roundHandler.roundMessageMap.roundMessageMap[roundNum] = make([]*TLCMessage, 0)
			}

			g.roundHandler.roundMessageMap.roundMessageMap[roundNum] = append(g.roundHandler.roundMessageMap.roundMessageMap[roundNum], msg)

			if DEBUGROUND {
				fmt.Printf("Current Round: %d Confirmed Number %d \n", roundNum, len(g.roundHandler.roundMessageMap.roundMessageMap[roundNum]))
			}

			g.roundHandler.roundMessageMap.Mux.Unlock()
		}

		g.CheckRound()
	}
}

func (g *Gossiper) CheckRound() {
	// self round checking
	g.roundHandler.roundCounter.Mux.Lock()
	numSelfMsg := int(len(g.roundHandler.roundCounter.roundCounter[g.name]))
	g.roundHandler.roundCounter.Mux.Unlock()

	// check num of confirmed msg in this round
	// get the round now
	g.roundHandler.my_time.Mux.Lock()
	currentRound := g.roundHandler.my_time.round
	g.roundHandler.my_time.Mux.Unlock()

	// check num of confirmed msg in current round
	g.roundHandler.roundMessageMap.Mux.Lock()
	numCurrentRound := len(g.roundHandler.roundMessageMap.roundMessageMap[currentRound])
	g.roundHandler.roundMessageMap.Mux.Unlock()

	if DEBUGROUND {
		fmt.Printf("Num of Self Unconfirmed: %d; CurrentRound: %d; Number of Confirmed: %d \n", numSelfMsg, currentRound, numCurrentRound)
	}
	// not use uint32,, negative is dangerous
	if numSelfMsg-1 > int(currentRound) && numCurrentRound > g.numNodes/2 {
		if DEBUGROUND {
			fmt.Println("Round condition has been reached")
		}
		// update my_time
		g.roundHandler.my_time.Mux.Lock()
		// get the message to send out
		g.roundHandler.my_time.round++

		if HW3OUTPUT {
			fmt.Printf("ADVANCING TO round ​%d BASED ON CONFIRMED MESSAGES %s \n", g.roundHandler.my_time.round, g.roundHandler.generateOriginText(currentRound))
		}

		fmt.Println(len(g.roundHandler.roundCounter.roundCounter[g.name]))
		fmt.Println(int(g.roundHandler.my_time.round))
		msgToSend := g.roundHandler.roundCounter.roundCounter[g.name][int(g.roundHandler.my_time.round)]
		g.roundHandler.my_time.Mux.Unlock()

		// send out the msg
		g.SendTLCMessage(msgToSend)
		// stop the ack listening for the last unconfirmed msg
		g.roundHandler.roundCounter.Mux.Lock()
		currentMsg := g.roundHandler.roundCounter.roundCounter[g.name][currentRound]
		g.roundHandler.roundCounter.Mux.Unlock()
		g.blockPublishHandler.UnregisterBlockPublish(&BlockPublishWatcher{id: currentMsg.ID})
	}
}

func (g *Gossiper) GetRound(gp *GossipPacket) int {
	if gp.TLCMessage.Confirmed != 0 {
		return -1
	}

	round := 0
	g.roundHandler.roundCounter.Mux.Lock()
	msgList, present := g.roundHandler.roundCounter.roundCounter[gp.TLCMessage.Origin]
	g.roundHandler.roundCounter.Mux.Unlock()

	if !present {
		return 0
	}

	for _, msg := range msgList {
		if msg.ID < gp.TLCMessage.ID {
			round++
		}
	}

	return round
}

func (r *RoundHandler) generateOriginText(prevRound uint32) string {
	r.roundMessageMap.Mux.Lock()
	msgs, present := r.roundMessageMap.roundMessageMap[prevRound]
	r.roundMessageMap.Mux.Unlock()

	if !present {
		fmt.Println("The previous round not exist")
		return ""
	}

	subStrs := make([]string, 0)
	for ind, msg := range msgs {
		subStrs = append(subStrs, fmt.Sprintf("origin%d %s ID%d %d", ind, msg.Origin, ind, msg.ID))
	}
	// TODO: whether there is a space after the comma
	return strings.Join(subStrs, ", ")
}

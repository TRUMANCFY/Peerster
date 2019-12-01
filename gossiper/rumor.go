package gossiper

import (
	// check the type of variable

	"fmt"
	"math/rand"
	"net"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
)

func (g *Gossiper) HandleRumorPacket(gp *GossipPacket, senderAddr *net.UDPAddr) {
	// diff == 0 => accept the rumor as rumor is rightly updated
	// diff > 0 => drop it (should we keep this for the future reference???)
	// diff < 0 => drop it
	// if sender is self, broadcast (mongering) the rumor

	// if it is a TCLMessage
	g.updateRouteTable(gp, senderAddr.String())

	if gp.TLCMessage != nil {
		g.HandleTLCMessage(gp, senderAddr)
	}

	diff := g.RumorStatusCheck(gp)

	var origin string
	var id uint32

	if gp.Rumor != nil {
		origin = gp.Rumor.Origin
		id = gp.Rumor.ID
	} else if gp.TLCMessage != nil {
		origin = gp.TLCMessage.Origin
		id = gp.TLCMessage.ID
	} else {
		fmt.Println("The GossipPacket is illegal!!!")
	}

	if DEBUGROUTE {
		fmt.Printf("Origin: %s, ID: %d, From: %s \n", origin, id, senderAddr.String())
	}

	// CHECKOUT
	if DEBUG {
		fmt.Println("DIFF is", diff)
	}

	// fmt.Printf("The difference between the comming rumor and current peerstatus is %d \n", diff)

	switch {
	case diff == 0:
		// accept the rumor
		// as soon as received, mongering should start
		// update the table, maybe have been done in acceptrumor function

		// CHECK
		if gp == nil {
			fmt.Println("CP1")
		}
		if g.address == senderAddr {
			// CHECK
			// The message is from local client

			go g.RumorMongeringPrepare(gp, nil)
		} else {
			go g.RumorMongeringPrepare(gp, GenerateStringSetSingleton(senderAddr.String()))
		}

		g.AcceptRumor(gp)

	case diff > 0:
		// TODO: consider the out-of-order problem

		// TODO: Still send the status packet to ask for the rumor
		// g.SendGossipPacket(g.CreateStatusPacket(), senderAddr)
		if DEBUG {
			fmt.Println("The new coming rumor ID is smaller than our local")
		}

		// send the rumor the sender want
	case diff < 0:
		if DEBUG {
			fmt.Println("The new coming rumor ID is larger than our local")
		}

		// we need to keep some advanced in our buffer
		if g.hw3ex3 {
			if gp.TLCMessage != nil {
				fmt.Printf("We have buffer TLC message from %s with ID %d \n", gp.TLCMessage.Origin, gp.TLCMessage.ID)
				g.bufferMsg.Mux.Lock()
				_, present := g.bufferMsg.msgBuffer[gp.TLCMessage.Origin]
				if !present {
					g.bufferMsg.msgBuffer[gp.TLCMessage.Origin] = make(map[uint32]*GossipPacket)
				}

				g.bufferMsg.msgBuffer[gp.TLCMessage.Origin][gp.TLCMessage.ID] = gp
				g.bufferMsg.Mux.Unlock()
			}
		}

	}

	// Send the StatusMessageToSender if the rumor is not from self
	if senderAddr != g.address {
		// send the status message to sender
		if DEBUG {
			fmt.Printf("Send Status to %s \n", senderAddr.String())
		}
		gpToSend := g.CreateStatusPacket()
		g.SendGossipPacket(gpToSend, senderAddr)
	}
}

func (g *Gossiper) RumorMongeringPrepare(gp *GossipPacket, excludedPeers *StringSet) (string, bool) {
	randomNeighbor, present := g.SelectRandomNeighbor(excludedPeers)

	if present {
		if gp == nil {
			fmt.Println("CP2")
		}
		go g.RumorMongeringAddrStr(gp, randomNeighbor)
	}

	return randomNeighbor, present
}

func (g *Gossiper) RumorMongeringAddrStr(gp *GossipPacket, peerStr string) {
	peerAddr, _ := net.ResolveUDPAddr("udp4", peerStr)
	g.RumorMongering(gp, peerAddr)
}

func (g *Gossiper) RumorMongering(gp *GossipPacket, peerAddr *net.UDPAddr) {
	// HW1-OUTPUT
	if HW1OUTPUT {
		fmt.Printf("MONGERING with %s \n", peerAddr.String())
	}

	go func() {
		// monitor the ack from the receiver
		observerChan := make(chan PeerStatus, CHANNEL_BUFFER_SIZE)
		timer := time.NewTimer(STATUS_MESSAGE_TIMEOUT)
		peerStr := peerAddr.String()

		// Register First
		// fmt.Printf("Register Identifier: %s Sender: %s \n", rumor.Origin, peerStr)

		var origin string
		var id uint32

		if gp.Rumor != nil {
			origin = gp.Rumor.Origin
			id = gp.Rumor.ID
		} else if gp.TLCMessage != nil {
			origin = gp.TLCMessage.Origin
			id = gp.TLCMessage.ID
		} else {
			fmt.Println("The GossipPacket is illegal!!!!")
		}

		g.dispatcher.taggerListener <- TaggerMessage{
			observerChan: observerChan,
			tagger: StatusTagger{
				sender:     peerStr,
				identifier: origin,
			},
			taggerMsgType: TakeIn,
		}

		unregister := func() {
			timer.Stop()
			g.dispatcher.taggerListener <- TaggerMessage{
				observerChan:  observerChan,
				taggerMsgType: TakeOut,
			}
		}

		for {
			select {
			case peerStatus, ok := <-observerChan:
				// the channel has been closed by dispatcher
				if !ok {
					return
				}

				if DEBUG {
					fmt.Printf("Receive Peer Status Origin: %s, NextID: %d \n", peerStatus.Identifier, peerStatus.NextID)
				}

				// this means that the peer has received the rumor (in this case, ps.nextID=rumor.ID+1)
				// or it already contains more advanced

				if peerStatus.NextID >= id {
					g.updatePeerStatusList(peerStr, peerStatus)
					// check whether the peer has been synced
					if g.isRumorSync(gp, peerStr) {
						// flip a coin to choose the next one
						g.flipCoinRumorMongering(gp, GenerateStringSetSingleton(peerStr))
					}

					// channel exit
					unregister()
				}
			case <-timer.C: // Timed out
				// Resend the rumor to another neighbor with prob 1/2

				if DEBUG {
					fmt.Println("TIMEOUT")
				}

				g.flipCoinRumorMongering(gp, GenerateStringSetSingleton(peerStr))
				unregister()
			}
		}

	}()

	// fmt.Println("MONGERING WITH PEER ", peerAddr)
	g.SendGossipPacket(gp, peerAddr)
}

func (g *Gossiper) flipCoinRumorMongering(gp *GossipPacket, excludedPeers *StringSet) {
	// 50 - 50
	if DEBUG {
		fmt.Println("Prepare to flip the coin")
	}

	// rand.Seed(time.Now().UTC().UnixNano())
	randInt := rand.Intn(2)
	if randInt == 0 {
		if gp == nil {
			fmt.Println("CP3")
		}
		neighborPeer, present := g.RumorMongeringPrepare(gp, excludedPeers)

		if present {
			// HW1-OUTPUT
			if HW1OUTPUT {
				fmt.Printf("FLIPPED COIN sending rumor to %s \n", neighborPeer)
			}
		} else {
			if DEBUG {
				fmt.Println("FLIPPED COIN not exist")
			}
		}
	} else {
		if DEBUG {
			fmt.Println("Choose not to flip the coin")
		}
	}
}

func (g *Gossiper) AcceptRumor(gp *GossipPacket) {
	// 1. put the rumor in the list
	// 2. update the peer status

	var origin string
	var messageID uint32

	if gp.Rumor != nil {
		origin = gp.Rumor.Origin
		messageID = gp.Rumor.ID
	} else if gp.TLCMessage != nil {
		origin = gp.TLCMessage.Origin
		messageID = gp.TLCMessage.ID
	} else {
		fmt.Println("The GossipPacket is illegal!!!")
	}

	if DEBUGTLC || DEBUGROUND {
		fmt.Printf("Accept Rumor Origin: %s ID: %d \n", origin, messageID)
	}

	if g.hw3ex3 {
		if gp.TLCMessage != nil {
			if gp.TLCMessage.Origin != g.name {
				g.roundHandler.messageChan <- gp.TLCMessage
			}
		}
	}

	g.rumorListLock.Lock()
	_, ok := g.rumorList[origin]
	g.rumorListLock.Unlock()

	if ok {
		g.rumorListLock.Lock()
		g.rumorList[origin][messageID] = gp
		g.rumorListLock.Unlock()

		g.peerStatusesLock.Lock()
		g.peerStatuses[origin] = PeerStatus{
			Identifier: origin,
			NextID:     messageID + 1,
		}
		g.peerStatusesLock.Unlock()

	} else {
		// this has been done during the *RumorStatusCheck* for the case not ok
		g.rumorList[origin] = make(map[uint32]*GossipPacket)
		g.rumorListLock.Lock()
		g.rumorList[origin][messageID] = gp
		g.rumorListLock.Unlock()

		g.peerStatusesLock.Lock()
		g.peerStatuses[origin] = PeerStatus{
			Identifier: origin,
			NextID:     messageID + 1,
		}
		g.peerStatusesLock.Unlock()
	}

	// check the buffer
	if g.hw3ex3 {
		if gp.TLCMessage != nil {
			g.bufferMsg.Mux.Lock()
			idGpMap, presentOrigin := g.bufferMsg.msgBuffer[origin]
			g.bufferMsg.Mux.Unlock()
			if presentOrigin {
				newGp, presentID := idGpMap[messageID+1]
				if presentID {
					go g.AcceptRumor(newGp)
				}
			}
		}
	}
}

func (g *Gossiper) isRumorSync(gp *GossipPacket, peerStr string) bool {
	g.peerWantListLock.Lock()
	defer g.peerWantListLock.Unlock()

	var origin string
	var id uint32

	if gp.Rumor != nil {
		origin = gp.Rumor.Origin
		id = gp.Rumor.ID
	} else if gp.TLCMessage != nil {
		origin = gp.TLCMessage.Origin
		id = gp.TLCMessage.ID
	} else {
		fmt.Println("The GossipPacket is illegal!!!!")
	}

	originList, present := g.peerWantList[peerStr]
	if !present {
		return false
	}

	peerStatus, present := originList[origin]

	if !present {
		return false
	}

	return peerStatus.NextID > id
}

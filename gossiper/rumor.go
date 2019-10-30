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

func (g *Gossiper) HandleRumorPacket(r *RumorMessage, senderAddr *net.UDPAddr) {
	// diff == 0 => accept the rumor as rumor is rightly updated
	// diff > 0 => drop it (should we keep this for the future reference???)
	// diff < 0 => drop it
	// if sender is self, broadcast (mongering) the rumor
	diff := g.RumorStatusCheck(r)

	// CHECKOUT
	fmt.Println("DIFF is", diff)

	// fmt.Printf("The difference between the comming rumor and current peerstatus is %d \n", diff)

	switch {
	case diff == 0:
		// accept the rumor
		// as soon as received, mongering should start
		// update the table, maybe have been done in acceptrumor function

		// CHECK
		if g.address == senderAddr {
			// CHECK
			// The message is from local client
			go g.RumorMongeringPrepare(r, nil)
		} else {
			go g.RumorMongeringPrepare(r, GenerateStringSetSingleton(senderAddr.String()))
		}

		g.AcceptRumor(r)

		g.updateRouteTable(r, senderAddr.String())

	case diff > 0:
		// TODO: consider the out-of-order problem

		// TODO: Still send the status packet to ask for the rumor
		// g.SendGossipPacket(g.CreateStatusPacket(), senderAddr)
		fmt.Println("The new coming rumor ID is larger than our local")
		// send the rumor the sender want

		// TODO-2: update the rumorTable
		g.updateRouteTable(r, senderAddr.String())

	case diff < 0:
		fmt.Println("The new coming rumor ID is smaller than our local")
	}

	// Send the StatusMessageToSender if the rumor is not from self
	if senderAddr != g.address {
		// send the status message to sender
		fmt.Printf("Send Status to %s \n", senderAddr.String())
		gpToSend := g.CreateStatusPacket()
		g.SendGossipPacket(gpToSend, senderAddr)
	}
}

func (g *Gossiper) RumorMongeringPrepare(rumor *RumorMessage, excludedPeers *StringSet) (string, bool) {
	randomNeighbor, present := g.SelectRandomNeighbor(excludedPeers)

	if present {
		go g.RumorMongeringAddrStr(rumor, randomNeighbor)
	}

	return randomNeighbor, present
}

func (g *Gossiper) RumorMongeringAddrStr(rumor *RumorMessage, peerStr string) {
	peerAddr, _ := net.ResolveUDPAddr("udp4", peerStr)
	g.RumorMongering(rumor, peerAddr)
}

func (g *Gossiper) RumorMongering(rumor *RumorMessage, peerAddr *net.UDPAddr) {
	// OUTPUT
	fmt.Printf("MONGERING with %s \n", peerAddr.String())
	go func() {
		// monitor the ack from the receiver
		observerChan := make(chan PeerStatus, CHANNEL_BUFFER_SIZE)
		timer := time.NewTimer(STATUS_MESSAGE_TIMEOUT)
		peerStr := peerAddr.String()

		// Register First
		// fmt.Printf("Register Identifier: %s Sender: %s \n", rumor.Origin, peerStr)
		g.dispatcher.taggerListener <- TaggerMessage{
			observerChan: observerChan,
			tagger: StatusTagger{
				sender:     peerStr,
				identifier: rumor.Origin,
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

				fmt.Printf("Receive Peer Status Origin: %s, NextID: %d \n", peerStatus.Identifier, peerStatus.NextID)

				// this means that the peer has received the rumor (in this case, ps.nextID=rumor.ID+1)
				// or it already contains more advanced

				if peerStatus.NextID >= rumor.ID {
					g.updatePeerStatusList(peerStr, peerStatus)
					// check whether the peer has been synced
					if g.isRumorSync(rumor, peerStr) {
						// flip a coin to choose the next one
						g.flipCoinRumorMongering(rumor, GenerateStringSetSingleton(peerStr))
					}

					// channel exit
					unregister()
				}
			case <-timer.C: // Timed out
				// Resend the rumor to another neighbor with prob 1/2
				fmt.Println("TIMEOUT")
				g.flipCoinRumorMongering(rumor, GenerateStringSetSingleton(peerStr))
				unregister()
			}
		}

	}()

	// fmt.Println("MONGERING WITH PEER ", peerAddr)
	g.SendGossipPacket(&GossipPacket{Rumor: rumor}, peerAddr)
}

func (g *Gossiper) flipCoinRumorMongering(rumor *RumorMessage, excludedPeers *StringSet) {
	// 50 - 50
	fmt.Println("Prepare to flip the coin")
	// rand.Seed(time.Now().UTC().UnixNano())
	randInt := rand.Intn(2)
	if randInt == 0 {
		neighborPeer, present := g.RumorMongeringPrepare(rumor, excludedPeers)

		if present {
			fmt.Printf("FLIPPED COIN sending rumor to %s \n", neighborPeer)
		} else {
			fmt.Println("FLIPPED COIN not exist")
		}
	} else {
		fmt.Println("Choose not to flip the coin")
	}
}

func (g *Gossiper) AcceptRumor(r *RumorMessage) {
	// 1. put the rumor in the list
	// 2. update the peer status
	fmt.Printf("Accept Rumor Origin: %s ID: %d \n", r.Origin, r.ID)
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
		g.peerStatuses[r.Origin] = PeerStatus{
			Identifier: r.Origin,
			NextID:     messageID + 1,
		}
	}
}

func (g *Gossiper) isRumorSync(rumor *RumorMessage, peerStr string) bool {
	g.peerWantListLock.Lock()
	defer g.peerWantListLock.Unlock()
	originList, present := g.peerWantList[peerStr]
	if !present {
		return false
	}

	peerStatus, present := originList[rumor.Origin]

	if !present {
		return false
	}

	return peerStatus.NextID > rumor.ID
}

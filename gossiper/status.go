package gossiper

import (
	// check the type of variable

	"fmt"
	"net"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
)

func (g *Gossiper) HandleStatusPacket(s *StatusPacket, sender *net.UDPAddr) {

	// put the peerstatus to the channel
	g.dispatcher.statusListener <- PeerStatusWrapper{
		sender:       sender.String(),
		peerStatuses: s.Want,
	}

	// rumorToSend, and rumorToAsk is []PeerStatus
	rumorToSend, rumorToAsk := g.ComputePeerStatusDiff(s.Want)

	// fmt.Println("TO SEND")
	// fmt.Println(rumorToSend)
	// fmt.Println("TO ASK")
	// fmt.Println(rumorToAsk)
	if len(rumorToSend) > 0 {
		// Just get first is not good
		// firstPeerStatus := rumorToSend[0]
		// g.rumorListLock.Lock()
		// firstRumor := g.rumorList[firstPeerStatus.Identifier][firstPeerStatus.NextID]
		// TODO: mongering first or send the status to the dispatcher first????

		firstRumor := g.FindMostUrgent(s.Want)

		if firstRumor == nil {
			firstPeerStatus := rumorToSend[0]
			g.rumorListLock.Lock()
			firstObject := g.rumorList[firstPeerStatus.Identifier][firstPeerStatus.NextID]
			g.rumorListLock.Unlock()
			firstRumor = firstObject
		}

		go g.RumorMongering(firstRumor, sender)

	}

	if len(rumorToAsk) > 0 {
		// deal with the case to send rumorToAsk
		go g.SendGossipPacket(g.CreateStatusPacket(), sender)

	}

	if len(rumorToSend) == 0 && len(rumorToAsk) == 0 {
		// OUTPUT-HW1
		if HW1OUTPUT {
			fmt.Printf("IN SYNC WITH %s \n", sender)
		}
	}
}

func (g *Gossiper) RumorStatusCheck(gp *GossipPacket) int {
	// To Check the status of the rumor message
	// If diff == 0, rightly updated, therefore
	// If diff > 0, the rumorMessage is head of the record peerStatus
	// if diff < 0, the rumorMessage is behind the record peerStatus
	g.peerStatusesLock.Lock()
	defer g.peerStatusesLock.Unlock()

	var origin string
	var id uint32

	if gp.Rumor != nil {
		origin = gp.Rumor.Origin
		id = gp.Rumor.ID
	} else if gp.TLCMessage != nil {
		origin = gp.TLCMessage.Origin
		id = gp.TLCMessage.ID
	} else {
		fmt.Println("The packet is illegal!!!")
	}
	_, ok := g.peerStatuses[origin]

	if !ok {
		// fmt.Println("This origin does not EXIST")
		// if r.ID == 1 {
		peerStatus := PeerStatus{
			Identifier: origin,
			NextID:     1,
		}
		g.peerStatuses[origin] = peerStatus
		// }
	}

	return int(g.peerStatuses[origin].NextID) - int(id)

}

func (g *Gossiper) updatePeerStatusList(peerStr string, peerStatus PeerStatus) bool {
	// add the new peerStatus in the local peer
	g.peerWantListLock.Lock()
	defer g.peerWantListLock.Unlock()

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
		if previousPeerStatus.NextID >= peerStatus.NextID {
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

func (g *Gossiper) FindMostUrgent(peerWant []PeerStatus) *GossipPacket {
	// make the same node
	nodepeer := make([]string, 0)
	nodeself := make([]string, 0)

	for _, psp := range peerWant {
		nodepeer = append(nodepeer, psp.Identifier)
	}

	g.peerStatusesLock.Lock()
	for _, pss := range g.peerStatuses {
		nodeself = append(nodeself, pss.Identifier)
	}
	g.peerStatusesLock.Unlock()

	nodepeerSet := GenerateStringSet(nodepeer)

	var target string
	gap := 0
	tmp := 0
	var idWant uint32

	for _, psp := range peerWant {
		if nodepeerSet.Has(psp.Identifier) {
			tmp = int(g.peerStatuses[psp.Identifier].NextID) - int(psp.NextID)
			if tmp > gap {
				gap = tmp
				target = psp.Identifier
				idWant = psp.NextID
			}
		}
	}

	if gap == 0 {
		return nil
	}

	newRumor := g.rumorList[target][idWant]
	if DEBUG {
		fmt.Printf("The most urgent is Origin: %s ID: %d \n", target, idWant)
	}

	return newRumor
}

func (g *Gossiper) ComputePeerStatusDiff(peerWant []PeerStatus) (rumorToSend, rumorToAsk []PeerStatus) {
	// TODO Check the concurrent map iteration and map write
	rumorToSend = make([]PeerStatus, 0)
	rumorToSend = make([]PeerStatus, 0)
	// record the ww
	peerOrigins := make([]string, 0)

	// local peerstatus : map[origin]PeerStatus

	for _, pw := range peerWant {
		peerOrigins = append(peerOrigins, pw.Identifier)

		g.peerStatusesLock.Lock()
		localStatus, present := g.peerStatuses[pw.Identifier]
		g.peerStatusesLock.Unlock()

		if !present {
			rumorToAsk = append(rumorToAsk, PeerStatus{Identifier: pw.Identifier, NextID: 1})
		} else if localStatus.NextID < pw.NextID {
			// it means we can ask the peer what we want
			rumorToAsk = append(rumorToAsk, localStatus)
		} else if localStatus.NextID > pw.NextID {
			rumorToSend = append(rumorToSend, pw)
		}
	}
	// our StringSet has provided some useful api
	peerOriginsSet := GenerateStringSet(peerOrigins)

	g.peerStatusesLock.Lock()

	for localPeer, _ := range g.peerStatuses {
		if !peerOriginsSet.Has(localPeer) {
			rumorToSend = append(rumorToSend, PeerStatus{Identifier: localPeer, NextID: 1})
		}
	}

	g.peerStatusesLock.Unlock()
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

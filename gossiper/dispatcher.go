package gossiper

import (
	"fmt"
)

// GoRoutine Dispatcher
// 1. we need one routine to receive GossipPacket from peers: func ReceiveFromPeers()
// 2. we need one routine to receive Message from clients: func ReceiveFromClients()
type Dispatcher struct {
	statusListener chan PeerStatusWrapper
	taggerListener chan TaggerMessage
}

func (d *Dispatcher) Run() {
	// tagger will indicate the origin(identifier) and the target(sender) to classifier the peerStatus received
	taggers := make(map[StatusTagger](map[PeerStatusObserver]bool))
	// observer is received key-value of taggers
	observers := make(map[PeerStatusObserver]StatusTagger)

	for {
		select {
		case taggerMsg := <-d.taggerListener:
			switch taggerMsg.taggerMsgType {
			case TakeIn:

				if DEBUG {
					fmt.Printf("Register sender: %s, identifier: %s \n", taggerMsg.tagger.sender, taggerMsg.tagger.identifier)
				}

				tagger := taggerMsg.tagger

				_, present := taggers[tagger]

				if !present {
					taggers[tagger] = make(map[PeerStatusObserver]bool)
				}

				// the boolean value "true" is useless for this case
				taggers[tagger][taggerMsg.observerChan] = true
				observers[taggerMsg.observerChan] = tagger

			case TakeOut:
				// Close the channel
				toClosedObserver := taggerMsg.observerChan
				tagger, present := observers[toClosedObserver]

				if present {
					// remove tagger from the local mapping

					if DEBUG {
						fmt.Printf("Unregister sender: %s, identifier: %s \n", tagger.sender, tagger.identifier)
					}
					delete(taggers[tagger], toClosedObserver)
					delete(observers, toClosedObserver)

					// close the channel
					close(toClosedObserver)
				} else {
					// panic(fmt.Sprintf("Origin: %s; Sender: %s NOT REGISTERED", taggerMsg.tagger.identifier, taggerMsg.tagger.sender))
					if DEBUG {
						fmt.Printf("Origin: %s; Sender: %s NOT REGISTERED \n", taggerMsg.tagger.identifier, taggerMsg.tagger.sender)
					}
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

				if DEBUG {
					fmt.Printf("Tagger: Identifier: %s, Sender: %s \n", tagger.identifier, tagger.sender)
				}

				for activeChan, _ := range activeChans {
					// observerChan is renamed as activeChan here
					activeChan <- peerStatus
				}
			}
		}
	}
}

func StartPeerStatusDispatcher() *Dispatcher {
	// TODO: whether the statusListener and registerListener should be buffered or unbuffered
	sListener := make(chan PeerStatusWrapper, CHANNEL_BUFFER_SIZE)
	rListener := make(chan TaggerMessage, CHANNEL_BUFFER_SIZE)

	dispatch := &Dispatcher{statusListener: sListener, taggerListener: rListener}
	go dispatch.Run()
	return dispatch
}

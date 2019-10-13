package gossiper

import "fmt"

type Dispatcher struct {
	statusListener   chan PeerStatusWrapper
	registerListener chan RegisterMessage
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

				_, present := taggers[tagger]

				if !present {
					taggers[tagger] = make(map[PeerStatusObserver]bool)
				}

				// the boolean value "true" is useless for this case
				taggers[tagger][registerMsg.observerChan] = true
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

func StartPeerStatusDispatcher() *Dispatcher {
	// TODO: whether the statusListener and registerListener should be buffered or unbuffered
	sListener := make(chan PeerStatusWrapper, CHANNEL_BUFFER_SIZE)
	rListener := make(chan RegisterMessage, CHANNEL_BUFFER_SIZE)

	dispatch := &Dispatcher{statusListener: sListener, registerListener: rListener}
	go dispatch.Run()
	return dispatch
}

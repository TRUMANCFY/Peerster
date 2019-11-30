package gossiper

import (
	"fmt"

	. "github.com/TRUMANCFY/Peerster/message"
)

type BlockPublishDispatcher struct {
	registerChan chan *BlockPublishRegisterTag
	tlcAckChan   chan *TLCAck
}

type BlockPublishRegisterTag struct {
	TagID    uint32
	observer chan<- *TLCAck
	msgType  TaggerMessageType
}

func NewBlockPublishDispatcher() *BlockPublishDispatcher {
	registerChan := make(chan *BlockPublishRegisterTag, CHANNEL_BUFFER_SIZE)
	tlcAckChan := make(chan *TLCAck, CHANNEL_BUFFER_SIZE)

	dispatcher := &BlockPublishDispatcher{
		registerChan: registerChan,
		tlcAckChan:   tlcAckChan,
	}

	go dispatcher.WatchTLCAck()

	return dispatcher
}

func (bp *BlockPublishHandler) RegisterBlockPublish(bpw *BlockPublishWatcher) {
	bp.blockPublishDispatcher.registerChan <- &BlockPublishRegisterTag{
		TagID:    bpw.id,
		observer: bpw.ackChan,
		msgType:  TakeIn,
	}
}

func (bp *BlockPublishHandler) UnregisterBlockPublish(bpw *BlockPublishWatcher) {
	bp.blockPublishDispatcher.registerChan <- &BlockPublishRegisterTag{
		TagID:   bpw.id,
		msgType: TakeOut,
	}
}

func (bpd *BlockPublishDispatcher) WatchTLCAck() {
	// TODO: Monitor the search reply
	// the query mapping is from TagID to the observeChannel
	// queryObserver := make(map[uint32]chan<- *SearchReply)
	ackObserver := make(map[uint32]chan<- *TLCAck)

	go func() {
		for {
			select {
			case regTag := <-bpd.registerChan:
				switch regTag.msgType {
				case TakeIn:
					if DEBUGTLC {
						fmt.Printf("Register TLC Message ID %d \n", regTag.TagID)
					}
					ackObserver[regTag.TagID] = regTag.observer
					break
				case TakeOut:
					fmt.Printf("Unregister TLC Message ID %d \n", regTag.TagID)
					_, present := ackObserver[regTag.TagID]

					if !present {
						if DEBUGTLC {
							fmt.Println("The tlc tag does not exist!!!")
						}
					} else {
						// close the routine
						close(ackObserver[regTag.TagID])
						delete(ackObserver, regTag.TagID)
					}
				}
			case ackReply := <-bpd.tlcAckChan:
				// add to the specific channel
				ackChan, present := ackObserver[ackReply.ID]
				if !present {
					if DEBUGTLC {
						fmt.Println("The tlc tag does not exist")
						continue
					}
				}

				ackChan <- ackReply
			}
		}
	}()
}

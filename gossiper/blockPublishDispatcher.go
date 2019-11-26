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

	// go func() {
	// 	for {
	// 		select {
	// 		case regTag := <-sd.registerChan:
	// 			switch regTag.msgType {
	// 			case TakeIn:
	// 				fmt.Printf("Query Register ID %d \n", regTag.TagID)
	// 				queryObserver[regTag.TagID] = regTag.observer
	// 				break
	// 			case TakeOut:
	// 				fmt.Printf("Query Unregister ID %d \n", regTag.TagID)
	// 				_, present := queryObserver[regTag.TagID]

	// 				if !present {
	// 					if DEBUGSEARCH {
	// 						fmt.Println("The search tag does not exist!")
	// 					}
	// 				} else {
	// 					delete(queryObserver, regTag.TagID)
	// 				}
	// 			}
	// 		case searchReply := <-sd.searchReplyChan:
	// 			for _, queryChan := range queryObserver {
	// 				queryChan <- searchReply
	// 			}
	// 		}
	// 	}
	// }()
}

package gossiper

import (
	"fmt"

	. "github.com/TRUMANCFY/Peerster/message"
)

type SearchDispatcher struct {
	registerChan    chan *QueryRegisterTag
	searchReplyChan chan *SearchReply
}

type QueryRegisterTag struct {
	TagID    uint32
	observer chan<- *SearchReply
	msgType  TaggerMessageType
}

func NewSearchReplyDispatcher() *SearchDispatcher {
	registerChan := make(chan *QueryRegisterTag, CHANNEL_BUFFER_SIZE)
	searchReplyChan := make(chan *SearchReply, CHANNEL_BUFFER_SIZE)

	dispatcher := &SearchDispatcher{
		registerChan:    registerChan,
		searchReplyChan: searchReplyChan,
	}

	go dispatcher.WatchSearchReply()

	return dispatcher
}

func (f *FileHandler) RegisterQuery(q *Query) {
	f.searchDispatcher.registerChan <- &QueryRegisterTag{
		TagID:    q.id,
		observer: q.replyChan,
		msgType:  TakeIn,
	}
}

func (f *FileHandler) UnregisterQuery(q *Query) {
	f.searchDispatcher.registerChan <- &QueryRegisterTag{
		TagID:   q.id,
		msgType: TakeOut,
	}
}

func (sd *SearchDispatcher) WatchSearchReply() {
	// TODO: Monitor the search reply
	// the query mapping is from TagID to the observeChannel
	queryObserver := make(map[uint32]chan<- *SearchReply)

	go func() {
		for {
			select {
			case regTag := <-sd.registerChan:
				switch regTag.msgType {
				case TakeIn:
					queryObserver[regTag.TagID] = regTag.observer
					break
				case TakeOut:
					_, present := queryObserver[regTag.TagID]

					if !present {
						if DEBUGSEARCH {
							fmt.Println("The search tag does not exist!")
						}
					} else {
						delete(queryObserver, regTag.TagID)
					}
				}
			case searchReply := <-sd.searchReplyChan:
				for _, queryChan := range queryObserver {
					queryChan <- searchReply
				}
			}
		}
	}()
}

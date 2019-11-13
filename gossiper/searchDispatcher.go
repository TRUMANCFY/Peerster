package gossiper

import (
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
}

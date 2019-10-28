package gossiper

import (
	. "github.com/TRUMANCFY/Peerster/message"
)

type ReplyObserver (chan<- *DataReply)

type FileDispatcher struct {
	registerChan chan registerTag
	replyChan    chan *DataReply
}

type registerTag struct {
	observer ReplyObserver
	hash     SHA256_HASH
	tagType  TaggerMessageType
}

func LaunchFileDispatcher() *FileDispatcher {
	registerChan := make(chan registerTag, CHANNEL_BUFFER_SIZE)
	replyChan := make(chan *DataReply, CHANNEL_BUFFER_SIZE)

	dispatcher := &FileDispatcher{
		registerChan: registerChan,
		replyChan:    replyChan,
	}

	go dispatcher.Run()

	return dispatcher
}

func (d *FileDispatcher) Run() {
	// TODO
}

func (d *FileDispatcher) Register(shaHash SHA256_HASH, observer ReplyObserver) {
	d.registerChan <- registerTag{
		observer: observer,
		hash:     shaHash,
		tagType:  TakeIn,
	}
}

func (d *FileDispatcher) Unregister(shaHash SHA256_HASH) {
	d.registerChan <- registerTag{
		hash:    shaHash,
		tagType: TakeOut,
	}
}

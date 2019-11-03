package gossiper

import (
	"fmt"

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
	recordBook := make(map[SHA256_HASH]ReplyObserver)

	go func() {
		for {
			select {
			case regTag := <-d.registerChan:
				switch regTag.tagType {
				case TakeIn:
					_, present := recordBook[regTag.hash]

					if present {
						if DEBUG {
							fmt.Println("There is already observer for this data")
						}
						close(regTag.observer)

						// TODO: think break or continue
						continue
					}

					recordBook[regTag.hash] = regTag.observer

				case TakeOut:
					observer, present := recordBook[regTag.hash]
					if !present {
						panic("There is no observer for this channel")
					}

					close(observer)
					delete(recordBook, regTag.hash)
				}
			case dataReply := <-d.replyChan:
				shaHash, err := HashToSha256(dataReply.HashValue)
				if err != nil {
					panic(err)
				}

				obsRecord, present := recordBook[shaHash]

				if present {
					obsRecord <- dataReply
				}

			}
		}
	}()
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

package gossiper

import (
	"net"
	"sync"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	. "github.com/TRUMANCFY/Peerster/util"
)

type Gossiper struct {
	address            *net.UDPAddr
	conn               *net.UDPConn
	uiAddr             *net.UDPAddr
	uiConn             *net.UDPConn
	name               string
	peersList          *PeersList
	simple             bool
	peerStatuses       map[string]PeerStatus
	peerWantList       map[string](map[string]PeerStatus)
	peerWantListLock   *sync.RWMutex
	rumorList          map[string](map[uint32]RumorMessage)
	rumorListLock      *sync.RWMutex
	simpleList         map[string](map[string]bool)
	simpleListLock     *sync.RWMutex
	dispatcher         *Dispatcher
	currentID          *CurrentID
	toSendChan         chan *GossipPacketWrapper
	antiEntropy        int
	guiAddr            string
	gui                bool
	guiPort            string
	routeTable         *RouteTable
	rtimer             time.Duration
	privateMessageList *PrivateMessageList
	fileHandler        *FileHandler
}

type CurrentID struct {
	currentID uint32
	Mux       *sync.Mutex
}

type PrivateMessageList struct {
	privateMessageList map[string][]PrivateMessage
	Mux                *sync.Mutex
}

type RouteTable struct {
	routeTable map[string]string
	Mux        *sync.Mutex
}

type PeersList struct {
	PeersList *StringSet
	Mux       *sync.Mutex
}

// advance and combined data structure
type GossipPacketWrapper struct {
	sender       *net.UDPAddr
	gossipPacket *GossipPacket
}

type ClientMessageWrapper struct {
	sender *net.UDPAddr
	msg    *Message
}

type MessageReceived struct {
	sender        *net.UDPAddr
	packetContent []byte
}

type TaggerMessageType int

const (
	TakeIn  TaggerMessageType = 0
	TakeOut TaggerMessageType = 1
)

type PeerStatusObserver (chan<- PeerStatus)

type TaggerMessage struct {
	observerChan  PeerStatusObserver
	tagger        StatusTagger
	taggerMsgType TaggerMessageType
}

type PeerStatusWrapper struct {
	sender       string
	peerStatuses []PeerStatus
}

type StatusTagger struct {
	sender     string
	identifier string
}
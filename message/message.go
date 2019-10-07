package message

import (
	"fmt"
)

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

type GossipPacket struct {
	Simple *SimpleMessage
}

type Message struct {
	Text string
}

// String member function is actually overrided method
// please refer to https://stackoverflow.com/questions/13247644/tostring-function-in-go
func (sm *SimpleMessage) String() string {
	return fmt.Sprintf("SIMPLE MESSAGE origin %s from %s contents %s", sm.OriginalName, sm.RelayPeerAddr, sm.Contents)
}

func (gp *GossipPacket) String() string {
	return gp.Simple.String()
}

func (m *Message) String() string {
	return fmt.Sprintf("CLIENT MESSAGE %s", m.Text)
}

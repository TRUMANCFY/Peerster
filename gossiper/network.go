package gossiper

import (
	// check the type of variable
	"net"

	. "github.com/TRUMANCFY/Peerster/message"
	"github.com/dedis/protobuf"
)

func (g *Gossiper) ReceiveFromPeers() <-chan *GossipPacketWrapper {
	res := make(chan *GossipPacketWrapper, CHANNEL_BUFFER_SIZE)
	messageReceiver := ReceiveFromConn(g.conn)

	go func() {
		for {
			var packetReceived GossipPacket
			msg := <-messageReceiver
			protobuf.Decode(msg.packetContent, &packetReceived)
			// if packetReceived.Rumor != nil {
			// 	fmt.Printf("Receive Rumor %s Origin %s ID %d \n", packetReceived.Rumor.Text, packetReceived.Rumor.Origin, packetReceived.Rumor.ID)
			// }
			res <- &GossipPacketWrapper{sender: msg.sender, gossipPacket: &packetReceived}
		}
	}()

	return res
}

func (g *Gossiper) ReceiveFromClients() <-chan *ClientMessageWrapper {
	res := make(chan *ClientMessageWrapper, CHANNEL_BUFFER_SIZE)
	messageReceiver := ReceiveFromConn(g.uiConn)

	go func() {
		for {
			var packetReceived Message
			msg := <-messageReceiver
			protobuf.Decode(msg.packetContent, &packetReceived)
			res <- &ClientMessageWrapper{sender: msg.sender, msg: &packetReceived}
			// OUTPUT
			// fmt.Printf("CLIENT MESSAGE %s \n", packetReceived.Text)
		}
	}()
	return res
}

func ReceiveFromConn(conn *net.UDPConn) <-chan *MessageReceived {
	res := make(chan *MessageReceived, CHANNEL_BUFFER_SIZE)
	go func() {
		for {
			packageBytes := make([]byte, UDP_DATAGRAM_MAX_SIZE)
			_, sender, _ := conn.ReadFromUDP(packageBytes)
			res <- &MessageReceived{sender: sender, packetContent: packageBytes}
		}
	}()

	return res
}

func (g *Gossiper) SendGossipPacket(gp *GossipPacket, target *net.UDPAddr) {
	// if gp.Rumor != nil {
	// 	fmt.Printf("Send %s Origin %s ID %d to %s \n", gp.Rumor.Text, gp.Rumor.Origin, gp.Rumor.ID, target.String())
	// }

	g.toSendChan <- &GossipPacketWrapper{sender: target, gossipPacket: gp}
}

func (g *Gossiper) SendGossipPacketStrAddr(gp *GossipPacket, targetStr string) {
	targetUDPAddr, err := net.ResolveUDPAddr("udp4", targetStr)

	if err != nil {
		panic(err)
	}

	g.SendGossipPacket(gp, targetUDPAddr)
}

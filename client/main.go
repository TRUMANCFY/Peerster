package main

import (
	"flag"
	"fmt"
	"net"

	. "github.com/TRUMANCFY/Peerster/message"
	"github.com/dedis/protobuf"
)

var uiPort = flag.String("UIPort", "1234", "please provide UI Port")
var msg = flag.String("msg", "Hello", "Please provide the message broadcasted")
var dest = flag.String("dest", "", "destination for the private message")

func main() {
	flag.Parse()
	fmt.Printf("UI port is %s \n", *uiPort)
	fmt.Printf("Message is %s \n", *msg)

	address := fmt.Sprintf("127.0.0.1:%s", *uiPort)

	sendMessage := &Message{Text: *msg, Destination: dest}
	packetBytes, err := protobuf.Encode(sendMessage)
	// fmt.Println(packetBytes)
	if err != nil {
		fmt.Println(err)
	}

	conn, err := net.Dial("udp4", address)

	if err != nil {
		fmt.Println(err)
	}

	conn.Write(packetBytes)
}

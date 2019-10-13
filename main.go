package main

import (
	"flag"
	"fmt" // check the type of variable
	"strings"

	. "github.com/TRUMANCFY/Peerster/gossiper"
	. "github.com/TRUMANCFY/Peerster/util"
)

var uiPort = flag.String("UIPort", "8001", "please provide UI Port")
var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "please provide gossip address")
var name = flag.String("name", "nodeA", "please provide the node name")
var peersStr = flag.String("peers", "127.0.0.1:5001, 10.1.1.7:5002", "please provide the peers")
var simple = flag.Bool("simple", false, "type")
var antiEntropy = flag.Int("antiEntropy", 10, "please provide the time interval for antiEntropy")

func main() {

	fmt.Println("Gossiper")
	flag.Parse()
	fmt.Printf("UI port is %s \n", *uiPort)
	fmt.Printf("Gossip address is %s \n", *gossipAddr)
	fmt.Printf("Gossip name is %s \n", *name)
	fmt.Printf("Gossip peers are %s \n", *peersStr)
	fmt.Printf("Simple mode is %t \n", *simple)
	fmt.Printf("AntiEntropy Value is %d \n", *antiEntropy)

	// Split the peers to list
	peersList := GenerateStringSet(strings.Split(*peersStr, ","))

	gossiper := NewGossiper(*gossipAddr, *uiPort, *name, peersList, *simple, *antiEntropy)

	gossiper.Run()
}

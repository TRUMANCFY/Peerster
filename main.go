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
var name = flag.String("name", "293324", "please provide the node name")
var peersStr = flag.String("peers", "", "please provide the peers")
var simple = flag.Bool("simple", false, "type")
var antiEntropy = flag.Int("antiEntropy", 10, "please provide the time interval for antiEntropy")
var gui = flag.Bool("gui", false, "gui??")
var guiPort = flag.String("GUIPort", "8080", "GUI_port fixed to 8080?")
var rtimer = flag.Int("rtimer", 0, "route rumors sending period in seconds, 0 to disable sending of route rumors (default 0)")
var hw3ex2 = flag.Bool("hw3ex2", false, "Run the part of HW3Ex2")
var hw3ex3 = flag.Bool("hw3ex3", false, "Run the part of HW3Ex3")
var hw3ex4 = flag.Bool("hw3ex4", false, "Run the part of HW3Ex4")
var numNodes = flag.Int("N", 0, "Total number of nodes in the network")
var stubbornTimeout = flag.Int("stubbornTimeout", 5, "stubbornTimeout")
var ackAll = flag.Bool("ackAll", false, "Ack To All")

func main() {

	fmt.Println("Gossiper")
	flag.Parse()
	fmt.Printf("UI port is %s \n", *uiPort)
	fmt.Printf("Gossip address is %s \n", *gossipAddr)
	fmt.Printf("Gossip name is %s \n", *name)
	fmt.Printf("Gossip peers are %s \n", *peersStr)
	fmt.Printf("Simple mode is %t \n", *simple)
	fmt.Printf("AntiEntropy Value is %d \n", *antiEntropy)
	fmt.Printf("Route timer value is %d \n", *rtimer)

	// Split the peers to list
	var peersList *StringSet
	if *peersStr == "" {
		peersList = GenerateStringSet(make([]string, 0))
	} else {
		peersList = GenerateStringSet(strings.Split(*peersStr, ","))
	}

	gossiper := NewGossiper(*gossipAddr, *uiPort, *name, peersList, *rtimer, *simple, *antiEntropy, *gui, *guiPort, *hw3ex2, *hw3ex3, *hw3ex4, *numNodes, *stubbornTimeout, *ackAll)

	gossiper.Run()
}

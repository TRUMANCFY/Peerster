package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	. "github.com/TRUMANCFY/Peerster/message"
	"github.com/dedis/protobuf"
)

var uiPort = flag.String("UIPort", "1234", "please provide UI Port")
var msg = flag.String("msg", "", "Please provide the message broadcasted")
var dest = flag.String("dest", "", "destination for the private message")
var file = flag.String("file", "", "file to be indexed by the gossiper, or filename of the requested file")
var request = flag.String("request", "", "metafile hash of the downloaded file")
var keywordsList = flag.String("keywords", "", "keyword search list")
var budget = flag.Uint64("budget", 0, "the budget is 0 we will switch to the exponential mode for search")

func main() {
	flag.Parse()
	fmt.Printf("UI port is %s \n", *uiPort)
	fmt.Printf("Message is %s \n", *msg)

	sendMessage := &Message{}

	// cope with keyword list
	var keywords []string
	if *keywordsList != "" {
		keywords = strings.Split(*keywordsList, ",")
	} else {
		keywords = nil
	}

	fmt.Println(keywords)

	switch {
	case *msg != "" && *dest != "":
		// send private message
		sendMessage.Text = *msg
		sendMessage.Destination = dest
		break
	case *msg != "":
		// send rumor message
		sendMessage.Text = *msg
		break
	case *dest != "" && *file != "" && *request != "":
		// download file
		sendMessage.Destination = dest
		sendMessage.File = file

		decoded, err := hex.DecodeString(*request)
		// we not only need
		// fmt.Printf(​ERROR (Unable to decode hex hash))

		if err != nil && len(decoded) != sha256.Size {
			fmt.Printf("ERROR %cUnable to decode hex hash%c \n", '(', ')')
			os.Exit(1)
		}

		sendMessage.Request = &decoded
		break
	case *file != "":
		// index file
		sendMessage.File = file
		break
	case *keywordsList != "":
		sendMessage.Keywords = keywords
		sendMessage.Budget = *budget
	default:
		// wrong combination
		fmt.Printf("ERROR %cBad argument combination%c \n", '(', ')')
		os.Exit(1)
	}

	packetBytes, err := protobuf.Encode(sendMessage)
	// fmt.Println(packetBytes)
	if err != nil {
		fmt.Println(err)
	}

	// Network transmission
	address := fmt.Sprintf("127.0.0.1:%s", *uiPort)

	conn, err := net.Dial("udp4", address)

	if err != nil {
		fmt.Println(err)
	}

	conn.Write(packetBytes)
}

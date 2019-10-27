package gossiper

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
	"github.com/gorilla/mux"
)

// TODO donot return the emtpy text
func (g *Gossiper) GetMessages() []RumorMessage {
	buffer := make([]RumorMessage, 0)

	for _, l1 := range g.rumorList {
		for _, l2 := range l1 {
			buffer = append(buffer, l2)
		}
	}

	return buffer
}

func (g *Gossiper) MessageHandler(w http.ResponseWriter, r *http.Request) {
	// TODO Message Handler
	switch r.Method {
	case "GET":
		// fmt.Println("MESSAGE GET")

		var messages struct {
			Messages []RumorMessage `json:"messages"`
		}

		messages.Messages = g.GetMessages()

		json.NewEncoder(w).Encode(messages)
	case "POST":
		// fmt.Println("MESSAGE POST")

		var message struct {
			Text string `json:"text"`
		}

		json.NewDecoder(r.Body).Decode(&message)

		fmt.Printf("Receive %v \n", message)
		go g.HandleNewMsg(message.Text)

		g.AckPost(true, w)

	}
}

func (g *Gossiper) HandleNewMsg(text string) {
	cmw := &ClientMessageWrapper{
		msg: &Message{
			Text: text,
		},
		sender: nil,
	}

	g.HandleClientMessage(cmw)
}

func (g *Gossiper) NodeHandler(w http.ResponseWriter, r *http.Request) {
	// TODO Node Handler
	switch r.Method {
	case "GET":
		// fmt.Println("NODE GET")

		var peers struct {
			Nodes []string `json:"nodes"`
		}

		peers.Nodes = g.peersList.PeersList.ToArray()

		json.NewEncoder(w).Encode(peers)

	case "POST":
		// fmt.Println("NODE POST")
		var peer struct {
			Addr string `json:"addr"`
		}

		json.NewDecoder(r.Body).Decode(&peer)

		g.AddPeer(peer.Addr)

		g.randomRumorMongering(peer.Addr)

		g.PrintPeers()

		g.AckPost(true, w)
	}
}

func (g *Gossiper) randomRumorMongering(peerStr string) {
	fmt.Println("Add new peer and rumormongering")
	if len(g.rumorList) > 0 {
		for _, rumor := range g.rumorList {
			fmt.Println(rumor)
			if len(rumor) == 0 {
				continue
			}

			for _, rm := range rumor {
				fmt.Println(rm)
				go g.RumorMongeringAddrStr(&rm, peerStr)
				// go g.SendGossipPacketStrAddr(&GossipPacket{Rumor: &rm}, peerStr)
				// break
			}
			// break
		}
	}

	return
}

func (g *Gossiper) IDHandler(w http.ResponseWriter, r *http.Request) {
	// TODO ID Handler
	if r.Method != "GET" {
		panic("Wrong Method, GET required")
	}

	// fmt.Println("PeerID GET")

	var id struct {
		ID string `json:"id"`
	}

	id.ID = g.name

	json.NewEncoder(w).Encode(id)
}

func (g *Gossiper) AckPost(success bool, w http.ResponseWriter) {
	var response struct {
		Success bool `json:"success"`
	}
	response.Success = success
	json.NewEncoder(w).Encode(response)
}

func (g *Gossiper) ListenToGUI() {
	// fake message
	// g.rumorList["a"] = make(map[uint32]RumorMessage)
	// g.rumorList["a"][0] = RumorMessage{
	// 	Origin: "B",
	// 	ID:     2,
	// 	Text:   "I am good",
	// }

	// receiveResp := make(chan *Response, MAX_RESP)

	r := mux.NewRouter()

	// set up routers
	r.HandleFunc("/message", g.MessageHandler).Methods("GET", "POST")
	r.HandleFunc("/node", g.NodeHandler).Methods("GET", "POST")
	r.HandleFunc("/id", g.IDHandler).Methods("GET")

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./webserver/gui/dist/"))))

	fmt.Printf("Server runs at %s \n", g.guiAddr)

	// if g.fixed {
	// 	fmt.Println("GUI Port is 127.0.0.1:8080")
	// 	g.guiAddr = "127.0.0.1:8080"
	// }

	srv := &http.Server{
		Handler:           r,
		Addr:              g.guiAddr,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

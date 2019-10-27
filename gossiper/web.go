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

func (g *Gossiper) GetMessages() []RumorMessage {
	buffer := make([]RumorMessage, 0)

	for _, l1 := range g.rumorList {
		for _, l2 := range l1 {
			// Check whether text is emtpy
			if l2.Text != "" {
				buffer = append(buffer, l2)
			}
		}
	}

	return buffer
}

func (g *Gossiper) GetPrivateMsgs() []PrivateMessage {
	buffer := make([]PrivateMessage, 0)

	for _, privateMsgs := range g.privateMessageList.privateMessageList {
		buffer = append(buffer, privateMsgs...)
	}

	return buffer
}

func (g *Gossiper) GetRoutes() []string {
	buffer := make([]string, 0)

	for route, _ := range g.routeTable.routeTable {
		buffer = append(buffer, route)
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
	if r.Method != "GET" {
		panic("Wrong Method for peerID, GET required")
	}

	// fmt.Println("PeerID GET")

	var id struct {
		ID string `json:"id"`
	}

	id.ID = g.name

	json.NewEncoder(w).Encode(id)
}

func (g *Gossiper) PrivateHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":

		var privateMsgs struct {
			Msgs []PrivateMessage `json:"msgs"`
		}

		privateMsgs.Msgs = g.GetPrivateMsgs()

		json.NewEncoder(w).Encode(privateMsgs)

	case "POST":
		// fmt.Println("Private POST")
		var privateMsgDest struct {
			Text string `json:"text"`
			Dest string `json:"dest"`
		}

		json.NewDecoder(r.Body).Decode(&privateMsgDest)

		privateMsg := PrivateMessage{
			Origin:      g.name,
			ID:          0,
			Text:        privateMsgDest.Text,
			Destination: privateMsgDest.Dest,
			HopLimit:    HOPLIMIT,
		}

		fmt.Println(privateMsg)

		go g.SendPrivateMessage(&privateMsg)

		g.AckPost(true, w)
	}
}

func (g *Gossiper) RouteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		panic("Wrong Method for Route, GET required")
	}

	var routes struct {
		Targets []string `json:"targets"`
	}

	routes.Targets = g.GetRoutes()

	json.NewEncoder(w).Encode(routes)
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

	// fake data for private message

	g.routeTable.routeTable["A"] = "127.0.0.1:5002"
	g.routeTable.routeTable["B"] = "127.0.0.1:5003"

	tmp1 := make([]PrivateMessage, 0)
	tmp1 = append(tmp1, PrivateMessage{
		Origin: "A",
		Text:   "A1",
	})

	tmp1 = append(tmp1, PrivateMessage{
		Origin: "A",
		Text:   "A2",
	})

	tmp2 := make([]PrivateMessage, 0)
	tmp2 = append(tmp2, PrivateMessage{
		Origin: "B",
		Text:   "B1",
	})

	tmp2 = append(tmp2, PrivateMessage{
		Origin: "B",
		Text:   "B2",
	})

	g.privateMessageList.privateMessageList["A"] = tmp1
	g.privateMessageList.privateMessageList["B"] = tmp2

	r := mux.NewRouter()

	// set up routers
	r.HandleFunc("/message", g.MessageHandler).Methods("GET", "POST")
	r.HandleFunc("/node", g.NodeHandler).Methods("GET", "POST")
	r.HandleFunc("/id", g.IDHandler).Methods("GET")

	// add new private features
	r.HandleFunc("/private", g.PrivateHandler).Methods("GET", "POST")
	r.HandleFunc("/routes", g.RouteHandler).Methods("GET")

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

package gossiper

import (
	"encoding/hex"
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
			if l2.Rumor != nil {
				if l2.Rumor.Text != "" {
					buffer = append(buffer, *l2.Rumor)
				}
			}
		}
	}

	return buffer
}

func (g *Gossiper) GetSearchedFiles() []string {
	searchedFiles := make([]string, 0)
	g.fileHandler.searchFiles.Mux.Lock()
	for _, sf := range g.fileHandler.searchFiles.searchedFiles {
		for _, csrc := range sf.chunkSrc {
			if !Contains(searchedFiles, csrc.Filename) {
				searchedFiles = append(searchedFiles, csrc.Filename)
			}
		}
	}
	g.fileHandler.searchFiles.Mux.Unlock()

	return searchedFiles
}

func Contains(strList []string, subStr string) bool {
	for _, strIn := range strList {
		if strIn == subStr {
			return true
		}
	}
	return false
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
		if route != g.name {
			buffer = append(buffer, route)
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
				if rm.TLCMessage != nil {
					continue
				}
				go g.RumorMongeringAddrStr(rm, peerStr)
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

func (g *Gossiper) FileIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		panic("Wrong Method for File, POST Required")
	}

	var file struct {
		FileName string `json:"filename"`
	}
	json.NewDecoder(r.Body).Decode(&file)

	fmt.Println(file.FileName)

	go g.FileIndexingRequest(file.FileName)

	g.AckPost(true, w)
}

func (g *Gossiper) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		panic("Wrong Method for Download, POST Required")
	}

	var download struct {
		Dest     string `json:"dest"`
		Hex      string `json:"hex"`
		FileName string `json:"filename"`
	}

	json.NewDecoder(r.Body).Decode(&download)

	fmt.Println("DOWNLOADREQ")
	fmt.Println(download)

	dest := download.Dest
	metaHash, err := hex.DecodeString(download.Hex)

	if err != nil {
		g.AckPost(false, w)
		return
	}

	filename := download.FileName

	metaSha, err := HashToSha256(metaHash)

	if err != nil {
		g.AckPost(false, w)
		return
	}

	result := g.RequestFile(dest, metaSha, filename)

	if result {
		g.AckPost(true, w)
		return
	}

	g.AckPost(false, w)

}

func (g *Gossiper) RoundMsg(w http.ResponseWriter, r *http.Request) {
	// g.roundHandler.roundMsg.roundMsg
	if r.Method == "POST" {
		return
	}

	var roundMsg struct {
		Msgs []string `json:"msgs"`
	}
	g.roundHandler.roundMsg.Mux.Lock()
	roundMsg.Msgs = g.roundHandler.roundMsg.roundMsg
	g.roundHandler.roundMsg.Mux.Unlock()

	json.NewEncoder(w).Encode(roundMsg)
}

func (g *Gossiper) ConfirmedMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		return
	}

	var confirmedMsg struct {
		Msgs []string `json:"msgs"`
	}

	confirmedMsg.Msgs = g.GetConfirmedMsg()

	json.NewEncoder(w).Encode(confirmedMsg)
}

func (g *Gossiper) GetConfirmedMsg() []string {
	res := make([]string, 0)
	g.rumorListLock.Lock()
	defer g.rumorListLock.Unlock()

	for _, idMap := range g.rumorList {
		for _, gp := range idMap {
			if gp.TLCMessage != nil {
				if gp.TLCMessage.Confirmed > 0 {
					tlcMessage := gp.TLCMessage
					res = append(res, fmt.Sprintf("CONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %x \n",
						tlcMessage.Origin,
						tlcMessage.ID,
						tlcMessage.TxBlock.Transaction.Name,
						tlcMessage.TxBlock.Transaction.Size,
						hex.EncodeToString(tlcMessage.TxBlock.Transaction.MetafileHash)))
				}
			}
		}
	}
	return res
}

func (g *Gossiper) SearchHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		var searchedFiles struct {
			Files []string `json:"files"`
		}

		searchedFiles.Files = g.GetSearchedFiles()

		json.NewEncoder(w).Encode(searchedFiles)

	case "POST":
		// fmt.Println("Private POST")
		var searchedReq struct {
			Keywords string `json:"keywords"`
			Budget   uint64 `json:"budget"`
		}

		json.NewDecoder(r.Body).Decode(&searchedReq)

		var strPointer = new(string)
		*strPointer = searchedReq.Keywords

		var budgetPointer = new(uint64)
		*budgetPointer = searchedReq.Budget

		cmw := &ClientMessageWrapper{
			msg: &Message{
				Keywords: strPointer,
				Budget:   budgetPointer,
			},
		}

		go g.HandleClientSearch(cmw)

	}
}

func (g *Gossiper) SearchDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		return
	}

	var searchDownloadReq struct {
		FileName string `json:"filename"`
	}
	json.NewDecoder(r.Body).Decode(&searchDownloadReq)

	fmt.Println(searchDownloadReq.FileName)

	sha, valid := g.findSHA(searchDownloadReq.FileName)

	if !valid {
		return
	}

	go g.RequestSearchedFile(sha, searchDownloadReq.FileName)

	g.AckPost(true, w)
}

func (g *Gossiper) findSHA(filename string) (SHA256_HASH, bool) {
	var sha SHA256_HASH

	g.fileHandler.searchFiles.Mux.Lock()
	defer g.fileHandler.searchFiles.Mux.Unlock()

	for _, sf := range g.fileHandler.searchFiles.searchedFiles {
		for _, csrc := range sf.chunkSrc {
			if csrc.Filename == filename {
				return sf.MetafileHash, true
			}
		}
	}

	return sha, false
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

	// g.routeTable.routeTable["A"] = "127.0.0.1:5002"
	// g.routeTable.routeTable["B"] = "127.0.0.1:5003"

	// tmp1 := make([]PrivateMessage, 0)
	// tmp1 = append(tmp1, PrivateMessage{
	// 	Origin: "A",
	// 	Text:   "A1",
	// })

	// tmp1 = append(tmp1, PrivateMessage{
	// 	Origin: "A",
	// 	Text:   "A2",
	// })

	// tmp2 := make([]PrivateMessage, 0)
	// tmp2 = append(tmp2, PrivateMessage{
	// 	Origin: "B",
	// 	Text:   "B1",
	// })

	// tmp2 = append(tmp2, PrivateMessage{
	// 	Origin: "B",
	// 	Text:   "B2",
	// })

	// g.privateMessageList.privateMessageList["A"] = tmp1
	// g.privateMessageList.privateMessageList["B"] = tmp2

	r := mux.NewRouter()

	// set up routers
	r.HandleFunc("/message", g.MessageHandler).Methods("GET", "POST")
	r.HandleFunc("/node", g.NodeHandler).Methods("GET", "POST")
	r.HandleFunc("/id", g.IDHandler).Methods("GET")

	// add new private features
	r.HandleFunc("/private", g.PrivateHandler).Methods("GET", "POST")
	r.HandleFunc("/routes", g.RouteHandler).Methods("GET")
	r.HandleFunc("/file", g.FileIndexHandler).Methods("POST")
	r.HandleFunc("/download", g.DownloadHandler).Methods("POST")

	// add search function
	r.HandleFunc("/search", g.SearchHandler).Methods("GET", "POST")
	r.HandleFunc("/searchDownload", g.SearchDownloadHandler).Methods("POST")
	if g.hw3ex2 || g.hw3ex3 {
		r.HandleFunc("/confirmedMsg", g.ConfirmedMsg).Methods("GET")
	}

	if g.hw3ex3 {
		r.HandleFunc("/roundMsg", g.RoundMsg).Methods("GET")
	}

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./webserver/gui/dist/"))))

	fmt.Printf("Server runs at %s \n", g.guiAddr)

	srv := &http.Server{
		Handler:           r,
		Addr:              g.guiAddr,
		WriteTimeout:      15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

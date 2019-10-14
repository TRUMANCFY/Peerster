package gossiper

// import (
// 	. "github.com/TRUMANCFY/Peerster/message"
// )

// type RequestType int

// const (
// 	MsgQuery  RequestType = 0
// 	NodeQuery RequestType = 1
// 	IDQuery   RequestType = 2
// )

// type CommType int

// const (
// 	GET  CommType = 0
// 	POST CommType = 1
// )

// type GetRequest struct {
// 	reqType RequestType
// }

// type PeerComing struct {
// 	peer string
// }

// type PostRequest struct {
// 	msgcoming  *Message
// 	peercoming *PeerComing
// }

// type Response struct {
// 	reqType RequestType
// 	rumors  []RumorMessage
// 	pid     string
// 	peers   []string
// }

// type Request struct {
// 	commType CommType
// 	getReq   *GetRequest
// 	postReq  *PostRequest
// }

// type TestMessage struct {
// 	Name string
// 	Body string
// 	Time int64
// }

// const MAX_RESP = 1024

// const GUI_ADDR = "127.0.0.1:8080"

// func (g *Gossiper) GetAllRumors() {

// }

// func (g *Gossiper) GetAllPeers() {

// }

// func (g *Gossiper) GetPeerID() {

// }

// func MessageHandler(w http.ResponseWriter, r *http.Request) {
// 	// TODO Message Handler
// 	switch r.Method {
// 	case "GET":
// 		fmt.Println("MESSAGE GET")
// 		m := TestMessage{"Alice", "Hello", 1294706395881547000}
// 		json.NewEncoder(w).Encode(m)
// 	case "POST":
// 		fmt.Println("MESSAGE POST")
// 	}
// }

// func NodeHandler(w http.ResponseWriter, r *http.Request) {
// 	// TODO Node Handler
// 	switch r.Method {
// 	case "GET":
// 		fmt.Println("NODE GET")
// 	case "POST":
// 		fmt.Println("NODE POST")
// 	}
// }

// func IDHandler(w http.ResponseWriter, r *http.Request) {
// 	// TODO ID Handler
// 	if r.Method != "GET" {
// 		panic("Wrong Method, GET required")
// 		return
// 	}

// 	fmt.Println("PeerID GET")
// }

// func (g *Gossiper) ListenToGUI() {
// 	// receiveResp := make(chan *Response, MAX_RESP)

// 	r := mux.NewRouter()

// set up routers
// 	r.HandleFunc("/message", MessageHandler).Methods("GET", "POST")
// 	r.HandleFunc("/node", NodeHandler).Methods("GET", "POST")
// 	r.HandleFunc("/id", IDHandler).Methods("GET")

// 	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("../webserver/gui/dist/"))))
// 	srv := &http.Server{
// 		Handler:           r,
// 		Addr:              GUI_ADDR,
// 		WriteTimeout:      15 * time.Second,
// 		ReadHeaderTimeout: 15 * time.Second,
// 	}

// 	log.Fatal(srv.ListenAndServe())
// }

// func (g *Gossiper) ListenToGUI1(reqListener <-chan *Request) {
// 	for {
// 		req, ok := <-reqListener

// 		if !ok {
// 			continue
// 		}

// 		switch req.commType {
// 		case GET:
// 			fmt.Println("GET")
// 			rType := req.getReq.reqType
// 			switch rType {
// 			case MsgQuery:
// 				fmt.Println("Ask for Msg")
// 				// allRumors := g.GetAllRumors()
// 			case NodeQuery:
// 				fmt.Println("Ask for Node")
// 				// allPeers := g.GetAllPeers()
// 			case IDQuery:
// 				fmt.Println("Ask for PeerID")
// 				// peerID := g.GetPeerID()
// 			}
// 		case POST:
// 			reqPost := req.postReq
// 			switch {
// 			case reqPost.msgcoming != nil:
// 				fmt.Println("Send Message")
// 				g.HandleClientMessage(&ClientMessageWrapper{msg: reqPost.msgcoming})
// 			case reqPost.peercoming != nil:
// 				fmt.Println("Add a peer")
// 				g.AddPeer(reqPost.peercoming.peer)
// 			}
// 		}
// 	}
// }

// func (g *Gossiper) ReceiveFromGUI() <-chan *RequestWrapper {
// 	res := make(chan *RequestWrapper, CHANNEL_BUFFER_SIZE)
// 	messageReceiver := ReceiveFromConn(g.uiConn)

// 	go func() {
// 		for {
// 			var requestReceived Request
// 			msg := <-messageReceiver
// 			protobuf.Decode(msg.packetContent, &requestReceived)
// 			res <- &RequestWrapper{sender: messageReceiver.sender, req: &requestReceived}
// 			// OUTPUT
// 		}
// 	}()
// 	return res
// }

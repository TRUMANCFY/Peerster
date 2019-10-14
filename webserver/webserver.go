package main

import (
	"log"
	"net/http"

	// . "github.com/TRUMANCFY/Peerster/message"
	// . "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	// . "github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("gui/dist/"))))
	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8080",
	}

	log.Fatal(srv.ListenAndServe())
}

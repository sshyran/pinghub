package main

import (
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

var addr = flag.String("addr", ":8081", "http service address")

func main() {
	flag.Parse()

	// Start the HTTP server
	http.Handle("/", newHandler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func newHandler() http.Handler {
	h := newHub()
	go h.run()

	r := mux.NewRouter()

	// Route websocket requests
	r.Headers("Connection", "Upgrade",
		"Upgrade", "websocket").Handler(wsHandler{h: h})

	// Route other GET and POST requests
	r.Methods("GET").Handler(getHandler{h: h})
	r.Methods("POST").Handler(postHandler{h: h})

	return r
}

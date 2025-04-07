package main

import (
	"log"
	"net/http"

	"galaxy.io/server/galaxy"
	"galaxy.io/server/websockets"
)

func main() {
	wsFactory := &websockets.WebsocketFactory{}

	world := galaxy.NewWorld(wsFactory)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		world.HandleNewConnection(w, r)
	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: %v", err)
	}
}

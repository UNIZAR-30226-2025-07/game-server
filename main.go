package main

import (
	"log"
	"os"
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

	ip := os.Getenv("GALAXY_SERVER_IP")
	port := os.Getenv("GALAXY_SERVER_PORT")

	log.Printf("server started in %v:%v", ip, port)
	err := http.ListenAndServe(ip+":"+port, nil)
	if err != nil {
		log.Fatal("ListenAndServe: %v", err)
	}

}

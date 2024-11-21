package main

import (
	"fmt"
	"mithril/websocket"
	"mithril/wsserver"
)

func connection(ws *websocket.Ws, srv *wsserver.Server) (uint16, error) {
	output, err, isClose := ws.Read()
	fmt.Println(string(output), isClose)
	fmt.Println(srv.Clients)
	if string(output) == "test" {
		srv.BroadcastToAll([]byte("Hahaha!"))
	}

	return 0, err
}

func main() {
	wsserver.CreateWebSocket("127.0.0.1", "1233", connection, "/ws")
}

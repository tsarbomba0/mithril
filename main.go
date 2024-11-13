package main

import (
	"fmt"
	"mithril/websocket"
)

func connection(ws *websocket.Ws) {
	bytes := make([]byte, 256)
	ws.Conn.Read(bytes)
	fmt.Println(bytes)
}

func main() {
	websocket.CreateWebSocket("127.0.0.1", "1222", connection, "/ws")
}

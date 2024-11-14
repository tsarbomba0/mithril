package main

import (
	"fmt"
	"mithril/websocket"
)

func connection(ws *websocket.Ws) {
	bytes := make([]byte, 256)
	l, err := ws.Conn.Read(bytes)
	if err != nil {
		panic(err)
	}
	fmt.Println(bytes[:l])
	ws.ReadFrame(bytes[:l])

}

func main() {
	websocket.CreateWebSocket("127.0.0.1", "222", connection, "/ws")
}

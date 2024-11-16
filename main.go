package main

import (
	"fmt"
	"mithril/websocket"
)

func connection(ws *websocket.Ws) error {
	output, err := ws.Read()
	if err != nil {
		if err == fmt.Errorf("empty array of bytes") {
			fmt.Println("Closed conn")
			return err
		}
	}
	ws.Write(output)
	return err
}

func main() {
	websocket.CreateWebSocket("127.0.0.1", "2000", connection, "/ws")
}

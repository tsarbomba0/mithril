package main

import (
	"errors"
	"mithril/websocket"
)

func connection(ws *websocket.Ws) (uint16, error) {
	output, err, isClose := ws.Read()
	if isClose {
		return 1000, errors.New("closed")
	} else if err != nil {
		return 1011, err
	}

	if string(output) == "fake" {
		return 1011, errors.New("test error")
	} else {
		ws.Write(output)
	}

	return 0, err
}

func main() {
	websocket.CreateWebSocket("127.0.0.1", "2000", connection, "/ws")
}

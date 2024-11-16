package main

import (
	"fmt"
	"mithril/websocket"
)

func connection(ws *websocket.Ws) {
	output, err := ws.Read()
	if err != nil {
		panic(err)
	}
	fmt.Println(string(output))
	if string(output) == "t" {
		fmt.Println("Gah!")
		l, err := ws.Write([]byte("Haha!"))
		fmt.Println(l)
		fmt.Println(err)
	}

}

func main() {
	websocket.CreateWebSocket("127.0.0.1", "2000", connection, "/ws")
}

package main

import (
	"fmt"
	"mithril/util"
	"mithril/wsclient"
)

func conn(ws *wsclient.ClientWs) {
	ws.Write([]byte("this stuff works i think"), 130)
	o, err := ws.Read()
	util.OnError(err)
	fmt.Println(string(o))
}

func main() {
	wsclient.ConnectWebSocket("127.0.0.1", "2000", conn)
}

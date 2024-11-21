# Mithril
Mostly functional.
Lacks TLS.

## Goals
Making it secure (TLS) and easy to use.
Make it support the DEFLATE algorithm.

## Examples

### Basic WebSocket server (prints out the received bytes)
```go
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
```

### Client
```go
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
```

## Reason
I wanted to dig some into slightly more low-level stuff and i want to eventually build some weird database based on websockets.



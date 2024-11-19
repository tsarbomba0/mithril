# Mithril
Not yet functional library for handling WebSocket connections.

## Goals
Making it fully functional and easy to use.
Deflate algorithm stuff?

## Examples of functionality

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

Example in main.go rn

## Why?
I wanted to dig some into slightly more low-level stuff and i want to eventually built some weird database based on websockets.



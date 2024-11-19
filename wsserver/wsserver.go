package wsserver

// Import containing the function to create a WebSocket server

import (
	"bufio"
	"io"
	"log"
	"mithril/util"
	"mithril/websocket"
	"net"
)

// Creates a WebSocket server
func CreateWebSocket(addr string, port string, handler func(websocket *websocket.Ws) (uint16, error), routeString string) {
	listener, err := net.Listen("tcp", addr+":"+port)
	util.OnError(err)
	log.Println("Server listening on address: " + addr + " and port: " + port)

	defer listener.Close()

	for {
		// create listener
		connection, err := listener.Accept()
		util.OnError(err)

		// Buffer
		buffer := bufio.NewReadWriter(bufio.NewReader(connection), bufio.NewWriter(connection))

		// goroutine
		go func(ws *websocket.Ws) {
			bytes := make([]byte, 256)
			length, err := ws.Conn.Read(bytes)

			if err == io.EOF {
				util.OnError(err)
			}

			method, route, err := ws.DetermineRequest(bytes[:length])
			util.OnError(err)

			if route == routeString {
				if method != "GET" {
					ws.SendHTTPError("400", "Invalid request method! Was:"+method+" Should be GET.")
					log.Println("Client attempted to connect with an incorrect method. (" + method + ")")
					ws.Conn.Close()
					return
				} else {
					headers := ws.GetHTTPHeaders(bytes[:length])
					log.Println("Obtained headers from HTTP request.")

					ws.ServerHandshake(headers["Sec-WebSocket-Key"])
					log.Println("Sent handshake to client.")

					for {
						status, err := handler(ws)
						if err != nil {
							log.Println("exception: ", err, " Closing connection.")
							ws.Close(status, err.Error())
							break
						}
					}
				}
			} else {
				// Send error and disconnect
				ws.SendHTTPError("400", "Invalid route! ("+route+")")
				ws.Conn.Close()
			}
		}(&websocket.Ws{Conn: connection, Buffer: buffer})
	}
}

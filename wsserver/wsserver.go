package wsserver

// Import containing the function to create a WebSocket server

import (
	"bufio"
	"io"
	"log"
	"mithril/util"
	"mithril/websocket"
	"net"
	"strconv"
)

type Server struct {
	Listener net.Listener
	Clients  []*websocket.Ws
	Address  string
	Port     string
}

// Returns a Server struct.
func initServer(address string, port string) *Server {
	listener, err := net.Listen("tcp", address+":"+port)
	util.OnError(err)

	log.Println("Server listening on address: " + address + " and port: " + port)

	var clientSlice []*websocket.Ws

	return &Server{
		Listener: listener,
		Clients:  clientSlice,
		Address:  address,
		Port:     port,
	}
}

// Accepts a TCP connection.
//
// Returns a net.Conn.
func (srv *Server) acceptConnection() *websocket.Ws {
	// Accepting conneciton
	connection, err := srv.Listener.Accept()
	util.OnError(err)

	// Buffer
	buffer := bufio.NewReadWriter(bufio.NewReader(connection), bufio.NewWriter(connection))
	// WebSocket instance
	ws := &websocket.Ws{Conn: connection, Buffer: buffer}

	srv.Clients = append(srv.Clients, ws)
	log.Println("Host " + connection.LocalAddr().String() + " connected.")

	return ws
}

// Broadcasts data to all clients.
func (srv *Server) BroadcastToAll(data []byte) {
	for i := 0; i <= len(srv.Clients)-1; i++ {
		_, err := srv.Clients[i].Write(data)
		if err != nil {
			srv.Clients[i] = srv.Clients[len(srv.Clients)-1]
			srv.Clients = srv.Clients[:len(srv.Clients)-1]
			log.Println(err)
		}
	}
	log.Println("Broadcasted " + strconv.Itoa(len(data)) + " bytes of data to all clients.")

}

// Creates a WebSocket server
func CreateWebSocket(addr string, port string, handler func(websocket *websocket.Ws, server *Server) (uint16, error), routeString string) {
	server := initServer(addr, port)
	defer server.Listener.Close()
	for {
		// create listener
		websocketInstance := server.acceptConnection()

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
						status, err := handler(ws, server)
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
		}(websocketInstance)
	}
}

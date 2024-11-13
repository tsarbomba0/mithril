package websocket

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
)

var CloseCodes map[int]string = map[int]string{
	1000: "NormalClosure",
	1001: "GoingAway",
	1002: "ProtocolError",
	1003: "UnknownType",
	1007: "InvalidPayloadData",
	1008: "PolicyViolation",
	1009: "MessageTooBig",
	1010: "ExtensionError",
	1011: "InternalError",
}

// TODO: Expand this
var HttpErrorCodes map[string]string = map[string]string{
	"400": "Bad request",
	"401": "Unauthorized",
	"403": "Forbidden",
	"404": "Not Found",
	"405": "Method Not Allowed",
	"406": "Not Acceptable",
	"407": "Proxy Authentication Required",
	"408": "Request Timeout",
}

type Ws struct {
	Conn   net.Conn
	Status uint8
}

// Error handler
func onError(err error) {
	if err != nil {
		panic(err)
	}
}

// Creates a WebSocket
func CreateWebSocket(addr string, port string, handler func(websocket *Ws), routeString string) {
	listener, err := net.Listen("tcp", addr+":"+port)
	onError(err)
	defer listener.Close()

	for {
		// create listener
		connection, err := listener.Accept()
		onError(err)

		// goroutine
		go func(ws *Ws) {
			bytes := make([]byte, 256)
			length, err := ws.Conn.Read(bytes)
			if err == io.EOF {
				err = nil
			} else {
				onError(err)
			}

			method, route, err := ws.determineRequest(bytes[:length])
			onError(err)

			if route == routeString {
				if method != "GET" {
					ws.Conn.Write([]byte("Invalid request method! Should be GET."))
					ws.Conn.Close()
					return
				} else {
					headers := ws.GetHTTPHeaders(bytes[:length])
					ws.ServerHandshake(headers["Sec-WebSocket-Key"])
					for {
						handler(ws)
					}
				}

			} else {
				// this to be replaced with proper http response
				ws.SendHTTPError("400", "Invalid route! ("+route+")")
				ws.Conn.Close()
			}
		}(&Ws{Conn: connection})

	}
}

// Creates the hash used to accept a WebSocket connection.
func (ws Ws) AcceptHash(k string) string {
	h := sha1.New()
	h.Write([]byte(k))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Returns a handshake to be sent via HTTP.
func (ws Ws) ServerHandshake(secretKey string) {
	var req strings.Builder
	req.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %s", ws.AcceptHash(secretKey)+"\r\n\r\n"))

	ws.Conn.Write([]byte(req.String()))
}

// Gets HTTP Headers
func (ws Ws) GetHTTPHeaders(dataBytes []byte) map[string]string {
	// Map to hold the headers
	settings := make(map[string]string)

	// Actual slice of headers from the request
	headers := strings.Split(string(dataBytes), "\r\n")

	// Iterating and attaching to the map (excluding the first entry, it's the HTTP version string)
	for i := 1; i < len(headers)-1; i++ {
		// splitting the string from the header
		re := regexp.MustCompile(`:\s`)
		splitString := re.Split(headers[i], -1)

		// if length of the split string is equal to 1 (empty slice) AND the next slice is not empty, do a error
		fmt.Println(splitString, len(splitString))
		if len(splitString) == 1 && len(re.Split(headers[i+1], -1)) != 1 {
			onError(fmt.Errorf("unexpected separation of headers! (line %d)", i))

			// else assign the value to the map
		} else if len(splitString) != 1 {
			key, value := splitString[0], splitString[1]
			settings[key] = value
		}
	}
	return settings
}

// Function to determine the type of the request and the route
func (ws Ws) determineRequest(byteArray []byte) (string, string, error) {
	buf := make([]byte, 128)
	reader := bytes.NewReader(byteArray)
	index, err := reader.Read(buf)

	re, _ := regexp.Compile("/[A-Za-z]+")
	route := re.FindString(string(buf[:index]))

	re, _ = regexp.Compile("POST|GET|PATCH|DELETE|PUT")
	method := re.FindString(string(buf[:index]))

	return method, route, err

}

// Function to send a HTTP error response
func (ws Ws) SendHTTPError(errorCode string, reason string) {
	var response strings.Builder
	response.WriteString("HTTP/1.1 " + errorCode + " " + HttpErrorCodes[errorCode] + "\r\n")
	response.WriteString("Content-Type: text/plain\r\n")
	response.WriteString("Content-Language: en\r\n\r\n")
	response.WriteString(reason + "\r\n\r\n")
	ws.Conn.Write([]byte(response.String()))
}

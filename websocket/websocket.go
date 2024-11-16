package websocket

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
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

// WebSocket type
type Ws struct {
	Conn   net.Conn
	Status uint8
	Buffer *bufio.ReadWriter
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

		// Buffer
		buffer := bufio.NewReadWriter(bufio.NewReader(connection), bufio.NewWriter(connection))
		fmt.Println("Hah!")
		// goroutine
		go func(ws *Ws) {
			bytes := make([]byte, 256)
			length, err := ws.Conn.Read(bytes)
			if err == io.EOF {
				ws.Conn.Close()
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

					testBytes := make([]byte, 2)
					for {
						// Handles a closed connection
						// Closed connection defined as
						// either receiving a empty bytearray
						l, _ := ws.Conn.Read(testBytes)
						//fmt.Println(l)
						if l > 0 && testBytes[0] != 136 {
							handler(ws)
						} else {
							ws.Conn.Write(testBytes)
							ws.Conn.Close()
							fmt.Println("Closed!")
							return
						}

					}

				}

			} else {
				// this to be replaced with proper http response
				ws.SendHTTPError("400", "Invalid route! ("+route+")")
				ws.Conn.Close()
			}
		}(&Ws{Conn: connection, Buffer: buffer})

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

// Reads a frame and returns it as a string.
// Returns a string and error
// string is empty if the frame isn't masked
// error is NOT nil if the frame isn't masked
func (ws *Ws) readFrame(frame []byte) ([]byte, error) {
	// error variable
	fmt.Println(frame)
	var err error = nil
	// decoded payload
	var decodedPayload bytes.Buffer

	// TODO: info from the first byte
	// info = frame[0]
	fin := frame[0] & 128
	rsv1 := frame[0] & 64
	rsv2 := frame[0] & 32
	rsv3 := frame[0] & 16
	opcode := frame[0] & 15
	fmt.Println(fin, rsv1, rsv2, rsv3, opcode)

	// mask variable
	var mask []byte

	// payload variable
	var payload []byte

	// fix
	if frame[1]&128 == 0 {
		err = errors.New("unmasked frame")
		fmt.Println(frame[0])
	} else {
		if frame[1]&127 == 127 {
			mask = frame[11:15]
			payload = frame[15:]

			for i := 0; i != len(payload); i++ {
				decodedPayload.WriteByte(payload[i] ^ mask[i%4])
			}

		} else if frame[1]&127 == 126 {
			mask = frame[4:8]
			payload = frame[8:]

			for i := 0; i != len(payload); i++ {
				decodedPayload.WriteByte(payload[i] ^ mask[i%4])
			}

		} else if frame[1]&127 == 125 {
			fmt.Println("Control frame!")
		} else {
			// mask
			mask = frame[2:6]
			// actual payload data
			payload = frame[6:]

			// performing a XOR on payload byte with mask byte
			for i := 0; i != len(payload); i++ {
				decodedPayload.WriteByte(payload[i] ^ mask[i%4])
			}
		}

	}

	// return string and error
	return decodedPayload.Bytes(), err
}

// create a frame
func (ws *Ws) WriteFrame(content []byte) []byte {
	var header []byte = make([]byte, 2)
	header[0] = byte(130)

	// data variable
	var data []byte

	// if length is less than 125, just attach it as a integer to the data array
	if len(content) <= 125 {
		header[1] = byte(len(content))
		data = header
		// if
	} else if 65535 >= len(content) {
		header[1] = 126

		var size []byte
		binary.BigEndian.PutUint16(size, uint16(len(content)))
		data = append(header, size...)
	} else if len(content) >= 65536 {
		header[1] = 127

		var size []byte
		binary.BigEndian.PutUint64(size, uint64(len(content)))
		data = append(header, size...)
	}

	// return byte array, length and error
	return append(data, content...)

}

// Simplified Read function
func (ws *Ws) Read() ([]byte, error) {
	var byteArray []byte = make([]byte, 4096)
	length, _ := ws.Buffer.Read(byteArray)
	frame, err := ws.readFrame(byteArray[:length])
	return frame, err
}

// Simplified Write function
func (ws *Ws) Write(byteArray []byte) (int, error) {
	frame := ws.WriteFrame(byteArray)
	nn, err := ws.Buffer.Write(frame)
	onError(err)
	err = ws.Buffer.Flush()

	// return amount of bytes written and error
	return nn, err
}

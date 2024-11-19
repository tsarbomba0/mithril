package wsclient

// Import containing a function to create a WebSocket client.

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"mithril/util"
	"net"
	"strings"
)

// Type describing a (client) WebSocket connection
type ClientWs struct {
	Conn    net.Conn
	Buffer  *bufio.ReadWriter
	Headers map[string]string
}

// Closes the connection
func (ws *ClientWs) Close(statusCode uint16, reason string) error {
	if len(reason)+2 > 125 {
		err := errors.New("control frame length exceeded 125")
		return err
	}

	var data = new(bytes.Buffer)

	binary.Write(data, binary.BigEndian, statusCode)
	binary.Write(data, binary.BigEndian, []byte(reason))
	_, err := ws.Write(data.Bytes(), 136)
	ws.Conn.Close()
	return err
}

// sends a Pong.
//
// Returns a error in case of one happening.
func (ws *ClientWs) Pong(pongMessage string) error {
	ws.Buffer.Write([]byte{138})
	_, err := ws.Buffer.Write([]byte(pongMessage))
	ws.Buffer.Flush()
	return err
}

// sends a Ping.
//
// Returns a error in case of one happening.
func (ws *ClientWs) Ping(pongMessage string) error {
	ws.Buffer.Write([]byte{137})
	_, err := ws.Buffer.Write([]byte(pongMessage))
	ws.Buffer.Flush()
	return err
}

// Creates a payload frame.
//
// In case of an error it returns a empty byte array containing 1 zero byte and an error.
func (ws *ClientWs) createFrame(data []byte, flags byte) ([]byte, error) {
	var bytes = new(bytes.Buffer)
	// full payload length in bytes as variable
	var length []byte
	var payloadLength int = len(data)
	bytes.WriteByte(flags)
	// Random mask
	var mask []byte = []byte{byte(rand.Uint()), byte(rand.Uint()), byte(rand.Uint()), byte(rand.Uint())}

	if len(data) < 65535 {
		// Writes length
		bytes.WriteByte(byte(128 + payloadLength))
		// Writes the mask to the buffer
		bytes.Write(mask)
		// Writes the masked data to the buffer
		for i := 0; i <= payloadLength-1; i++ {
			err := bytes.WriteByte(data[i] ^ mask[i%4])
			if err != nil {
				return []byte{0}, err
			}
		}
	} else if payloadLength >= 65535 && payloadLength < 65536 {
		// Writes length
		bytes.WriteByte(254)
		// Writes the mask to the buffer
		bytes.Write(mask)
		// Writes the extended payload length
		binary.LittleEndian.PutUint16(length, uint16(payloadLength))
		bytes.Write(length)
		// Writes the masked data to the buffer
		for i := 0; i <= payloadLength-1; i++ {
			err := bytes.WriteByte(data[i] ^ mask[i%4])
			if err != nil {
				return []byte{0}, err
			}
		}

	} else {
		// Writes length
		bytes.WriteByte(255)
		// Writes the mask to the buffer
		bytes.Write(mask)
		// Writes the extended payload length
		binary.LittleEndian.PutUint64(length, uint64(payloadLength))
		bytes.Write(length)
		// Writes the masked data to the buffer
		for i := 0; i <= payloadLength-1; i++ {
			err := bytes.WriteByte(data[i] ^ mask[i%4])
			if err != nil {
				return []byte{0}, err
			}
		}
	}

	return bytes.Bytes(), nil
}

// Writes to the connection.
//
// Returns the amount of bytes written and error
func (ws *ClientWs) Write(data []byte, flags byte) (int, error) {
	frame, err := ws.createFrame(data, flags)
	if err != nil {
		return 0, err
	}
	n, err := ws.Buffer.Write(frame)
	ws.Buffer.Flush()
	return n, err
}

func (ws *ClientWs) decodeFrame(data []byte) ([]byte, string, error) {
	if data[1]&128 != 0 {
		util.OnError(errors.New("masked frame received"))
	}
	frameType, fin, err := util.Validate(data, false)
	if err != nil {
		return []byte{0}, "unknown", err
	}

	// implement continuation frames
	if fin == 0 {
		fmt.Println("Continuation frame!!!!!!!")
	}
	switch data[1] & 127 {
	case 127:
		return data[10:], frameType, err
	case 126:
		return data[4:], frameType, err
	default:
		return data[2:], frameType, err
	}

}

// Reads from the connection.
//
// Returns the amount of bytes written and error
func (ws *ClientWs) Read() ([]byte, error) {
	var data []byte = make([]byte, 4096)
	length, err := ws.Buffer.Read(data)
	if err != nil {
		return []byte{0}, err
	}
	decodedFrame, frameType, err := ws.decodeFrame(data[:length])
	if err != nil {
		return []byte{0}, err
	}
	switch frameType {
	case "binary":
		fallthrough
	case "text":
		// switch statement
		return decodedFrame, err
	case "ping":
		ws.Pong("Pong!")
		return decodedFrame, err
	case "pong":
		fmt.Println("Received a pong frame!")
		return decodedFrame, err
	case "close":
		ws.Close(1000, "Closing!")
	}
	ws.Buffer.Flush()
	return decodedFrame, err
}

// Generate Secure WebSocket key.
func generateWebSocketKey() string {
	nonce := new(bytes.Buffer)
	for i := 0; i <= 16; i++ {
		nonce.WriteByte(byte(rand.UintN(255)))
	}
	return base64.StdEncoding.EncodeToString(nonce.Bytes())
}

// Create a HTTP handshake to the server.
func serverHandshake(key string) []byte {
	var request strings.Builder
	request.WriteString("GET /ws HTTP/1.1\r\n")
	request.WriteString("Host: test.com:2000\r\n")
	request.WriteString("Upgrade: websocket\r\n")
	request.WriteString("Sec-WebSocket-Key: " + key + "\r\n")
	request.WriteString("Sec-WebSocket-Version: 13\r\n\r\n")

	return []byte(request.String())
}

// Validate Secure WebSocket Accept key from server.
func validateWebsocketAccept(accept string, key string) bool {
	h := sha1.New()
	h.Write([]byte(key))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil)) == accept
}

// Create a connection to a websocket.
func ConnectWebSocket(address string, port string, handler func(ws *ClientWs)) {
	connection, err := net.Dial("tcp", address+":"+port)
	util.OnError(err)
	defer connection.Close()
	var connectionEstablished bool = false

	// Buffer for the connection
	buffer := bufio.NewReadWriter(bufio.NewReader(connection), bufio.NewWriter(connection))

	// Scanner
	scanner := bufio.NewScanner(connection)

	// Sec-WebSocket-Key
	websocketKey := generateWebSocketKey()

	// Handshake with server
	handshake := serverHandshake(websocketKey)
	connection.Write(handshake)

	// Map of obtained headers
	var headers map[string]string = make(map[string]string)

	// If connection was established, hands over control to the handler
	if connectionEstablished {
		handler(&ClientWs{Conn: connection, Buffer: buffer, Headers: headers})
	} else {
		// Scan first line
		ok := scanner.Scan()
		if !ok {
			panic(io.EOF)
		}
		output := scanner.Text()
		if string(output) == "HTTP/1.1 101 Switching Protocols" {
			for {
				ok := scanner.Scan()
				if !ok {
					log.Println("EOF from server. Closing connection.")
					connection.Close()
					return
				}

				output := scanner.Text()
				if output != "" {
					keyValue := strings.Split(output, ":")
					headers[keyValue[0]] = strings.ReplaceAll(keyValue[1], " ", "")
				} else {
					break
				}
			}
			log.Println("Successfully obtained the headers from the HTTP reply")

		} else {
			log.Println("Invalid HTTP Reply: " + string(output) + ". Closing connection.")
			connection.Close()
		}

		if validateWebsocketAccept(headers["Sec-WebSocket-Accept"], websocketKey) {
			log.Println("Sec-WebSocket-Accept field is valid. Handing over control to the handler function.")
			connectionEstablished = true
			handler(&ClientWs{Conn: connection, Buffer: buffer, Headers: headers})
		} else {
			log.Println("Invalid Sec-WebSocket-Accept. Closed connection with WebSocket")
			connection.Close()
		}
	}
}

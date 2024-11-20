package websocket

// Import containing the WebSocket struct and it's associated methods
import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"mithril/util"
	"net"
	"regexp"
	"strings"
)

// WebSocket type
type Ws struct {
	Conn     net.Conn
	Buffer   *bufio.ReadWriter
	PingSent bool
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
		// if length of the split string is equal to 1 (empty slice) AND the next slice is not empty, return a error
		if len(splitString) == 1 && len(re.Split(headers[i+1], -1)) != 1 {
			util.OnError(fmt.Errorf("unexpected separation of headers! (line %d)", i))
			// else assign the value to the map
		} else if len(splitString) != 1 {
			key, value := splitString[0], splitString[1]
			settings[key] = value
		}
	}
	return settings
}

// Function to determine the type of the request and the route
func (ws Ws) DetermineRequest(byteArray []byte) (string, string, error) {
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
	response.WriteString("HTTP/1.1 " + errorCode + " " + util.HttpErrorCodes[errorCode] + "\r\n")
	response.WriteString("Content-Type: text/plain\r\n")
	response.WriteString("Content-Language: en\r\n\r\n")
	response.WriteString(reason + "\r\n\r\n")
	ws.Conn.Write([]byte(response.String()))
}

// Reads a frame and returns it as a string.
//
// Returns a bytearray and error (error can be nil)
func (ws *Ws) ReadFrame(frame []byte) ([]byte, error, bool) {
	// decoded payload
	var decodedPayload bytes.Buffer

	// Validating
	frameType, _, err := util.Validate(frame, true)

	if err != nil {
		return decodedPayload.Bytes(), err, false
	}

	// variable for determining if it's a close frame
	var isClose bool = false
	// mask variable
	var mask []byte
	// payload variable
	var payload []byte

	switch frameType {
	case "binary":
		fallthrough
	case "text":
		// switch statement
		switch frame[1] & 127 {
		case 127:
			mask = frame[11:15]
			payload = frame[15:]

			for i := 0; i != len(payload); i++ {
				decodedPayload.WriteByte(payload[i] ^ mask[i%4])
			}
		case 126:
			mask = frame[4:8]
			payload = frame[8:]

			for i := 0; i != len(payload); i++ {
				decodedPayload.WriteByte(payload[i] ^ mask[i%4])
			}
		default:
			// mask
			mask = frame[2:6]
			// actual payload data
			payload = frame[6:]

			// performing a XOR on payload byte with mask byte
			for i := 0; i != len(payload); i++ {
				decodedPayload.WriteByte(payload[i] ^ mask[i%4])
			}
		}
	case "ping":
		ws.Pong()

	case "pong":
		if ws.PingSent {
			log.Println("Received a pong frame.")
			ws.PingSent = false
		} else {
			log.Println("Received a pong frame without a earlier ping frame.")
		}

	case "close":
		isClose = true
	}
	// return string and error
	return decodedPayload.Bytes(), err, isClose

}

// Create a frame to be sent to client.
//
// Returns a byte array.
func (ws *Ws) createFrame(content []byte, flags byte) []byte {
	var header []byte = make([]byte, 2)
	header[0] = flags
	// data variable
	var data []byte

	// if length is less than 125, just attach it as a integer to the data array
	if len(content) <= 125 {
		header[1] = byte(len(content))
		data = header

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

// Simplified Read function.
func (ws *Ws) Read() ([]byte, error, bool) {
	var byteArray []byte = make([]byte, 4096)
	length, _ := ws.Buffer.Read(byteArray)
	frame, err, isClose := ws.ReadFrame(byteArray[:length])

	// return decoded frame and error
	return frame, err, isClose
}

// Simplified Write function.
func (ws *Ws) Write(byteArray []byte) (int, error) {
	frame := ws.createFrame(byteArray, 130)
	nn, _ := ws.Buffer.Write(frame)
	err := ws.Buffer.Flush()

	// return amount of bytes written and error
	return nn, err
}

// Write function that allows sending your own flags.
func (ws *Ws) SpecialWrite(byteArray []byte, flags byte) (int, error) {
	frame := ws.createFrame(byteArray, flags)
	nn, err := ws.Buffer.Write(frame)
	ws.Buffer.Flush()

	// return amount of bytes written and error
	return nn, err
}

// Sends a close frame with status code and a reason.
//
// Returns a error (can be nil)
func (ws *Ws) Close(statusCode uint16, reason string) error {
	if len(reason)+2 > 125 {
		err := errors.New("control frame length exceeded 125")
		return err
	}

	var data = new(bytes.Buffer)

	binary.Write(data, binary.BigEndian, statusCode)
	binary.Write(data, binary.BigEndian, []byte(reason))
	_, err := ws.SpecialWrite(data.Bytes(), 136)
	ws.Conn.Close()
	log.Println("Closed connection!")
	return err
}

// Sends a ping
//
// Returns a error (can be nil)
func (ws *Ws) Ping() error {
	ws.Buffer.Write([]byte{137})
	return ws.Buffer.Flush()
}

// Sends a pong
//
// Returns a error (can be nil)
func (ws *Ws) Pong() error {
	ws.Buffer.Write([]byte{138})
	return ws.Buffer.Flush()
}

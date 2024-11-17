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
	Conn               net.Conn
	Status             uint8
	Buffer             *bufio.ReadWriter
	ContinuationBuffer *bufio.ReadWriter
}

// Error handler
func onError(err error) {
	if err != nil {
		panic(err)
	}
}

// Creates a WebSocket
func CreateWebSocket(addr string, port string, handler func(websocket *Ws) error, routeString string) {
	listener, err := net.Listen("tcp", addr+":"+port)
	onError(err)
	defer listener.Close()

	for {
		// create listener
		connection, err := listener.Accept()
		onError(err)

		// Buffer
		buffer := bufio.NewReadWriter(bufio.NewReader(connection), bufio.NewWriter(connection))
		continuationBuffer := bufio.NewReadWriter(bufio.NewReader(connection), bufio.NewWriter(connection))
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

					//testBytes := make([]byte, 16)
					for {
						err := handler(ws)
						if err != nil {
							fmt.Println("error: ", err)
							ws.Conn.Close()
							break
						}

					}

				}

			} else {
				// this to be replaced with proper http response
				ws.SendHTTPError("400", "Invalid route! ("+route+")")
				ws.Conn.Close()
			}
		}(&Ws{Conn: connection, Buffer: buffer, ContinuationBuffer: continuationBuffer})

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

// Validates the received frame
func validate (frame []byte) error {
	var err error
	if len(frame) == 0 {
		err = errors.New("empty array of bytes")
		return err
	}

	controlFrame := false
	fragmented := false

	var fin byte = frame[0] & 128
	var opcode byte = frame[0] & 3
	var length byte = frame[1] & 127
	var maskBit byte = frame[1] & 128

	if fin == 0 {
		fragmented = true
	}

	if opcode >= 8 || opcode <= 10 {
		controlFrame = true
	} 
		
	
	// Checks if a control frame is fragmented
	if fragmented && controlFrame {
		return errors.New("fragmented control frame")
	// Checks if a control frame has a larger payload than 125 bytes
	} else if controlFrame && length > 125 {
		return errors.New("length of control frame payload larger than 125 bytes")
	// checks for a unmasked frame
	} else if maskBit == 0 && !controlFrame {
		return errors.New("unmasked frame")
	}

	return err
}

// Reads a frame and returns it as a string.
// Returns a string and error
func (ws *Ws) readFrame(frame []byte) ([]byte, error) {
	// error variable
	var err error = nil
	// decoded payload
	var decodedPayload bytes.Buffer

	// Validating
	err = validate(frame)
	if err != nil {
		return decodedPayload.Bytes(), err
	}

	// mask variable
	var mask []byte
	// payload variable
	var payload []byte
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

	// return string and error
	return decodedPayload.Bytes(), err
	
}

// create a frame
func (ws *Ws) WriteFrame(content []byte, flags byte) []byte {
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

// Simplified Read function
func (ws *Ws) Read() ([]byte, error) {
	var byteArray []byte = make([]byte, 4096)
	length, _ := ws.Buffer.Read(byteArray)
	frame, err := ws.readFrame(byteArray[:length])

	// return decoded frame and error
	return frame, err
}

// Simplified Write function
func (ws *Ws) Write(byteArray []byte) (int, error) {
	frame := ws.WriteFrame(byteArray, 130)
	nn, err := ws.Buffer.Write(frame)
	fmt.Println(err)
	err = ws.Buffer.Flush()

	// return amount of bytes written and error
	return nn, err
}

// Write function that allows sending your own flags
func (ws *Ws) SpecialWrite(byteArray []byte, flags byte) (int, error) {
	frame := ws.WriteFrame(byteArray, flags)
	nn, err := ws.Buffer.Write(frame)
	fmt.Println(err)
	err = ws.Buffer.Flush()

	// return amount of bytes written and error
	return nn, err
}

// Sends a close frame with status code and a reason
func (ws *Ws) Close(statusCode uint16, reason string) error {
	if len(reason)+2 > 125 {
		err := errors.New("control frame length exceeded 125")
		return err
	}

	var data = new(bytes.Buffer)

	binary.Write(data, binary.BigEndian, statusCode)
	binary.Write(data, binary.BigEndian, reason)
	fmt.Println(data.Bytes())
	_, err := ws.SpecialWrite(data.Bytes(), 136)
	return err
}
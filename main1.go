package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"
)

// Creates the hash used to accept a WebSocket connection.
func acceptHash(k string) string {
	h := sha1.New()
	h.Write([]byte(k))
	h.Write([]byte("258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Returns a handshake to be sent via HTTP.
func serverHandshake(secretKey string) string {
	var req strings.Builder
	req.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %s", acceptHash(secretKey)+"\r\n\r\n"))

	return req.String()
}

// Gets HTTP Headers
func getHTTPHeaders(dataBytes []byte) map[string]string {
	// Map to hold the headers
	settings := make(map[string]string)

	// Actual slice of headers from the request
	headers := strings.Split(string(dataBytes), "\r\n")

	// Iterating and attaching to the map (excluding the first entry, it's the HTTP version string)
	for i := 1; i < len(headers)-1; i++ {
		// splitting the string from the header
		re := regexp.MustCompile(":\\s")
		splitString := re.Split(headers[i], -1)

		// if length of the split string is equal to 1 (empty slice) AND the next slice is not empty, do a error
		fmt.Println(splitString, len(splitString))
		if len(splitString) == 1 && len(re.Split(headers[i+1], -1)) != 1 {
			onError(errors.New(fmt.Sprintf("Unexpected separation of headers! (line %d)", i)))

			// else assign the value to the map
		} else if len(splitString) != 1 {
			key, value := splitString[0], splitString[1]
			settings[key] = value
		}
	}

	return settings
}

// Function to determine the type of the request and the route
func determineRequest(byteArray []byte) (string, string, error) {
	buf := make([]byte, 128)
	reader := bytes.NewReader(byteArray)
	index, err := reader.Read(buf)

	re, _ := regexp.Compile("/[A-Za-z]+")
	route := re.FindString(string(buf[:index]))

	re, _ = regexp.Compile("POST|GET|PATCH|DELETE|PUT")
	method := re.FindString(string(buf[:index]))

	return method, route, err

}

// Error handler
func onError(err error) {
	if err != nil {
		panic(err)
	}
}

func Connection(conn net.Conn) {
	bytes := make([]byte, 256)
	for {
		length, err := conn.Read(bytes)
		if err == io.EOF {
			err = nil
		} else {
			onError(err)
		}

		//method, route, err := determineRequest(bytes[:length])
		//onError(err)

		headers := getHTTPHeaders(bytes[:length])
		fmt.Print(headers["Sec-WebSocket-Key"])

		conn.Write([]byte(serverHandshake(headers["Sec-WebSocket-Key"])))

	}
	conn.Close()
}

func main() {
	var serverAddress string
	var serverPort string

	// Listening on some address
	if len(os.Args) >= 2 {
		serverAddress = os.Args[1]
	} else {
		fmt.Println("No address provided, listening on all.")
		serverAddress = ""
	}
	if len(os.Args) >= 3 {
		serverPort = os.Args[2]
	} else {
		fmt.Println("No port provided, defaulting to 80")
		serverPort = "80"
	}

	listener, err := net.Listen("tcp", serverAddress+":"+serverPort)
	onError(err)
	defer listener.Close()

	for {
		connection, err := listener.Accept()
		onError(err)

		// go routine
		go Connection(connection)

	}
}

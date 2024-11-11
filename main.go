package main

import (
	"fmt"
	"net"
	"io"
	"bytes"
	"strings"
	"regexp"
	"errors"
	"crypto/sha1"
	"encoding/base64"
)

// sha1sum of incoming string encoded to base64
func sha1base64(str string) string {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

// (Shit) function to generate a handshake response
func serverHandshake(secretKey string) string {
	var outputSecretKey strings.Builder
	outputSecretKey.WriteString(secretKey)
	outputSecretKey.WriteString("258EAFA5-E914-47DA-95CA-C5AB0DC85B11")

	var req strings.Builder
	req.WriteString("HTTP/1.1 101 Switching Protocols\r\n")
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	req.WriteString(fmt.Sprintf("Sec-WebSocket-Accept: %s\r\n", sha1base64(outputSecretKey.String())))

	return req.String()
}

// Handling errors
func onError(err error) {
	if err != nil {
		panic(err)
	}
}

// Function to determine the type of the request and the route
func determineRequest(byteArray []byte) ([]string, error) {
	buf := make([]byte, 128)
	reader := bytes.NewReader(byteArray)
	index, err := reader.Read(buf)

	re, _:= regexp.Compile("/[A-Za-z]+")
	route := re.FindString(string(buf[:index]))

	re, _ = regexp.Compile("POST|GET|PATCH|DELETE|PUT")
	method := re.FindString(string(buf[:index]))

	return []string{method, route}, err

}

func GETRequest(req string, route string, conn net.Conn){
	// Slice of headers separated by CRLF
	headers := strings.Split(req, "\r\n")
	fmt.Println(headers)

	// At first empty map, will contain the headers in accessible form
	settings := make(map[string]string)
	switch route {
	case "/ws":
		for i := 1; i<len(headers)-1; i++ {
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
		io.WriteString(conn, serverHandshake(settings["Sec-WebSocket-Key"]))

	}
	// Print out map
	fmt.Println(settings)
}

// Function to handle the connection
func handle(conn net.Conn) {
		// Create buffer of 256 bytes
		dataBytes := make([]byte, 256)
		conn.Read(dataBytes)

		if wsEstablished {
			// Websocket code
			// blah blah
		} else {
			// Method and route for request
			specifics, err := determineRequest(dataBytes)
			onError(err)

			// Switch cases for methods
			switch specifics[0]{
			case "GET":
				GETRequest(string(dataBytes), specifics[1], conn)
			case "POST":
				fmt.Println("POST Request!")
			case "PATCH":
				fmt.Println("PATCH Request!")
			case "PUT":
				fmt.Println("PUT Request!")
			case "DELETE":
				fmt.Println("DELETE Request!")
			}
		}
	
	conn.Close()
}

func main() {

	// Listening
	listener, err := net.Listen("tcp", ":80")
	onError(err)
	defer listener.Close()
	
	// loop 
	for {
		connection, err := listener.Accept()
		onError(err)

		// go routine
		go handle(connection)
	}
}
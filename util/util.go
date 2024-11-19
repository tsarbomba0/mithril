package util

// Import containing utility variables and methods.

import (
	"errors"
)

// Contains WebSocket close codes.
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

// Contains HTTP error codes.
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

// Validates the received frame.
//
// Returns a error (can be nil).
func Validate(frame []byte, mustMask bool) (string, byte, error) {
	var err error

	if len(frame) == 0 {
		err = errors.New("empty array of bytes")
		return "empty", byte(0), err
	}

	var controlFrame bool = false
	var frameType string

	var fin byte = frame[0] & 128
	var opcode byte = frame[0] & 15
	var length byte = frame[1] & 127
	var maskBit byte = frame[1] & 128

	switch opcode {
	case 0:
		frameType = "continuation"
	case 1:
		frameType = "text"
	case 2:
		frameType = "binary"
	case 8:
		frameType = "close"
		controlFrame = true
	case 9:
		frameType = "ping"
		controlFrame = true
	case 10:
		frameType = "pong"
		controlFrame = true
	default:
		return "unknown", fin, errors.New("used a reserved opcode")
	}

	// Checks if a control frame is fragmented
	if fin == 0 && controlFrame {
		return frameType, fin, errors.New("fragmented control frame")
		// Checks if a control frame has a larger payload than 125 bytes
	} else if controlFrame && length > 125 {
		return frameType, fin, errors.New("length of control frame payload larger than 125 bytes")
		// checks for a unmasked frame
	} else if maskBit == 0 && !controlFrame && mustMask {
		return frameType, fin, errors.New("unmasked frame")
	} else {
		return frameType, fin, nil
	}

}

// Error handler.
func OnError(err error) {
	if err != nil {
		panic(err)
	}
}

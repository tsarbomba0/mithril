package tls

import (
	"bytes"
	"encoding/binary"
	"math/rand/v2"
)

// TERRIBLE ATTEMPT AT TLS
// THIS IS ALL FOR TESTING

const (
	clientHello        = byte(1)
	serverHello        = byte(2)
	certificateMessage = byte(0x0B)
	serverKeyExchange  = byte(0x0C)
	serverHelloDone    = byte(0x0E)
	changeCipherSpec   = byte(0x14)
)

// TLS Record header
type TLSRecord struct {
	contentType uint
	version     uint16
	length      uint16
}

func (record *TLSRecord) Bytes() []byte {
	var a []byte = []byte{byte(record.contentType)}
	versionBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(versionBytes, record.version)
	lengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthBytes, record.length)
	a = append(a, versionBytes...)
	return append(a, lengthBytes...)
}

// Extensions
func extensionsMessage(domain string) []byte {
	// Domain
	extensionDomain := new(bytes.Buffer)

	domainBytes := []byte(domain)
	// extension type 2B
	extensionDomain.Write([]byte{0, 0})
	// length 2B
	extensionDomain.Write([]byte{0, byte(len(domainBytes) + 5)})
	// server name list 2B server name type 1B
	extensionDomain.Write([]byte{0, byte(len(domainBytes) + 3), 0})
	// server name length 2B
	extensionDomain.Write([]byte{0, byte(len(domainBytes))})
	// server domain
	extensionDomain.Write(domainBytes)

	// Encryption
	extensionSignatureAlgorithms := new(bytes.Buffer)
	// 16B Algorithms
	algorithms := []byte{
		6, 1, // SHA512 + RSA
		6, 3, // SHA512 + ECDSA
		5, 1, // SHA384 + RSA
		5, 3, // SHA384 + ECDSA
		4, 1, // SHA256 + RSA
		4, 3, // SHA256 + ECDSA
		2, 1, // SHA1 + RSA
		2, 3, // SHA1 + ECDSA
	}
	// Length of algorithm array
	algorithmsLength := len(algorithms)
	// 0x000D => extension type signature_algorithms
	// byte(algorithmsLength+2) => length  byte(algorithmsLength) => length of algorithm array
	extensionSignatureAlgorithms.Write([]byte{
		0x00, 0x0D,
		byte(algorithmsLength + 2), byte(algorithmsLength),
	})
	// 16B algorithm array
	extensionSignatureAlgorithms.Write(algorithms)

	// Renegotiations info
	renegotiationInfo := new(bytes.Buffer)
	renegotiationInfo.Write([]byte{
		0xFF, 0x01,
		0x00, 0x01,
		0x00,
	})

	return append(append(renegotiationInfo.Bytes(), extensionSignatureAlgorithms.Bytes()...), extensionDomain.Bytes()...)
}

// Client Hello message
func ClientHello(record *TLSRecord) []byte {
	var message bytes.Buffer
	// Record header
	message.Write(record.Bytes())

	// Client hello ID 1B
	message.WriteByte(byte(0x01))

	// Append size 512 bytes 3B
	message.Write([]byte{0, 2, 0})

	// TLS version 2B
	message.Write([]byte{3, 3})

	// Random 32-bytes 32B
	var randomBytes []byte = make([]byte, 32)
	for i := 1; i <= len(randomBytes); i++ {
		randomBytes[i] = byte(rand.UintN(255))
	}
	message.Write(randomBytes)

	// Null session id 1B
	message.WriteByte(0)

	// Cipher suit length 2B
	message.Write([]byte{0, 2})

	// Cipher suit 2B TLS_DHE_RSA_WITH_AES_128_CBC_SHA
	message.Write([]byte{0, 33})

	// Compression methods 1B
	message.WriteByte(1)

	// Compression none 1B
	message.WriteByte(0)

	// Extensions
	extensionBytes := extensionsMessage("www.test.org")
	extensionsLength := make([]byte, 2)
	binary.BigEndian.PutUint16(extensionsLength, uint16(len(extensionBytes)))
	message.Write(extensionsLength)
	message.Write(extensionBytes)

	// Padding
	messageBytes := message.Bytes()
	requiredPadding := 512 - len(messageBytes) + 5
	padding := make([]byte, requiredPadding)

	return append(messageBytes, padding...)
}

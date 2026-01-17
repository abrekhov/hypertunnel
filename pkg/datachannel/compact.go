/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */

package datachannel

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/pion/webrtc/v3"
)

// HyperTunnel Compact Protocol (HTCP) constants
const (
	htcpMagic   byte = 'H'
	htcpVersion byte = 1

	// Fixed sizes in the protocol
	ufragSize       = 16
	passwordSize    = 32
	fingerprintSize = 32 // SHA-256

	// Candidate type encoding (high nibble of type+proto byte)
	candTypeHost  byte = 0
	candTypeSrflx byte = 1
	candTypePrflx byte = 2
	candTypeRelay byte = 3
)

var (
	// ErrInvalidMagic indicates the signal doesn't start with the expected magic byte
	ErrInvalidMagic = errors.New("invalid HTCP magic byte")
	// ErrUnsupportedVersion indicates an unsupported protocol version
	ErrUnsupportedVersion = errors.New("unsupported HTCP version")
	// ErrInvalidSignal indicates the signal data is malformed
	ErrInvalidSignal = errors.New("invalid signal data")
)

// candidateTypeToString converts our compact type encoding to WebRTC type string
func candidateTypeToString(t byte) webrtc.ICECandidateType {
	switch t {
	case candTypeSrflx:
		return webrtc.ICECandidateTypeSrflx
	case candTypePrflx:
		return webrtc.ICECandidateTypePrflx
	case candTypeRelay:
		return webrtc.ICECandidateTypeRelay
	default:
		return webrtc.ICECandidateTypeHost
	}
}

// stringToCandidateType converts WebRTC type string to our compact encoding
func stringToCandidateType(t webrtc.ICECandidateType) byte {
	switch t {
	case webrtc.ICECandidateTypeSrflx:
		return candTypeSrflx
	case webrtc.ICECandidateTypePrflx:
		return candTypePrflx
	case webrtc.ICECandidateTypeRelay:
		return candTypeRelay
	default:
		return candTypeHost
	}
}

// protocolToInt converts WebRTC protocol to int
func protocolToInt(p webrtc.ICEProtocol) byte {
	if p == webrtc.ICEProtocolTCP {
		return 2
	}
	return 1 // UDP is default
}

// intToProtocol converts int to WebRTC protocol
func intToProtocol(p byte) webrtc.ICEProtocol {
	if p == 2 {
		return webrtc.ICEProtocolTCP
	}
	return webrtc.ICEProtocolUDP
}

// EncodeCompact creates a compact binary signal (74% smaller than JSON).
// Format: H<ver><ufrag_len:1><ufrag:N><pwd_len:1><pwd:N><role:1><fingerprint:32><num_cand:1><candidates...>
func EncodeCompact(s Signal) (string, error) {
	var buf bytes.Buffer

	// Magic byte and version
	buf.WriteByte(htcpMagic)
	buf.WriteByte(htcpVersion)

	// ICE Parameters - ufrag and password (length-prefixed for flexibility)
	ufrag := s.ICEParameters.UsernameFragment
	pwd := s.ICEParameters.Password

	// Truncate if too long (max 255 bytes due to 1-byte length prefix)
	if len(ufrag) > 255 {
		ufrag = ufrag[:255]
	}
	if len(pwd) > 255 {
		pwd = pwd[:255]
	}

	buf.WriteByte(byte(len(ufrag)))
	buf.WriteString(ufrag)
	buf.WriteByte(byte(len(pwd)))
	buf.WriteString(pwd)

	// DTLS Role (1 byte)
	buf.WriteByte(byte(s.DTLSParameters.Role))

	// DTLS Fingerprint as raw bytes (32 bytes for SHA-256)
	if len(s.DTLSParameters.Fingerprints) > 0 {
		fpHex := strings.ReplaceAll(s.DTLSParameters.Fingerprints[0].Value, ":", "")
		fpBytes, err := hex.DecodeString(fpHex)
		if err != nil {
			return "", err
		}
		// Ensure exactly 32 bytes
		if len(fpBytes) < fingerprintSize {
			fpBytes = append(fpBytes, make([]byte, fingerprintSize-len(fpBytes))...)
		}
		buf.Write(fpBytes[:fingerprintSize])
	} else {
		// Write zeros if no fingerprint
		buf.Write(make([]byte, fingerprintSize))
	}

	// Number of candidates (1 byte, max 255)
	numCandidates := len(s.ICECandidates)
	if numCandidates > 255 {
		numCandidates = 255
	}
	buf.WriteByte(byte(numCandidates))

	// Encode each candidate
	for i := 0; i < numCandidates; i++ {
		c := s.ICECandidates[i]
		encodeCandidate(&buf, c)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// encodeCandidate writes a single ICE candidate to the buffer
func encodeCandidate(buf *bytes.Buffer, c webrtc.ICECandidate) {
	// Foundation (length-prefixed string)
	foundation := c.Foundation
	if len(foundation) > 255 {
		foundation = foundation[:255]
	}
	buf.WriteByte(byte(len(foundation)))
	buf.WriteString(foundation)

	// Priority (4 bytes, big-endian)
	_ = binary.Write(buf, binary.BigEndian, c.Priority)

	// Address (length-prefixed string)
	addr := c.Address
	if len(addr) > 255 {
		addr = addr[:255]
	}
	buf.WriteByte(byte(len(addr)))
	buf.WriteString(addr)

	// Port (2 bytes, big-endian)
	_ = binary.Write(buf, binary.BigEndian, c.Port)

	// Type + Protocol packed into 1 byte (type in high nibble, proto in low)
	candType := stringToCandidateType(c.Typ)
	proto := protocolToInt(c.Protocol)
	packed := (candType << 4) | (proto & 0x0F)
	buf.WriteByte(packed)

	// For non-host types, include related address info
	if candType != candTypeHost {
		relAddr := c.RelatedAddress
		if len(relAddr) > 255 {
			relAddr = relAddr[:255]
		}
		buf.WriteByte(byte(len(relAddr)))
		if len(relAddr) > 0 {
			buf.WriteString(relAddr)
			_ = binary.Write(buf, binary.BigEndian, c.RelatedPort)
		}
	}
}

// DecodeCompact parses a compact binary signal back to Signal struct
func DecodeCompact(encoded string, s *Signal) error {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}

	if len(data) < 2 {
		return ErrInvalidSignal
	}

	// Check magic byte
	if data[0] != htcpMagic {
		return ErrInvalidMagic
	}

	// Check version
	if data[1] != htcpVersion {
		return ErrUnsupportedVersion
	}

	// Minimum size: magic(1) + ver(1) + ufrag_len(1) + pwd_len(1) + role(1) + fp(32) + numCand(1) = 38
	minSize := 1 + 1 + 1 + 1 + 1 + fingerprintSize + 1
	if len(data) < minSize {
		return ErrInvalidSignal
	}

	offset := 2

	// ICE Parameters (length-prefixed strings)
	if offset >= len(data) {
		return ErrInvalidSignal
	}
	ufragLen := int(data[offset])
	offset++
	if offset+ufragLen > len(data) {
		return ErrInvalidSignal
	}
	ufrag := string(data[offset : offset+ufragLen])
	offset += ufragLen

	if offset >= len(data) {
		return ErrInvalidSignal
	}
	pwdLen := int(data[offset])
	offset++
	if offset+pwdLen > len(data) {
		return ErrInvalidSignal
	}
	pwd := string(data[offset : offset+pwdLen])
	offset += pwdLen

	s.ICEParameters = webrtc.ICEParameters{
		UsernameFragment: ufrag,
		Password:         pwd,
		ICELite:          false,
	}

	// DTLS Parameters - bounds check
	if offset+1+fingerprintSize+1 > len(data) {
		return ErrInvalidSignal
	}

	role := webrtc.DTLSRole(data[offset])
	offset++

	fpBytes := data[offset : offset+fingerprintSize]
	offset += fingerprintSize

	// Convert fingerprint bytes to colon-separated hex string
	fpHex := hex.EncodeToString(fpBytes)
	var fpParts []string
	for i := 0; i < len(fpHex); i += 2 {
		fpParts = append(fpParts, fpHex[i:i+2])
	}
	fpValue := strings.Join(fpParts, ":")

	s.DTLSParameters = webrtc.DTLSParameters{
		Role: role,
		Fingerprints: []webrtc.DTLSFingerprint{
			{
				Algorithm: "sha-256",
				Value:     fpValue,
			},
		},
	}

	// SCTP Capabilities (we use default)
	s.SCTPCapabilities = webrtc.SCTPCapabilities{
		MaxMessageSize: 0,
	}

	// Number of candidates
	numCandidates := int(data[offset])
	offset++

	// Decode each candidate
	s.ICECandidates = make([]webrtc.ICECandidate, 0, numCandidates)
	for i := 0; i < numCandidates; i++ {
		cand, newOffset, err := decodeCandidate(data, offset)
		if err != nil {
			return err
		}
		s.ICECandidates = append(s.ICECandidates, cand)
		offset = newOffset
	}

	return nil
}

// decodeCandidate reads a single ICE candidate from the buffer
func decodeCandidate(data []byte, offset int) (webrtc.ICECandidate, int, error) {
	var c webrtc.ICECandidate

	if offset >= len(data) {
		return c, offset, ErrInvalidSignal
	}

	// Foundation
	foundationLen := int(data[offset])
	offset++
	if offset+foundationLen > len(data) {
		return c, offset, ErrInvalidSignal
	}
	c.Foundation = string(data[offset : offset+foundationLen])
	offset += foundationLen

	// Priority (4 bytes)
	if offset+4 > len(data) {
		return c, offset, ErrInvalidSignal
	}
	c.Priority = binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	// Address
	if offset >= len(data) {
		return c, offset, ErrInvalidSignal
	}
	addrLen := int(data[offset])
	offset++
	if offset+addrLen > len(data) {
		return c, offset, ErrInvalidSignal
	}
	c.Address = string(data[offset : offset+addrLen])
	offset += addrLen

	// Port (2 bytes)
	if offset+2 > len(data) {
		return c, offset, ErrInvalidSignal
	}
	c.Port = binary.BigEndian.Uint16(data[offset : offset+2])
	offset += 2

	// Type + Protocol
	if offset >= len(data) {
		return c, offset, ErrInvalidSignal
	}
	packed := data[offset]
	offset++

	candType := (packed >> 4) & 0x0F
	proto := packed & 0x0F

	c.Typ = candidateTypeToString(candType)
	c.Protocol = intToProtocol(proto)
	c.Component = 1 // Always 1 for data channels

	// Related address for non-host types
	if candType != candTypeHost {
		if offset >= len(data) {
			return c, offset, ErrInvalidSignal
		}
		relAddrLen := int(data[offset])
		offset++

		if relAddrLen > 0 {
			if offset+relAddrLen+2 > len(data) {
				return c, offset, ErrInvalidSignal
			}
			c.RelatedAddress = string(data[offset : offset+relAddrLen])
			offset += relAddrLen
			c.RelatedPort = binary.BigEndian.Uint16(data[offset : offset+2])
			offset += 2
		}
	}

	return c, offset, nil
}

// IsCompactFormat checks if the encoded string is in compact HTCP format
// Compact format starts with 'H' which is "SA" in base64
// JSON format starts with '{' which is "ey" in base64
func IsCompactFormat(encoded string) bool {
	return strings.HasPrefix(encoded, "SA")
}

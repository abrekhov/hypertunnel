/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */

package datachannel

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/pion/webrtc/v3"
)

// createCompactTestSignal creates a realistic test signal with multiple candidates
func createCompactTestSignal() Signal {
	return Signal{
		ICECandidates: []webrtc.ICECandidate{
			{
				Foundation: "3537766002",
				Priority:   2130706431,
				Address:    "192.168.1.100",
				Protocol:   webrtc.ICEProtocolUDP,
				Port:       31545,
				Typ:        webrtc.ICECandidateTypeHost,
				Component:  1,
			},
			{
				Foundation:     "842163049",
				Priority:       1694498815,
				Address:        "203.0.113.42",
				Protocol:       webrtc.ICEProtocolUDP,
				Port:           54321,
				Typ:            webrtc.ICECandidateTypeSrflx,
				Component:      1,
				RelatedAddress: "192.168.1.100",
				RelatedPort:    31545,
			},
			{
				Foundation:     "1677722412",
				Priority:       33562367,
				Address:        "198.51.100.5",
				Protocol:       webrtc.ICEProtocolUDP,
				Port:           3478,
				Typ:            webrtc.ICECandidateTypeRelay,
				Component:      1,
				RelatedAddress: "192.168.1.100",
				RelatedPort:    31545,
			},
		},
		ICEParameters: webrtc.ICEParameters{
			UsernameFragment: "GOXteffFpNfkHMrj",
			Password:         "lceNxPWPURZrbEPXWczKSrsRwIppKSZQ",
			ICELite:          false,
		},
		DTLSParameters: webrtc.DTLSParameters{
			Role: webrtc.DTLSRoleClient,
			Fingerprints: []webrtc.DTLSFingerprint{
				{
					Algorithm: "sha-256",
					Value:     "2f:a0:55:de:c2:70:55:aa:ef:6c:af:64:8e:68:90:03:0a:e2:cf:39:8d:a6:5d:ab:c9:fe:0d:b8:d6:aa:82:db",
				},
			},
		},
		SCTPCapabilities: webrtc.SCTPCapabilities{
			MaxMessageSize: 0,
		},
	}
}

func TestEncodeDecodeCompactRoundTrip(t *testing.T) {
	original := createCompactTestSignal()

	// Encode
	encoded, err := EncodeCompact(original)
	if err != nil {
		t.Fatalf("EncodeCompact failed: %v", err)
	}

	// Decode
	var decoded Signal
	err = DecodeCompact(encoded, &decoded)
	if err != nil {
		t.Fatalf("DecodeCompact failed: %v", err)
	}

	// Verify ICE Parameters
	if decoded.ICEParameters.UsernameFragment != original.ICEParameters.UsernameFragment {
		t.Errorf("UsernameFragment mismatch: got %q, want %q",
			decoded.ICEParameters.UsernameFragment, original.ICEParameters.UsernameFragment)
	}
	if decoded.ICEParameters.Password != original.ICEParameters.Password {
		t.Errorf("Password mismatch: got %q, want %q",
			decoded.ICEParameters.Password, original.ICEParameters.Password)
	}

	// Verify DTLS Parameters
	if decoded.DTLSParameters.Role != original.DTLSParameters.Role {
		t.Errorf("DTLS Role mismatch: got %v, want %v",
			decoded.DTLSParameters.Role, original.DTLSParameters.Role)
	}
	if len(decoded.DTLSParameters.Fingerprints) != 1 {
		t.Fatalf("Expected 1 fingerprint, got %d", len(decoded.DTLSParameters.Fingerprints))
	}
	if decoded.DTLSParameters.Fingerprints[0].Value != original.DTLSParameters.Fingerprints[0].Value {
		t.Errorf("Fingerprint mismatch: got %q, want %q",
			decoded.DTLSParameters.Fingerprints[0].Value, original.DTLSParameters.Fingerprints[0].Value)
	}

	// Verify candidates count
	if len(decoded.ICECandidates) != len(original.ICECandidates) {
		t.Fatalf("Candidate count mismatch: got %d, want %d",
			len(decoded.ICECandidates), len(original.ICECandidates))
	}

	// Verify each candidate
	for i, origCand := range original.ICECandidates {
		decCand := decoded.ICECandidates[i]

		if decCand.Foundation != origCand.Foundation {
			t.Errorf("Candidate %d Foundation mismatch: got %q, want %q", i, decCand.Foundation, origCand.Foundation)
		}
		if decCand.Priority != origCand.Priority {
			t.Errorf("Candidate %d Priority mismatch: got %d, want %d", i, decCand.Priority, origCand.Priority)
		}
		if decCand.Address != origCand.Address {
			t.Errorf("Candidate %d Address mismatch: got %q, want %q", i, decCand.Address, origCand.Address)
		}
		if decCand.Port != origCand.Port {
			t.Errorf("Candidate %d Port mismatch: got %d, want %d", i, decCand.Port, origCand.Port)
		}
		if decCand.Typ != origCand.Typ {
			t.Errorf("Candidate %d Type mismatch: got %v, want %v", i, decCand.Typ, origCand.Typ)
		}
		if decCand.Protocol != origCand.Protocol {
			t.Errorf("Candidate %d Protocol mismatch: got %v, want %v", i, decCand.Protocol, origCand.Protocol)
		}

		// Check related address for non-host types
		if origCand.Typ != webrtc.ICECandidateTypeHost {
			if decCand.RelatedAddress != origCand.RelatedAddress {
				t.Errorf("Candidate %d RelatedAddress mismatch: got %q, want %q", i, decCand.RelatedAddress, origCand.RelatedAddress)
			}
			if decCand.RelatedPort != origCand.RelatedPort {
				t.Errorf("Candidate %d RelatedPort mismatch: got %d, want %d", i, decCand.RelatedPort, origCand.RelatedPort)
			}
		}
	}
}

func TestCompactSizeReduction(t *testing.T) {
	signal := createCompactTestSignal()

	// JSON + Base64 (original format)
	jsonBytes, _ := json.Marshal(signal)
	jsonB64 := base64.StdEncoding.EncodeToString(jsonBytes)

	// Compact format
	compactB64, err := EncodeCompact(signal)
	if err != nil {
		t.Fatalf("EncodeCompact failed: %v", err)
	}

	reduction := float64(len(jsonB64)-len(compactB64)) / float64(len(jsonB64)) * 100

	t.Logf("JSON+Base64 size: %d chars", len(jsonB64))
	t.Logf("Compact size: %d chars", len(compactB64))
	t.Logf("Size reduction: %.1f%%", reduction)

	// Expect at least 60% reduction
	if reduction < 60 {
		t.Errorf("Expected at least 60%% size reduction, got %.1f%%", reduction)
	}
}

func TestIsCompactFormat(t *testing.T) {
	tests := []struct {
		name     string
		encoded  string
		expected bool
	}{
		{
			name:     "Compact format starts with SA",
			encoded:  "SAFHT1h0ZWZm...",
			expected: true,
		},
		{
			name:     "JSON format starts with ey",
			encoded:  "eyJpY2VDYW5kaWRhdGVzIjpb...",
			expected: false,
		},
		{
			name:     "Empty string",
			encoded:  "",
			expected: false,
		},
		{
			name:     "Random string",
			encoded:  "abcdefg",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCompactFormat(tt.encoded)
			if result != tt.expected {
				t.Errorf("IsCompactFormat(%q) = %v, want %v", tt.encoded, result, tt.expected)
			}
		})
	}
}

func TestEncodeSingleCandidate(t *testing.T) {
	signal := Signal{
		ICECandidates: []webrtc.ICECandidate{
			{
				Foundation: "1234567890",
				Priority:   2130706431,
				Address:    "10.0.0.1",
				Protocol:   webrtc.ICEProtocolUDP,
				Port:       12345,
				Typ:        webrtc.ICECandidateTypeHost,
				Component:  1,
			},
		},
		ICEParameters: webrtc.ICEParameters{
			UsernameFragment: "testufrag1234567",
			Password:         "testpassword12345678901234567890",
		},
		DTLSParameters: webrtc.DTLSParameters{
			Role: webrtc.DTLSRoleServer,
			Fingerprints: []webrtc.DTLSFingerprint{
				{
					Algorithm: "sha-256",
					Value:     "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99:aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99",
				},
			},
		},
	}

	encoded, err := EncodeCompact(signal)
	if err != nil {
		t.Fatalf("EncodeCompact failed: %v", err)
	}

	var decoded Signal
	err = DecodeCompact(encoded, &decoded)
	if err != nil {
		t.Fatalf("DecodeCompact failed: %v", err)
	}

	if len(decoded.ICECandidates) != 1 {
		t.Fatalf("Expected 1 candidate, got %d", len(decoded.ICECandidates))
	}

	if decoded.ICECandidates[0].Address != "10.0.0.1" {
		t.Errorf("Address mismatch: got %q, want %q", decoded.ICECandidates[0].Address, "10.0.0.1")
	}
}

func TestDecodeInvalidSignals(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		wantErr error
	}{
		{
			name:    "Not base64",
			encoded: "!!!invalid-base64!!!",
			wantErr: nil, // base64 decode error
		},
		{
			name:    "Wrong magic byte",
			encoded: base64.StdEncoding.EncodeToString([]byte("X\x01" + strings.Repeat("\x00", 82))),
			wantErr: ErrInvalidMagic,
		},
		{
			name:    "Unsupported version",
			encoded: base64.StdEncoding.EncodeToString([]byte("H\x99" + strings.Repeat("\x00", 82))),
			wantErr: ErrUnsupportedVersion,
		},
		{
			name:    "Too short",
			encoded: base64.StdEncoding.EncodeToString([]byte("H\x01")),
			wantErr: ErrInvalidSignal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s Signal
			err := DecodeCompact(tt.encoded, &s)
			if err == nil {
				t.Error("Expected error, got nil")
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Errorf("Expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestTCPProtocol(t *testing.T) {
	signal := Signal{
		ICECandidates: []webrtc.ICECandidate{
			{
				Foundation: "tcp123",
				Priority:   1000,
				Address:    "192.168.1.1",
				Protocol:   webrtc.ICEProtocolTCP,
				Port:       443,
				Typ:        webrtc.ICECandidateTypeHost,
				Component:  1,
			},
		},
		ICEParameters: webrtc.ICEParameters{
			UsernameFragment: "tcptest123456789",
			Password:         "tcppassword1234567890123456789012",
		},
		DTLSParameters: webrtc.DTLSParameters{
			Role: webrtc.DTLSRoleClient,
			Fingerprints: []webrtc.DTLSFingerprint{
				{
					Algorithm: "sha-256",
					Value:     "00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00",
				},
			},
		},
	}

	encoded, err := EncodeCompact(signal)
	if err != nil {
		t.Fatalf("EncodeCompact failed: %v", err)
	}

	var decoded Signal
	err = DecodeCompact(encoded, &decoded)
	if err != nil {
		t.Fatalf("DecodeCompact failed: %v", err)
	}

	if decoded.ICECandidates[0].Protocol != webrtc.ICEProtocolTCP {
		t.Errorf("Protocol mismatch: got %v, want TCP", decoded.ICECandidates[0].Protocol)
	}
}

func BenchmarkEncodeCompact(b *testing.B) {
	signal := createCompactTestSignal()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = EncodeCompact(signal)
	}
}

func BenchmarkEncodeJSON(b *testing.B) {
	signal := createCompactTestSignal()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		jsonBytes, _ := json.Marshal(signal)
		_ = base64.StdEncoding.EncodeToString(jsonBytes)
	}
}

func BenchmarkDecodeCompact(b *testing.B) {
	signal := createCompactTestSignal()
	encoded, _ := EncodeCompact(signal)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var s Signal
		_ = DecodeCompact(encoded, &s)
	}
}

func BenchmarkDecodeJSON(b *testing.B) {
	signal := createCompactTestSignal()
	jsonBytes, _ := json.Marshal(signal)
	encoded := base64.StdEncoding.EncodeToString(jsonBytes)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var s Signal
		b, _ := base64.StdEncoding.DecodeString(encoded)
		_ = json.Unmarshal(b, &s)
	}
}

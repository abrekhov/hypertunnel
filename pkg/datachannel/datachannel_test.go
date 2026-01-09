/*
 *   Copyright © 2021-2026 Anton Brekhov <anton@abrekhov.ru>
 *   All rights reserved.
 */
package datachannel

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSignal is a helper to create a test Signal struct
func createTestSignal() Signal {
	return Signal{
		ICECandidates: []webrtc.ICECandidate{},
		ICEParameters: webrtc.ICEParameters{
			UsernameFragment: "test-ufrag",
			Password:         "test-password",
		},
		DTLSParameters: webrtc.DTLSParameters{
			Role: webrtc.DTLSRoleAuto,
			Fingerprints: []webrtc.DTLSFingerprint{
				{
					Algorithm: "sha-256",
					Value:     "00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF",
				},
			},
		},
		SCTPCapabilities: webrtc.SCTPCapabilities{
			MaxMessageSize: 65536,
		},
	}
}

func TestEncode(t *testing.T) {
	t.Run("encodes signal to base64", func(t *testing.T) {
		signal := createTestSignal()
		encoded := Encode(signal)

		// Should return a non-empty string
		assert.NotEmpty(t, encoded, "Encoded signal should not be empty")

		// Should be valid base64
		_, err := base64.StdEncoding.DecodeString(encoded)
		assert.NoError(t, err, "Encoded signal should be valid base64")
	})

	t.Run("encoded signal can be decoded back", func(t *testing.T) {
		original := createTestSignal()
		encoded := Encode(original)

		// Decode back
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		require.NoError(t, err)

		var signal Signal
		err = json.Unmarshal(decoded, &signal)
		require.NoError(t, err)

		// Compare key fields
		assert.Equal(t, original.ICEParameters.UsernameFragment, signal.ICEParameters.UsernameFragment)
		assert.Equal(t, original.ICEParameters.Password, signal.ICEParameters.Password)
		assert.Equal(t, original.DTLSParameters.Role, signal.DTLSParameters.Role)
	})

	t.Run("encodes empty signal", func(t *testing.T) {
		signal := Signal{}
		encoded := Encode(signal)

		assert.NotEmpty(t, encoded, "Should encode empty signal")

		// Verify it's valid base64
		_, err := base64.StdEncoding.DecodeString(encoded)
		assert.NoError(t, err)
	})

	t.Run("encodes complex signal with multiple ICE candidates", func(t *testing.T) {
		signal := createTestSignal()

		// Add multiple ICE candidates (using empty structs for testing)
		signal.ICECandidates = []webrtc.ICECandidate{
			{},
			{},
		}

		encoded := Encode(signal)
		assert.NotEmpty(t, encoded)
	})

	t.Run("produces deterministic output", func(t *testing.T) {
		signal := createTestSignal()
		encoded1 := Encode(signal)
		encoded2 := Encode(signal)

		assert.Equal(t, encoded1, encoded2, "Same signal should produce same encoding")
	})
}

func TestDecode(t *testing.T) {
	t.Run("decodes valid base64 signal", func(t *testing.T) {
		original := createTestSignal()
		encoded := Encode(original)

		var decoded Signal
		assert.NotPanics(t, func() {
			Decode(encoded, &decoded)
		})

		assert.Equal(t, original.ICEParameters.UsernameFragment, decoded.ICEParameters.UsernameFragment)
		assert.Equal(t, original.ICEParameters.Password, decoded.ICEParameters.Password)
	})

	t.Run("handles empty ICE candidates", func(t *testing.T) {
		signal := Signal{
			ICECandidates: []webrtc.ICECandidate{},
			ICEParameters: webrtc.ICEParameters{
				UsernameFragment: "test",
				Password:         "pass",
			},
		}

		encoded := Encode(signal)
		var decoded Signal
		Decode(encoded, &decoded)

		assert.Empty(t, decoded.ICECandidates)
		assert.Equal(t, signal.ICEParameters.UsernameFragment, decoded.ICEParameters.UsernameFragment)
	})

	t.Run("decode and encode round trip", func(t *testing.T) {
		original := createTestSignal()
		encoded1 := Encode(original)

		var decoded Signal
		Decode(encoded1, &decoded)

		encoded2 := Encode(decoded)

		// The two encoded strings should be equal (round trip)
		assert.JSONEq(t, mustDecodeBase64JSON(encoded1), mustDecodeBase64JSON(encoded2),
			"Round trip encoding should produce equivalent JSON")
	})
}

func TestSignalStructure(t *testing.T) {
	t.Run("signal has all required fields", func(t *testing.T) {
		signal := Signal{
			ICECandidates: []webrtc.ICECandidate{},
		}

		// These fields should be present (not testing nil, just verifying structure exists)
		assert.NotNil(t, signal.ICECandidates, "ICECandidates should be present")
		// Other fields are value types, not pointers, so they're always "present"
		_ = signal.ICEParameters
		_ = signal.DTLSParameters
		_ = signal.SCTPCapabilities
	})

	t.Run("can marshal to JSON", func(t *testing.T) {
		signal := createTestSignal()
		data, err := json.Marshal(signal)

		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		// Verify JSON structure
		var result map[string]interface{}
		err = json.Unmarshal(data, &result)
		assert.NoError(t, err)
	})
}

func TestEncodeDecode_EdgeCases(t *testing.T) {
	t.Run("handles special characters in signal data", func(t *testing.T) {
		signal := createTestSignal()
		signal.ICEParameters.UsernameFragment = "test-ü-ñ-特殊"
		signal.ICEParameters.Password = "pässwörd-🔐"

		encoded := Encode(signal)
		var decoded Signal
		Decode(encoded, &decoded)

		assert.Equal(t, signal.ICEParameters.UsernameFragment, decoded.ICEParameters.UsernameFragment)
		assert.Equal(t, signal.ICEParameters.Password, decoded.ICEParameters.Password)
	})

	t.Run("handles large signal data", func(t *testing.T) {
		signal := createTestSignal()

		// Test with large parameters instead of ICE candidates
		// (ICECandidates require complex validation)
		signal.ICEParameters.Password = string(make([]byte, 1000))

		encoded := Encode(signal)
		assert.NotEmpty(t, encoded)

		var decoded Signal
		Decode(encoded, &decoded)

		// Verify the large data is preserved
		assert.Equal(t, len(signal.ICEParameters.Password), len(decoded.ICEParameters.Password))
	})
}

// Helper function to decode base64 and return JSON string
func mustDecodeBase64JSON(encoded string) string {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		panic(err)
	}
	return string(decoded)
}

func BenchmarkEncode(b *testing.B) {
	signal := createTestSignal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Encode(signal)
	}
}

func BenchmarkDecode(b *testing.B) {
	signal := createTestSignal()
	encoded := Encode(signal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var decoded Signal
		Decode(encoded, &decoded)
	}
}

func BenchmarkEncodeDecodeRoundTrip(b *testing.B) {
	signal := createTestSignal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoded := Encode(signal)
		var decoded Signal
		Decode(encoded, &decoded)
	}
}

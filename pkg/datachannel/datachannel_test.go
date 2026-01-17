/*
 *   Copyright © 2021-2026 Anton Brekhov <anton@abrekhov.ru>
 *   All rights reserved.
 */
package datachannel

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
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

		// Decode back using the Decode function (handles compact format)
		var signal Signal
		Decode(encoded, &signal)

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
		// Both use compact format now, so we compare the decoded signals
		var decoded2 Signal
		Decode(encoded2, &decoded2)

		assert.Equal(t, decoded.ICEParameters.UsernameFragment, decoded2.ICEParameters.UsernameFragment)
		assert.Equal(t, decoded.ICEParameters.Password, decoded2.ICEParameters.Password)
		assert.Equal(t, decoded.DTLSParameters.Role, decoded2.DTLSParameters.Role)
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

		// Test with large parameters (max 255 bytes due to compact format length prefix)
		// The compact format truncates to 255 bytes
		largePassword := string(make([]byte, 200))
		signal.ICEParameters.Password = largePassword

		encoded := Encode(signal)
		assert.NotEmpty(t, encoded)

		var decoded Signal
		Decode(encoded, &decoded)

		// Verify the data is preserved (up to 255 bytes)
		assert.Equal(t, len(largePassword), len(decoded.ICEParameters.Password))
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

func TestEncode_InvalidObjectExits(t *testing.T) {
	runHelperProcess(t, "encode-invalid")
}

func TestDecode_InvalidBase64Exits(t *testing.T) {
	runHelperProcess(t, "decode-invalid-base64")
}

func TestDecode_InvalidJSONExits(t *testing.T) {
	runHelperProcess(t, "decode-invalid-json")
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mode := ""
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			mode = os.Args[i+1]
			break
		}
	}

	switch mode {
	case "encode-invalid":
		Encode(make(chan int))
	case "decode-invalid-base64":
		Decode("not-base64!!", &Signal{})
	case "decode-invalid-json":
		Decode(base64.StdEncoding.EncodeToString([]byte("{bad json")), &Signal{})
	}

	os.Exit(0)
}

func runHelperProcess(t *testing.T, mode string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", mode)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected process to exit with error for mode %s", mode)
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() == 0 {
			t.Fatalf("expected non-zero exit code for mode %s", mode)
		}
		return
	}

	t.Fatalf("unexpected error running helper process for mode %s: %v", mode, err)
}

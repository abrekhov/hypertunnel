/*
 *   Copyright © 2021-2026 Anton Brekhov <anton@abrekhov.ru>
 *   All rights reserved.
 */
package hashutils

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromKeyToAESKey(t *testing.T) {
	t.Run("returns 32 byte key for AES-256", func(t *testing.T) {
		key := FromKeyToAESKey("test-password")
		assert.Len(t, key, 32, "AES-256 requires 32 byte key")
	})

	t.Run("produces deterministic output", func(t *testing.T) {
		password := "my-secure-password"
		key1 := FromKeyToAESKey(password)
		key2 := FromKeyToAESKey(password)

		assert.Equal(t, key1, key2, "Same password should produce same key")
	})

	t.Run("different passwords produce different keys", func(t *testing.T) {
		key1 := FromKeyToAESKey("password1")
		key2 := FromKeyToAESKey("password2")

		assert.NotEqual(t, key1, key2, "Different passwords should produce different keys")
	})

	t.Run("empty string produces valid key", func(t *testing.T) {
		key := FromKeyToAESKey("")
		assert.Len(t, key, 32, "Empty string should still produce 32 byte key")
		assert.NotNil(t, key, "Key should not be nil")
	})

	t.Run("unicode characters are handled correctly", func(t *testing.T) {
		key1 := FromKeyToAESKey("パスワード")  // Japanese
		key2 := FromKeyToAESKey("пароль") // Russian
		key3 := FromKeyToAESKey("🔐🔑")     // Emojis

		assert.Len(t, key1, 32)
		assert.Len(t, key2, 32)
		assert.Len(t, key3, 32)
		assert.NotEqual(t, key1, key2)
		assert.NotEqual(t, key2, key3)
	})

	t.Run("known test vectors", func(t *testing.T) {
		// SHA-256 of "test" should be a known value
		testCases := []struct {
			input    string
			expected string
		}{
			{
				input:    "test",
				expected: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
			{
				input:    "password",
				expected: "5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8",
			},
			{
				input:    "",
				expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.input, func(t *testing.T) {
				key := FromKeyToAESKey(tc.input)
				hexKey := hex.EncodeToString(key)
				assert.Equal(t, tc.expected, hexKey,
					"SHA-256 hash should match expected value")
			})
		}
	})

	t.Run("long passwords are handled correctly", func(t *testing.T) {
		longPassword := string(make([]byte, 10000)) // 10KB password
		key := FromKeyToAESKey(longPassword)
		assert.Len(t, key, 32, "Long passwords should still produce 32 byte key")
	})
}

func TestFromKeyToAESKey_Consistency(t *testing.T) {
	// Test that the function is consistent across multiple calls
	password := "consistency-test"
	iterations := 1000

	firstKey := FromKeyToAESKey(password)
	for i := 0; i < iterations; i++ {
		key := FromKeyToAESKey(password)
		require.Equal(t, firstKey, key,
			"Key generation should be consistent across iterations")
	}
}

func BenchmarkFromKeyToAESKey(b *testing.B) {
	password := "benchmark-password"

	b.Run("short password", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			FromKeyToAESKey(password)
		}
	})

	b.Run("long password", func(b *testing.B) {
		longPassword := string(make([]byte, 1024))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			FromKeyToAESKey(longPassword)
		}
	})
}

/*
 *   Copyright © 2021-2026 Anton Brekhov <anton@abrekhov.ru>
 *   All rights reserved.
 */

//go:build integration

package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/abrekhov/hypertunnel/pkg/hashutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncryptDecryptRoundTrip tests the full encryption and decryption cycle
func TestEncryptDecryptRoundTrip(t *testing.T) {
	tempDir := t.TempDir()

	testCases := []struct {
		name       string
		content    string
		keyphrase  string
		bufferSize int32
	}{
		{
			name:       "small text file",
			content:    "Hello, World!",
			keyphrase:  "test-password",
			bufferSize: 1024,
		},
		{
			name:       "empty file",
			content:    "",
			keyphrase:  "empty-password",
			bufferSize: 1024,
		},
		{
			name:       "unicode content",
			content:    "Hello 世界! Привет мир! 🔐🔑",
			keyphrase:  "unicode-pass-世界",
			bufferSize: 512,
		},
		{
			name:       "large content with small buffer",
			content:    string(make([]byte, 10000)),
			keyphrase:  "large-file-password",
			bufferSize: 256,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create original file
			originalFile := filepath.Join(tempDir, "original-"+tc.name+".txt")
			err := os.WriteFile(originalFile, []byte(tc.content), 0644)
			require.NoError(t, err)

			// Encrypt
			encryptedFile := originalFile + ".enc"
			err = encryptFile(originalFile, encryptedFile, tc.keyphrase, tc.bufferSize)
			require.NoError(t, err)

			// Verify encrypted file exists and is different from original
			encryptedData, err := os.ReadFile(encryptedFile)
			require.NoError(t, err)
			if len(tc.content) > 0 {
				assert.NotEqual(t, []byte(tc.content), encryptedData[:len(tc.content)])
			}

			// Decrypt
			decryptedFile := encryptedFile + ".dec"
			err = decryptFile(encryptedFile, decryptedFile, tc.keyphrase, tc.bufferSize)
			require.NoError(t, err)

			// Verify decrypted content matches original
			decryptedData, err := os.ReadFile(decryptedFile)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(decryptedData),
				"Decrypted content should match original")

			// Cleanup
			os.Remove(originalFile)
			os.Remove(encryptedFile)
			os.Remove(decryptedFile)
		})
	}
}

// TestEncryptionWithWrongPassword verifies that decryption with wrong password produces garbage
func TestEncryptionWithWrongPassword(t *testing.T) {
	tempDir := t.TempDir()
	content := "Secret content that should not be decryptable"
	correctPassword := "correct-password"
	wrongPassword := "wrong-password"

	// Create and encrypt file
	originalFile := filepath.Join(tempDir, "secret.txt")
	err := os.WriteFile(originalFile, []byte(content), 0644)
	require.NoError(t, err)

	encryptedFile := originalFile + ".enc"
	err = encryptFile(originalFile, encryptedFile, correctPassword, 1024)
	require.NoError(t, err)

	// Try to decrypt with wrong password
	decryptedFile := encryptedFile + ".dec"
	err = decryptFile(encryptedFile, decryptedFile, wrongPassword, 1024)
	require.NoError(t, err)

	// Verify decrypted content is NOT the original
	decryptedData, err := os.ReadFile(decryptedFile)
	require.NoError(t, err)
	assert.NotEqual(t, content, string(decryptedData),
		"Wrong password should not decrypt correctly")
}

// TestEncryptionIVRandomness verifies that encrypting the same file produces different output
func TestEncryptionIVRandomness(t *testing.T) {
	tempDir := t.TempDir()
	content := "Same content, different IV"
	password := "test-password"

	// Encrypt same content twice
	originalFile := filepath.Join(tempDir, "original.txt")
	err := os.WriteFile(originalFile, []byte(content), 0644)
	require.NoError(t, err)

	encryptedFile1 := originalFile + ".enc1"
	err = encryptFile(originalFile, encryptedFile1, password, 1024)
	require.NoError(t, err)

	encryptedFile2 := originalFile + ".enc2"
	err = encryptFile(originalFile, encryptedFile2, password, 1024)
	require.NoError(t, err)

	// Read encrypted files
	encrypted1, err := os.ReadFile(encryptedFile1)
	require.NoError(t, err)
	encrypted2, err := os.ReadFile(encryptedFile2)
	require.NoError(t, err)

	// The encrypted content should be different (due to different IVs)
	// But both should decrypt to the same original content
	assert.NotEqual(t, encrypted1, encrypted2,
		"Same content encrypted twice should produce different ciphertext (different IVs)")

	// Decrypt both and verify they produce the same content
	decryptedFile1 := encryptedFile1 + ".dec"
	err = decryptFile(encryptedFile1, decryptedFile1, password, 1024)
	require.NoError(t, err)

	decryptedFile2 := encryptedFile2 + ".dec"
	err = decryptFile(encryptedFile2, decryptedFile2, password, 1024)
	require.NoError(t, err)

	decrypted1, err := os.ReadFile(decryptedFile1)
	require.NoError(t, err)
	decrypted2, err := os.ReadFile(decryptedFile2)
	require.NoError(t, err)

	assert.Equal(t, content, string(decrypted1))
	assert.Equal(t, content, string(decrypted2))
}

// encryptFile mimics the encryption logic from cmd/encrypt.go
func encryptFile(inputPath, outputPath, keyphrase string, bufferSize int32) error {
	keyHash := hashutils.FromKeyToAESKey(keyphrase)

	infile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer infile.Close()

	outfile, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outfile.Close()

	block, err := aes.NewCipher(keyHash)
	if err != nil {
		return err
	}

	iv := make([]byte, block.BlockSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return err
	}

	buf := make([]byte, bufferSize)
	stream := cipher.NewCTR(block, iv)

	for {
		n, err := infile.Read(buf)
		if n > 0 {
			stream.XORKeyStream(buf, buf[:n])
			if _, err := outfile.Write(buf[:n]); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	// Write IV at the end
	if _, err := outfile.Write(iv); err != nil {
		return err
	}

	return nil
}

// decryptFile mimics the decryption logic from cmd/decrypt.go
func decryptFile(inputPath, outputPath, keyphrase string, bufferSize int32) error {
	keyHash := hashutils.FromKeyToAESKey(keyphrase)

	infile, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer infile.Close()

	fi, err := infile.Stat()
	if err != nil {
		return err
	}

	outfile, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer outfile.Close()

	block, err := aes.NewCipher(keyHash)
	if err != nil {
		return err
	}

	iv := make([]byte, block.BlockSize())
	msgLen := fi.Size() - int64(len(iv))

	// Read IV from end of file
	if _, err = infile.ReadAt(iv, msgLen); err != nil {
		return err
	}

	// Reset file pointer to beginning
	if _, err = infile.Seek(0, 0); err != nil {
		return err
	}

	buf := make([]byte, bufferSize)
	stream := cipher.NewCTR(block, iv)

	for {
		n, err := infile.Read(buf)
		if n > 0 {
			if n > int(msgLen) {
				n = int(msgLen)
			}
			msgLen -= int64(n)
			stream.XORKeyStream(buf, buf[:n])
			if _, err := outfile.Write(buf[:n]); err != nil {
				return err
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

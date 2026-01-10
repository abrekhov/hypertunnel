/*
Copyright © 2024 Anton Brekhov <anton@abrekhov.ru>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transfer

import (
	"bytes"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChecksumWriter(t *testing.T) {
	t.Run("creates checksum writer with underlying writer", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		require.NotNil(t, cw)
		assert.NotNil(t, cw.writer)
		assert.NotNil(t, cw.hash)
	})
}

func TestChecksumWriter_Write(t *testing.T) {
	t.Run("writes data to underlying writer and updates hash", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		data := []byte("hello world")
		n, err := cw.Write(data)

		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf.Bytes())
	})

	t.Run("handles multiple writes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		cw.Write([]byte("hello "))
		cw.Write([]byte("world"))

		assert.Equal(t, "hello world", buf.String())
	})

	t.Run("handles empty write", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		n, err := cw.Write([]byte{})
		require.NoError(t, err)
		assert.Equal(t, 0, n)
	})
}

func TestChecksumWriter_Sum(t *testing.T) {
	t.Run("returns correct SHA256 checksum", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		data := []byte("hello world")
		cw.Write(data)

		sum := cw.Sum()

		// Expected SHA256 of "hello world"
		expected := sha256.Sum256(data)
		assert.Equal(t, expected[:], sum)
	})

	t.Run("returns correct checksum for empty data", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		sum := cw.Sum()

		// SHA256 of empty string
		expected := sha256.Sum256([]byte{})
		assert.Equal(t, expected[:], sum)
	})

	t.Run("returns correct checksum for large data", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		// Write 1MB of data
		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		cw.Write(data)

		sum := cw.Sum()
		expected := sha256.Sum256(data)
		assert.Equal(t, expected[:], sum)
	})

	t.Run("checksum is accumulated across multiple writes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		cw.Write([]byte("hello "))
		cw.Write([]byte("world"))

		sum := cw.Sum()

		// Should be checksum of "hello world", not individual parts
		expected := sha256.Sum256([]byte("hello world"))
		assert.Equal(t, expected[:], sum)
	})
}

func TestChecksumWriter_SumHex(t *testing.T) {
	t.Run("returns checksum as hex string", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		cw.Write([]byte("hello world"))

		sumHex := cw.SumHex()

		// Known SHA256 of "hello world"
		expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		assert.Equal(t, expected, sumHex)
	})
}

func TestChecksumWriter_BytesWritten(t *testing.T) {
	t.Run("tracks total bytes written", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		cw.Write([]byte("hello"))
		cw.Write([]byte("world"))

		assert.Equal(t, int64(10), cw.BytesWritten())
	})

	t.Run("starts at zero", func(t *testing.T) {
		buf := &bytes.Buffer{}
		cw := NewChecksumWriter(buf)

		assert.Equal(t, int64(0), cw.BytesWritten())
	})
}

func TestNewChecksumReader(t *testing.T) {
	t.Run("creates checksum reader with underlying reader", func(t *testing.T) {
		data := bytes.NewReader([]byte("test data"))
		cr := NewChecksumReader(data)

		require.NotNil(t, cr)
		assert.NotNil(t, cr.reader)
		assert.NotNil(t, cr.hash)
	})
}

func TestChecksumReader_Read(t *testing.T) {
	t.Run("reads data from underlying reader and updates hash", func(t *testing.T) {
		data := []byte("hello world")
		cr := NewChecksumReader(bytes.NewReader(data))

		buf := make([]byte, len(data))
		n, err := cr.Read(buf)

		require.NoError(t, err)
		assert.Equal(t, len(data), n)
		assert.Equal(t, data, buf)
	})

	t.Run("handles multiple reads", func(t *testing.T) {
		data := []byte("hello world")
		cr := NewChecksumReader(bytes.NewReader(data))

		buf1 := make([]byte, 5)
		buf2 := make([]byte, 6)

		n1, _ := cr.Read(buf1)
		n2, _ := cr.Read(buf2)

		assert.Equal(t, 5, n1)
		assert.Equal(t, 6, n2)
		assert.Equal(t, []byte("hello"), buf1)
		assert.Equal(t, []byte(" world"), buf2)
	})

	t.Run("returns EOF at end of data", func(t *testing.T) {
		data := []byte("test")
		cr := NewChecksumReader(bytes.NewReader(data))

		buf := make([]byte, 100)
		n, err := cr.Read(buf)

		assert.Equal(t, 4, n)
		// First read may or may not return EOF depending on implementation
		// Read again to get EOF
		_, err = cr.Read(buf)
		assert.Equal(t, io.EOF, err)
	})
}

func TestChecksumReader_Sum(t *testing.T) {
	t.Run("returns correct checksum after reading all data", func(t *testing.T) {
		data := []byte("hello world")
		cr := NewChecksumReader(bytes.NewReader(data))

		// Read all data
		buf := make([]byte, len(data))
		cr.Read(buf)

		sum := cr.Sum()
		expected := sha256.Sum256(data)
		assert.Equal(t, expected[:], sum)
	})

	t.Run("checksum only covers data read so far", func(t *testing.T) {
		data := []byte("hello world")
		cr := NewChecksumReader(bytes.NewReader(data))

		// Read only part of data
		buf := make([]byte, 5)
		cr.Read(buf) // reads "hello"

		sum := cr.Sum()
		expected := sha256.Sum256([]byte("hello"))
		assert.Equal(t, expected[:], sum)
	})
}

func TestChecksumReader_BytesRead(t *testing.T) {
	t.Run("tracks total bytes read", func(t *testing.T) {
		data := []byte("hello world")
		cr := NewChecksumReader(bytes.NewReader(data))

		buf := make([]byte, 5)
		cr.Read(buf)
		cr.Read(buf)

		assert.Equal(t, int64(10), cr.BytesRead())
	})
}

func TestVerifyChecksum(t *testing.T) {
	t.Run("returns true for matching checksum", func(t *testing.T) {
		data := []byte("hello world")
		expected := sha256.Sum256(data)

		result := VerifyChecksum(data, expected[:])
		assert.True(t, result)
	})

	t.Run("returns false for mismatched checksum", func(t *testing.T) {
		data := []byte("hello world")
		wrong := sha256.Sum256([]byte("different data"))

		result := VerifyChecksum(data, wrong[:])
		assert.False(t, result)
	})

	t.Run("returns true for empty data with correct checksum", func(t *testing.T) {
		data := []byte{}
		expected := sha256.Sum256(data)

		result := VerifyChecksum(data, expected[:])
		assert.True(t, result)
	})
}

func TestCalculateFileChecksum(t *testing.T) {
	t.Run("calculates checksum of file", func(t *testing.T) {
		// Create temp file
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		content := []byte("hello world")
		err := os.WriteFile(tmpFile, content, 0644)
		require.NoError(t, err)

		// Calculate checksum
		sum, err := CalculateFileChecksum(tmpFile)
		require.NoError(t, err)

		expected := sha256.Sum256(content)
		assert.Equal(t, expected[:], sum)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := CalculateFileChecksum("/nonexistent/file.txt")
		assert.Error(t, err)
	})

	t.Run("handles empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "empty.txt")
		err := os.WriteFile(tmpFile, []byte{}, 0644)
		require.NoError(t, err)

		sum, err := CalculateFileChecksum(tmpFile)
		require.NoError(t, err)

		expected := sha256.Sum256([]byte{})
		assert.Equal(t, expected[:], sum)
	})

	t.Run("handles large file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "large.bin")

		// Create 1MB file
		data := make([]byte, 1024*1024)
		for i := range data {
			data[i] = byte(i % 256)
		}
		err := os.WriteFile(tmpFile, data, 0644)
		require.NoError(t, err)

		sum, err := CalculateFileChecksum(tmpFile)
		require.NoError(t, err)

		expected := sha256.Sum256(data)
		assert.Equal(t, expected[:], sum)
	})
}

func TestVerifyFileChecksum(t *testing.T) {
	t.Run("returns true for matching file checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		content := []byte("hello world")
		err := os.WriteFile(tmpFile, content, 0644)
		require.NoError(t, err)

		expected := sha256.Sum256(content)
		result, err := VerifyFileChecksum(tmpFile, expected[:])
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("returns false for mismatched file checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		content := []byte("hello world")
		err := os.WriteFile(tmpFile, content, 0644)
		require.NoError(t, err)

		wrong := sha256.Sum256([]byte("different"))
		result, err := VerifyFileChecksum(tmpFile, wrong[:])
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := VerifyFileChecksum("/nonexistent/file.txt", []byte{})
		assert.Error(t, err)
	})
}

func TestChecksumHexString(t *testing.T) {
	t.Run("converts checksum bytes to hex string", func(t *testing.T) {
		sum := sha256.Sum256([]byte("hello world"))
		hexStr := ChecksumToHex(sum[:])

		assert.Equal(t, 64, len(hexStr)) // SHA256 = 32 bytes = 64 hex chars
		expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		assert.Equal(t, expected, hexStr)
	})
}

func TestHexToChecksum(t *testing.T) {
	t.Run("converts hex string to checksum bytes", func(t *testing.T) {
		hexStr := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
		sum, err := HexToChecksum(hexStr)

		require.NoError(t, err)
		expected := sha256.Sum256([]byte("hello world"))
		assert.Equal(t, expected[:], sum)
	})

	t.Run("returns error for invalid hex", func(t *testing.T) {
		_, err := HexToChecksum("not-valid-hex")
		assert.Error(t, err)
	})

	t.Run("returns error for wrong length", func(t *testing.T) {
		_, err := HexToChecksum("abc123")
		assert.Error(t, err)
	})
}

// Benchmarks
func BenchmarkChecksumWriter_Write(b *testing.B) {
	buf := &bytes.Buffer{}
	cw := NewChecksumWriter(buf)
	data := make([]byte, 65534) // WebRTC chunk size

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cw.Write(data)
	}
}

func BenchmarkCalculateFileChecksum(b *testing.B) {
	// Create temp file
	tmpDir := b.TempDir()
	tmpFile := filepath.Join(tmpDir, "benchmark.bin")
	data := make([]byte, 1024*1024) // 1MB
	os.WriteFile(tmpFile, data, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateFileChecksum(tmpFile)
	}
}


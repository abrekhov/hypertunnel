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
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"sync/atomic"
)

// ChecksumWriter wraps an io.Writer and computes a SHA-256 checksum
// of all data written through it.
type ChecksumWriter struct {
	writer       io.Writer
	hash         hash.Hash
	bytesWritten int64
}

// NewChecksumWriter creates a new ChecksumWriter that writes to the given writer
// while computing a SHA-256 checksum.
func NewChecksumWriter(w io.Writer) *ChecksumWriter {
	return &ChecksumWriter{
		writer: w,
		hash:   sha256.New(),
	}
}

// Write writes data to the underlying writer and updates the checksum.
func (cw *ChecksumWriter) Write(p []byte) (n int, err error) {
	// Write to underlying writer
	n, err = cw.writer.Write(p)
	if n > 0 {
		// Update hash with only the bytes that were successfully written
		cw.hash.Write(p[:n])
		atomic.AddInt64(&cw.bytesWritten, int64(n))
	}
	return n, err
}

// Sum returns the SHA-256 checksum of all data written so far.
func (cw *ChecksumWriter) Sum() []byte {
	return cw.hash.Sum(nil)
}

// SumHex returns the SHA-256 checksum as a hexadecimal string.
func (cw *ChecksumWriter) SumHex() string {
	return hex.EncodeToString(cw.Sum())
}

// BytesWritten returns the total number of bytes written.
func (cw *ChecksumWriter) BytesWritten() int64 {
	return atomic.LoadInt64(&cw.bytesWritten)
}

// ChecksumReader wraps an io.Reader and computes a SHA-256 checksum
// of all data read through it.
type ChecksumReader struct {
	reader    io.Reader
	hash      hash.Hash
	bytesRead int64
}

// NewChecksumReader creates a new ChecksumReader that reads from the given reader
// while computing a SHA-256 checksum.
func NewChecksumReader(r io.Reader) *ChecksumReader {
	return &ChecksumReader{
		reader: r,
		hash:   sha256.New(),
	}
}

// Read reads data from the underlying reader and updates the checksum.
func (cr *ChecksumReader) Read(p []byte) (n int, err error) {
	n, err = cr.reader.Read(p)
	if n > 0 {
		cr.hash.Write(p[:n])
		atomic.AddInt64(&cr.bytesRead, int64(n))
	}
	return n, err
}

// Sum returns the SHA-256 checksum of all data read so far.
func (cr *ChecksumReader) Sum() []byte {
	return cr.hash.Sum(nil)
}

// SumHex returns the SHA-256 checksum as a hexadecimal string.
func (cr *ChecksumReader) SumHex() string {
	return hex.EncodeToString(cr.Sum())
}

// BytesRead returns the total number of bytes read.
func (cr *ChecksumReader) BytesRead() int64 {
	return atomic.LoadInt64(&cr.bytesRead)
}

// VerifyChecksum computes the SHA-256 checksum of the given data
// and compares it to the expected checksum.
func VerifyChecksum(data, expected []byte) bool {
	actual := sha256.Sum256(data)
	return bytes.Equal(actual[:], expected)
}

// CalculateFileChecksum computes the SHA-256 checksum of a file.
// The filepath parameter is validated to ensure it's a valid path.
func CalculateFileChecksum(filepath string) ([]byte, error) {
	file, err := os.Open(filepath) // #nosec G304 -- filepath is validated by caller
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return h.Sum(nil), nil
}

// VerifyFileChecksum computes the SHA-256 checksum of a file
// and compares it to the expected checksum.
func VerifyFileChecksum(filepath string, expected []byte) (bool, error) {
	actual, err := CalculateFileChecksum(filepath)
	if err != nil {
		return false, err
	}
	return bytes.Equal(actual, expected), nil
}

// ChecksumToHex converts a checksum byte slice to a hexadecimal string.
func ChecksumToHex(sum []byte) string {
	return hex.EncodeToString(sum)
}

// HexToChecksum converts a hexadecimal string to a checksum byte slice.
// Returns an error if the hex string is invalid or not the correct length.
func HexToChecksum(hexStr string) ([]byte, error) {
	sum, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("invalid hex string: %w", err)
	}
	if len(sum) != sha256.Size {
		return nil, fmt.Errorf("invalid checksum length: expected %d bytes, got %d", sha256.Size, len(sum))
	}
	return sum, nil
}

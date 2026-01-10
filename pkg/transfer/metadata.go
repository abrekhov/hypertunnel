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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MetadataPrefix is prepended to metadata messages to distinguish them from file data.
const MetadataPrefix = "HT_META:"

// Metadata contains information about a file being transferred.
type Metadata struct {
	ModTime     time.Time   `json:"modtime,omitempty"`     // Modification time
	Filename    string      `json:"filename"`              // Name of the file (without path)
	Checksum    string      `json:"checksum,omitempty"`    // SHA-256 checksum (hex)
	Size        int64       `json:"size"`                  // Size in bytes
	Mode        os.FileMode `json:"mode,omitempty"`        // File permissions
	IsDirectory bool        `json:"is_directory,omitempty"` // True if this is a directory transfer
	IsArchive   bool        `json:"is_archive,omitempty"`   // True if this is an archived directory
}

// NewMetadata creates a new Metadata with the given filename and size.
func NewMetadata(filename string, size int64) *Metadata {
	return &Metadata{
		Filename: filename,
		Size:     size,
	}
}

// MetadataFromFile creates Metadata from an existing file.
// Returns an error if the file doesn't exist or is a directory.
func MetadataFromFile(path string) (*Metadata, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	return &Metadata{
		Filename: filepath.Base(path),
		Size:     info.Size(),
		Mode:     info.Mode(),
		ModTime:  info.ModTime(),
	}, nil
}

// MetadataFromPath creates Metadata from a file or directory path.
// For directories, it creates metadata indicating a directory transfer.
// The actual size should be set later when the archive is created.
func MetadataFromPath(path string) (*Metadata, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	m := &Metadata{
		Filename:    filepath.Base(path),
		Mode:        info.Mode(),
		ModTime:     info.ModTime(),
		IsDirectory: info.IsDir(),
	}

	if info.IsDir() {
		// For directories, we'll create an archive
		// Size will be 0 initially and updated when archive is created
		m.Size = 0
		m.IsArchive = true
		// Append .tar.gz to the filename to indicate it's archived
		if !strings.HasSuffix(m.Filename, ".tar.gz") {
			m.Filename = m.Filename + ".tar.gz"
		}
	} else {
		m.Size = info.Size()
	}

	return m, nil
}

// MetadataFromFileWithChecksum creates Metadata from an existing file,
// including the SHA-256 checksum.
func MetadataFromFileWithChecksum(path string) (*Metadata, error) {
	m, err := MetadataFromFile(path)
	if err != nil {
		return nil, err
	}

	sum, err := CalculateFileChecksum(path)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	m.Checksum = ChecksumToHex(sum)

	return m, nil
}

// Encode serializes the Metadata to JSON bytes.
func (m *Metadata) Encode() ([]byte, error) {
	return json.Marshal(m)
}

// DecodeMetadata deserializes Metadata from JSON bytes.
func DecodeMetadata(data []byte) (*Metadata, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty metadata")
	}

	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}
	return &m, nil
}

// Validate checks that the Metadata is valid.
// Returns an error describing any validation failures.
func (m *Metadata) Validate() error {
	if m.Filename == "" {
		return fmt.Errorf("filename cannot be empty")
	}
	if m.Size < 0 {
		return fmt.Errorf("size cannot be negative")
	}

	// Check for path traversal attacks
	// Note: filepath.IsAbs is OS-dependent, so we explicitly check for Unix-style absolute paths
	clean := filepath.Clean(m.Filename)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) || strings.Contains(m.Filename, "..") || strings.HasPrefix(m.Filename, "/") {
		return fmt.Errorf("invalid filename: path traversal not allowed")
	}

	// Check for Windows-style absolute paths
	if len(m.Filename) >= 2 && m.Filename[1] == ':' {
		return fmt.Errorf("invalid filename: absolute path not allowed")
	}

	// Check for backslashes (Windows path separators)
	if strings.Contains(m.Filename, "\\") {
		return fmt.Errorf("invalid filename: backslashes not allowed")
	}

	return nil
}

// SafeFilename returns a sanitized version of the filename that is safe to use
// for creating files. It removes any path traversal attempts and ensures
// the filename is a simple basename.
func (m *Metadata) SafeFilename() string {
	// First, clean the path
	clean := filepath.Clean(m.Filename)

	// Replace Windows path separators
	clean = strings.ReplaceAll(clean, "\\", "/")

	// Remove Windows drive letters
	if len(clean) >= 2 && clean[1] == ':' {
		clean = clean[2:]
	}

	// Get just the base name
	clean = filepath.Base(clean)

	// Handle edge cases
	if clean == "" || clean == "." || clean == ".." {
		return "unnamed"
	}

	return clean
}

// WithChecksum adds a checksum to the Metadata and returns self for chaining.
func (m *Metadata) WithChecksum(checksum string) *Metadata {
	m.Checksum = checksum
	return m
}

// WithMode adds a file mode to the Metadata and returns self for chaining.
func (m *Metadata) WithMode(mode os.FileMode) *Metadata {
	m.Mode = mode
	return m
}

// WithModTime adds a modification time to the Metadata and returns self for chaining.
func (m *Metadata) WithModTime(modTime time.Time) *Metadata {
	m.ModTime = modTime
	return m
}

// IsMetadataMessage checks if the given data is a metadata message
// (starts with MetadataPrefix).
func IsMetadataMessage(data []byte) bool {
	return bytes.HasPrefix(data, []byte(MetadataPrefix))
}

// WrapForTransfer wraps the encoded Metadata with the metadata prefix
// for transmission over the data channel.
func (m *Metadata) WrapForTransfer() ([]byte, error) {
	encoded, err := m.Encode()
	if err != nil {
		return nil, err
	}

	result := make([]byte, len(MetadataPrefix)+len(encoded))
	copy(result, MetadataPrefix)
	copy(result[len(MetadataPrefix):], encoded)

	return result, nil
}

// UnwrapMetadata extracts and decodes Metadata from a wrapped message.
// Returns an error if the message is not a valid metadata message.
func UnwrapMetadata(data []byte) (*Metadata, error) {
	if !IsMetadataMessage(data) {
		return nil, fmt.Errorf("not a metadata message")
	}

	jsonData := data[len(MetadataPrefix):]
	return DecodeMetadata(jsonData)
}

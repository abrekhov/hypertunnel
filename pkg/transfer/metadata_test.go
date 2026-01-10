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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetadata(t *testing.T) {
	t.Run("creates metadata with required fields", func(t *testing.T) {
		m := NewMetadata("test.txt", 1024)

		assert.Equal(t, "test.txt", m.Filename)
		assert.Equal(t, int64(1024), m.Size)
		assert.Empty(t, m.Checksum)
		assert.Zero(t, m.Mode)
		assert.True(t, m.ModTime.IsZero())
	})
}

func TestMetadataFromFile(t *testing.T) {
	t.Run("creates metadata from existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		content := []byte("hello world")
		err := os.WriteFile(tmpFile, content, 0644)
		require.NoError(t, err)

		m, err := MetadataFromFile(tmpFile)

		require.NoError(t, err)
		assert.Equal(t, "testfile.txt", m.Filename)
		assert.Equal(t, int64(len(content)), m.Size)
		assert.NotZero(t, m.Mode)
		assert.False(t, m.ModTime.IsZero())
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		_, err := MetadataFromFile("/nonexistent/file.txt")
		assert.Error(t, err)
	})

	t.Run("returns error for directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := MetadataFromFile(tmpDir)
		assert.Error(t, err)
	})

	t.Run("includes checksum when requested", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		content := []byte("hello world")
		err := os.WriteFile(tmpFile, content, 0644)
		require.NoError(t, err)

		m, err := MetadataFromFileWithChecksum(tmpFile)

		require.NoError(t, err)
		assert.NotEmpty(t, m.Checksum)
		assert.Equal(t, 64, len(m.Checksum)) // SHA256 hex = 64 chars
	})
}

func TestMetadata_Encode(t *testing.T) {
	t.Run("encodes metadata to JSON bytes", func(t *testing.T) {
		m := &Metadata{
			Filename: "test.txt",
			Size:     1024,
			Checksum: "abc123",
		}

		encoded, err := m.Encode()

		require.NoError(t, err)
		assert.Contains(t, string(encoded), "test.txt")
		assert.Contains(t, string(encoded), "1024")
		assert.Contains(t, string(encoded), "abc123")
	})

	t.Run("handles special characters in filename", func(t *testing.T) {
		m := &Metadata{
			Filename: "test file with spaces & special.txt",
			Size:     100,
		}

		encoded, err := m.Encode()
		require.NoError(t, err)

		// Decode to verify
		decoded, err := DecodeMetadata(encoded)
		require.NoError(t, err)
		assert.Equal(t, m.Filename, decoded.Filename)
	})

	t.Run("handles unicode filename", func(t *testing.T) {
		m := &Metadata{
			Filename: "测试文件.txt",
			Size:     100,
		}

		encoded, err := m.Encode()
		require.NoError(t, err)

		decoded, err := DecodeMetadata(encoded)
		require.NoError(t, err)
		assert.Equal(t, m.Filename, decoded.Filename)
	})
}

func TestDecodeMetadata(t *testing.T) {
	t.Run("decodes valid JSON to metadata", func(t *testing.T) {
		json := []byte(`{"filename":"test.txt","size":1024,"checksum":"abc123"}`)

		m, err := DecodeMetadata(json)

		require.NoError(t, err)
		assert.Equal(t, "test.txt", m.Filename)
		assert.Equal(t, int64(1024), m.Size)
		assert.Equal(t, "abc123", m.Checksum)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		_, err := DecodeMetadata([]byte("not json"))
		assert.Error(t, err)
	})

	t.Run("returns error for empty data", func(t *testing.T) {
		_, err := DecodeMetadata([]byte{})
		assert.Error(t, err)
	})

	t.Run("handles missing optional fields", func(t *testing.T) {
		json := []byte(`{"filename":"test.txt","size":1024}`)

		m, err := DecodeMetadata(json)

		require.NoError(t, err)
		assert.Equal(t, "test.txt", m.Filename)
		assert.Empty(t, m.Checksum)
		assert.Zero(t, m.Mode)
	})
}

func TestMetadata_Validate(t *testing.T) {
	t.Run("valid metadata passes validation", func(t *testing.T) {
		m := &Metadata{
			Filename: "test.txt",
			Size:     1024,
		}

		err := m.Validate()
		assert.NoError(t, err)
	})

	t.Run("empty filename fails validation", func(t *testing.T) {
		m := &Metadata{
			Filename: "",
			Size:     1024,
		}

		err := m.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "filename")
	})

	t.Run("negative size fails validation", func(t *testing.T) {
		m := &Metadata{
			Filename: "test.txt",
			Size:     -1,
		}

		err := m.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "size")
	})

	t.Run("zero size is valid", func(t *testing.T) {
		m := &Metadata{
			Filename: "empty.txt",
			Size:     0,
		}

		err := m.Validate()
		assert.NoError(t, err)
	})

	t.Run("path traversal in filename fails validation", func(t *testing.T) {
		testCases := []string{
			"../../../etc/passwd",
			"..\\..\\windows\\system32\\config",
			"/etc/passwd",
			"C:\\Windows\\System32",
			"foo/../bar",
		}

		for _, tc := range testCases {
			t.Run(tc, func(t *testing.T) {
				m := &Metadata{
					Filename: tc,
					Size:     100,
				}

				err := m.Validate()
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid")
			})
		}
	})

	t.Run("filename with subdirectory is valid", func(t *testing.T) {
		m := &Metadata{
			Filename: "subdir/file.txt",
			Size:     100,
		}

		err := m.Validate()
		// This might be valid or invalid depending on implementation
		// For single file transfer, subdirectories might be disallowed
		// For directory transfer, they would be allowed
		// For now, we allow it
		assert.NoError(t, err)
	})
}

func TestMetadata_SafeFilename(t *testing.T) {
	t.Run("returns filename for simple name", func(t *testing.T) {
		m := &Metadata{Filename: "test.txt"}
		assert.Equal(t, "test.txt", m.SafeFilename())
	})

	t.Run("strips path components for path traversal", func(t *testing.T) {
		m := &Metadata{Filename: "../../../etc/passwd"}
		assert.Equal(t, "passwd", m.SafeFilename())
	})

	t.Run("strips absolute path", func(t *testing.T) {
		m := &Metadata{Filename: "/etc/passwd"}
		assert.Equal(t, "passwd", m.SafeFilename())
	})

	t.Run("handles Windows paths", func(t *testing.T) {
		m := &Metadata{Filename: "C:\\Windows\\System32\\config"}
		safe := m.SafeFilename()
		assert.NotContains(t, safe, "\\")
		assert.NotContains(t, safe, ":")
	})

	t.Run("returns 'unnamed' for empty after sanitization", func(t *testing.T) {
		m := &Metadata{Filename: ".."}
		assert.Equal(t, "unnamed", m.SafeFilename())
	})
}

func TestMetadata_WithChecksum(t *testing.T) {
	t.Run("adds checksum to metadata", func(t *testing.T) {
		m := NewMetadata("test.txt", 1024)
		m.WithChecksum("abc123")

		assert.Equal(t, "abc123", m.Checksum)
	})

	t.Run("returns self for chaining", func(t *testing.T) {
		m := NewMetadata("test.txt", 1024)
		result := m.WithChecksum("abc123")

		assert.Same(t, m, result)
	})
}

func TestMetadata_WithMode(t *testing.T) {
	t.Run("adds file mode to metadata", func(t *testing.T) {
		m := NewMetadata("test.txt", 1024)
		m.WithMode(0755)

		assert.Equal(t, os.FileMode(0755), m.Mode)
	})
}

func TestMetadata_WithModTime(t *testing.T) {
	t.Run("adds modification time to metadata", func(t *testing.T) {
		m := NewMetadata("test.txt", 1024)
		modTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		m.WithModTime(modTime)

		assert.Equal(t, modTime, m.ModTime)
	})
}

func TestMetadata_IsMetadataMessage(t *testing.T) {
	t.Run("returns true for metadata prefix", func(t *testing.T) {
		data := []byte(MetadataPrefix + `{"filename":"test.txt","size":1024}`)
		assert.True(t, IsMetadataMessage(data))
	})

	t.Run("returns false for non-metadata message", func(t *testing.T) {
		data := []byte("some random binary data")
		assert.False(t, IsMetadataMessage(data))
	})

	t.Run("returns false for empty data", func(t *testing.T) {
		assert.False(t, IsMetadataMessage([]byte{}))
	})
}

func TestMetadata_WrapUnwrap(t *testing.T) {
	t.Run("wrap adds prefix and unwrap removes it", func(t *testing.T) {
		m := &Metadata{
			Filename: "test.txt",
			Size:     1024,
			Checksum: "abc123",
		}

		wrapped, err := m.WrapForTransfer()
		require.NoError(t, err)

		assert.True(t, IsMetadataMessage(wrapped))

		unwrapped, err := UnwrapMetadata(wrapped)
		require.NoError(t, err)
		assert.Equal(t, m.Filename, unwrapped.Filename)
		assert.Equal(t, m.Size, unwrapped.Size)
		assert.Equal(t, m.Checksum, unwrapped.Checksum)
	})

	t.Run("unwrap returns error for non-metadata message", func(t *testing.T) {
		_, err := UnwrapMetadata([]byte("not metadata"))
		assert.Error(t, err)
	})
}

// Benchmarks
func BenchmarkMetadata_Encode(b *testing.B) {
	m := &Metadata{
		Filename: "large_file_with_long_name.tar.gz",
		Size:     1024 * 1024 * 1024,
		Checksum: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		Mode:     0644,
		ModTime:  time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Encode()
	}
}

func BenchmarkDecodeMetadata(b *testing.B) {
	m := &Metadata{
		Filename: "large_file_with_long_name.tar.gz",
		Size:     1024 * 1024 * 1024,
		Checksum: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		Mode:     0644,
		ModTime:  time.Now(),
	}
	encoded, _ := m.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeMetadata(encoded)
	}
}

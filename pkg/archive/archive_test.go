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

package archive

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndExtractTarGz(t *testing.T) {
	// Create a temporary source directory
	srcDir, err := os.MkdirTemp("", "hypertunnel-test-src-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(srcDir) }()

	// Create test files and directories
	testFiles := map[string]string{
		"file1.txt":          "Hello, World!",
		"subdir/file2.txt":   "Test content",
		"subdir/file3.txt":   "More test content",
		"subdir2/file4.txt":  "Another file",
		"subdir2/deep/file5": "Deep file",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(srcDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0750))           // #nosec G301 - test file
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0600)) // #nosec G306 - test file
	}

	// Create archive
	var buf bytes.Buffer
	opts := DefaultOptions()
	bytesWritten, err := CreateTarGz(&buf, srcDir, opts)
	require.NoError(t, err)
	assert.Greater(t, bytesWritten, int64(0))
	assert.Greater(t, buf.Len(), 0)

	// Extract archive to new directory
	destDir, err := os.MkdirTemp("", "hypertunnel-test-dest-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(destDir) }()

	err = ExtractTarGz(&buf, destDir, opts)
	require.NoError(t, err)

	// Verify extracted files
	for path, expectedContent := range testFiles {
		fullPath := filepath.Join(destDir, path)
		content, err := os.ReadFile(fullPath) // #nosec G304 - test file
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content), "File content mismatch: %s", path)
	}
}

func TestCreateTarGzSingleFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "hypertunnel-test-file-*.txt")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	testContent := "Single file content"
	_, err = tmpFile.WriteString(testContent)
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Create archive
	var buf bytes.Buffer
	opts := DefaultOptions()
	bytesWritten, err := CreateTarGz(&buf, tmpFile.Name(), opts)
	require.NoError(t, err)
	assert.Greater(t, bytesWritten, int64(0))

	// Extract archive
	destDir, err := os.MkdirTemp("", "hypertunnel-test-dest-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(destDir) }()

	err = ExtractTarGz(&buf, destDir, opts)
	require.NoError(t, err)

	// Verify extracted file
	extractedPath := filepath.Join(destDir, filepath.Base(tmpFile.Name()))
	content, err := os.ReadFile(extractedPath) // #nosec G304 - test file
	require.NoError(t, err)
	assert.Equal(t, testContent, string(content))
}

func TestExcludePatterns(t *testing.T) {
	// Create a temporary source directory
	srcDir, err := os.MkdirTemp("", "hypertunnel-test-exclude-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(srcDir) }()

	// Create test files
	testFiles := []string{
		"include.txt",
		"exclude.log",
		"subdir/include2.txt",
		"subdir/exclude2.log",
	}

	for _, path := range testFiles {
		fullPath := filepath.Join(srcDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0750)) // #nosec G301 - test file
		require.NoError(t, os.WriteFile(fullPath, []byte("content"), 0600)) // #nosec G306 - test file
	}

	// Create archive with exclude patterns
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.ExcludePatterns = []string{"*.log"}
	_, err = CreateTarGz(&buf, srcDir, opts)
	require.NoError(t, err)

	// Extract archive
	destDir, err := os.MkdirTemp("", "hypertunnel-test-dest-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(destDir) }()

	err = ExtractTarGz(&buf, destDir, opts)
	require.NoError(t, err)

	// Verify only .txt files were included
	_, err = os.Stat(filepath.Join(destDir, "include.txt"))
	assert.NoError(t, err, "include.txt should exist")

	_, err = os.Stat(filepath.Join(destDir, "exclude.log"))
	assert.True(t, os.IsNotExist(err), "exclude.log should not exist")

	_, err = os.Stat(filepath.Join(destDir, "subdir/include2.txt"))
	assert.NoError(t, err, "subdir/include2.txt should exist")

	_, err = os.Stat(filepath.Join(destDir, "subdir/exclude2.log"))
	assert.True(t, os.IsNotExist(err), "subdir/exclude2.log should not exist")
}

func TestIsValidPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"normal path", "file.txt", true},
		{"subdirectory", "subdir/file.txt", true},
		{"deep path", "a/b/c/file.txt", true},
		{"parent traversal", "../file.txt", false},
		{"absolute path", "/etc/passwd", false},
		{"hidden traversal", "subdir/../../../etc/passwd", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsDirectory(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "hypertunnel-test-dir-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create temporary file
	tmpFile, err := os.CreateTemp(tmpDir, "test-*.txt")
	require.NoError(t, err)
	_ = tmpFile.Close()

	// Test directory
	isDir, err := IsDirectory(tmpDir)
	require.NoError(t, err)
	assert.True(t, isDir)

	// Test file
	isDir, err = IsDirectory(tmpFile.Name())
	require.NoError(t, err)
	assert.False(t, isDir)

	// Test non-existent path
	_, err = IsDirectory("/path/that/does/not/exist")
	assert.Error(t, err)
}

func TestShouldExclude(t *testing.T) {
	patterns := []string{"*.log", "*.tmp", "node_modules"}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"exclude log file", "app.log", true},
		{"exclude tmp file", "cache.tmp", true},
		{"exclude node_modules", "node_modules", true},
		{"include txt file", "file.txt", false},
		{"include go file", "main.go", false},
		{"exclude nested log", "subdir/debug.log", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExclude(tt.path, patterns)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPreservePermissions(t *testing.T) {
	// Create a temporary source directory
	srcDir, err := os.MkdirTemp("", "hypertunnel-test-perms-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(srcDir) }()

	// Create a file with specific permissions
	testFile := filepath.Join(srcDir, "executable.sh")
	err = os.WriteFile(testFile, []byte("#!/bin/bash\necho test"), 0700) // #nosec G306 - test file needs exec perms
	require.NoError(t, err)

	// Create archive with permission preservation
	var buf bytes.Buffer
	opts := DefaultOptions()
	opts.PreservePermissions = true
	_, err = CreateTarGz(&buf, srcDir, opts)
	require.NoError(t, err)

	// Extract archive
	destDir, err := os.MkdirTemp("", "hypertunnel-test-dest-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(destDir) }()

	err = ExtractTarGz(&buf, destDir, opts)
	require.NoError(t, err)

	// Verify permissions
	extractedFile := filepath.Join(destDir, "executable.sh")
	info, err := os.Stat(extractedFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0700), info.Mode().Perm())
}

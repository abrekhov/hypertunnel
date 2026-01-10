/*
 *   Copyright © 2021-2026 Anton Brekhov <anton@abrekhov.ru>
 *   All rights reserved.
 */
package datachannel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAskForConfirmation(t *testing.T) {
	t.Run("returns true for 'y' input", func(t *testing.T) {
		input := strings.NewReader("y\n")
		result := askForConfirmation("Test question", input)
		assert.True(t, result, "Should return true for 'y' input")
	})

	t.Run("returns true for 'yes' input", func(t *testing.T) {
		input := strings.NewReader("yes\n")
		result := askForConfirmation("Test question", input)
		assert.True(t, result, "Should return true for 'yes' input")
	})

	t.Run("returns true for 'Y' input (case insensitive)", func(t *testing.T) {
		input := strings.NewReader("Y\n")
		result := askForConfirmation("Test question", input)
		assert.True(t, result, "Should return true for 'Y' input")
	})

	t.Run("returns true for 'YES' input (case insensitive)", func(t *testing.T) {
		input := strings.NewReader("YES\n")
		result := askForConfirmation("Test question", input)
		assert.True(t, result, "Should return true for 'YES' input")
	})

	t.Run("returns false for 'n' input", func(t *testing.T) {
		input := strings.NewReader("n\n")
		result := askForConfirmation("Test question", input)
		assert.False(t, result, "Should return false for 'n' input")
	})

	t.Run("returns false for 'no' input", func(t *testing.T) {
		input := strings.NewReader("no\n")
		result := askForConfirmation("Test question", input)
		assert.False(t, result, "Should return false for 'no' input")
	})

	t.Run("returns false for 'N' input (case insensitive)", func(t *testing.T) {
		input := strings.NewReader("N\n")
		result := askForConfirmation("Test question", input)
		assert.False(t, result, "Should return false for 'N' input")
	})

	t.Run("handles empty input and retries", func(t *testing.T) {
		// Empty lines followed by a valid response
		input := strings.NewReader("\n\ny\n")
		result := askForConfirmation("Test question", input)
		assert.True(t, result, "Should handle empty input and retry")
	})

	t.Run("returns false after 3 invalid attempts", func(t *testing.T) {
		// Provide 3 empty inputs (should exhaust retries)
		input := strings.NewReader("\n\n\n\n")
		result := askForConfirmation("Test question", input)
		assert.False(t, result, "Should return false after exhausting retries")
	})

	t.Run("handles whitespace in input", func(t *testing.T) {
		input := strings.NewReader("  y  \n")
		result := askForConfirmation("Test question", input)
		assert.True(t, result, "Should trim whitespace from input")
	})

	t.Run("returns false for invalid input after retries", func(t *testing.T) {
		// Invalid inputs that exhaust retries
		input := strings.NewReader("maybe\n\n\n")
		result := askForConfirmation("Test question", input)
		assert.False(t, result, "Should return false for invalid input")
	})
}

func TestFileOverwriteCheck(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	t.Run("detects existing file correctly", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tempDir, "existing-file.txt")
		err := os.WriteFile(testFile, []byte("existing content"), 0644)
		require.NoError(t, err)

		// Check if file exists
		_, err = os.Stat(testFile)

		// BUG: The current code uses os.IsExist(err) which is incorrect
		// os.IsExist(err) returns true if the ERROR indicates the file exists
		// But when the file exists, os.Stat returns err == nil
		// The correct check should be: err == nil (file exists) or !os.IsNotExist(err)

		// This test demonstrates the bug
		if err == nil {
			// File exists - this is correct
			assert.Nil(t, err, "File should exist")

			// But the buggy code checks: os.IsExist(err)
			// When err is nil, os.IsExist(err) returns false!
			assert.False(t, os.IsExist(err),
				"BUG: os.IsExist(nil) returns false when file exists")
		}
	})

	t.Run("correctly identifies non-existent file", func(t *testing.T) {
		// Check for a file that doesn't exist
		testFile := filepath.Join(tempDir, "non-existent-file.txt")
		_, err := os.Stat(testFile)

		// File doesn't exist, err should not be nil
		assert.Error(t, err, "Should return error for non-existent file")
		assert.True(t, os.IsNotExist(err), "Error should indicate file doesn't exist")

		// The correct check for "file exists" is:
		// if err == nil || !os.IsNotExist(err)
		fileExists := err == nil
		assert.False(t, fileExists, "File should not exist")
	})

	t.Run("demonstrates correct file existence check", func(t *testing.T) {
		// Create a test file
		existingFile := filepath.Join(tempDir, "test-exists.txt")
		err := os.WriteFile(existingFile, []byte("test"), 0644)
		require.NoError(t, err)

		// CORRECT way to check if file exists:
		_, err = os.Stat(existingFile)
		if err == nil {
			// File exists
			t.Log("✓ Correct: File exists when err == nil")
		} else if os.IsNotExist(err) {
			// File does not exist
			t.Error("File should exist")
		} else {
			// Other error (permission, etc.)
			t.Errorf("Unexpected error: %v", err)
		}

		// INCORRECT way (current bug in handlers.go:19):
		_, err = os.Stat(existingFile)
		if os.IsExist(err) {
			// This will NEVER be true for existing files!
			t.Error("BUG: This branch never executes for existing files")
		}
	})
}

func TestFileTransferHandler_FileOverwriteProtection(t *testing.T) {
	// This is an integration-style test that would require WebRTC setup
	// For now, we document the expected behavior

	t.Run("should prevent overwriting existing files", func(t *testing.T) {
		// Expected behavior:
		// 1. Check if file exists: _, err := os.Stat(filename)
		// 2. If err == nil, file exists - should warn/prompt user
		// 3. If os.IsNotExist(err), file doesn't exist - safe to create
		// 4. If other error, handle appropriately

		// Current bug: uses os.IsExist(err) which returns false when err is nil
		// This means existing files are never detected!

		t.Skip("Integration test - requires WebRTC DataChannel setup")
	})
}

func TestFileOperations(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("can create new file when it doesn't exist", func(t *testing.T) {
		newFile := filepath.Join(tempDir, "new-file.txt")

		// Verify file doesn't exist
		_, err := os.Stat(newFile)
		require.True(t, os.IsNotExist(err), "File should not exist initially")

		// Create the file
		fd, err := os.Create(newFile)
		require.NoError(t, err)
		defer fd.Close()

		// Write some data
		_, err = fd.Write([]byte("test data"))
		require.NoError(t, err)

		// Verify file now exists
		info, err := os.Stat(newFile)
		require.NoError(t, err)
		assert.Equal(t, "new-file.txt", info.Name())
	})

	t.Run("can detect existing file before overwrite", func(t *testing.T) {
		existingFile := filepath.Join(tempDir, "existing.txt")

		// Create initial file
		err := os.WriteFile(existingFile, []byte("original"), 0644)
		require.NoError(t, err)

		// Check if file exists - CORRECT way
		_, err = os.Stat(existingFile)
		fileExists := (err == nil)

		assert.True(t, fileExists, "Should correctly detect existing file")

		// In production, we should prompt user here before overwriting
	})
}

// BenchmarkAskForConfirmation measures performance of confirmation dialog
func BenchmarkAskForConfirmation(b *testing.B) {
	input := strings.NewReader("y\n")
	question := "Benchmark question?"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = askForConfirmation(question, input)
		input.Reset("y\n")
	}
}

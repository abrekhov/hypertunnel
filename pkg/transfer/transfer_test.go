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
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestState_String(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateIdle, "idle"},
		{StateMetadata, "exchanging metadata"},
		{StateTransferring, "transferring"},
		{StateVerifying, "verifying"},
		{StateComplete, "complete"},
		{StateFailed, "failed"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestNewTransfer(t *testing.T) {
	t.Run("creates transfer in idle state", func(t *testing.T) {
		tr := NewTransfer()
		require.NotNil(t, tr)
		assert.Equal(t, StateIdle, tr.state)
	})
}

func TestTransfer_StartSend(t *testing.T) {
	t.Run("initializes transfer for existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		content := []byte("hello world")
		err := os.WriteFile(tmpFile, content, 0600)
		require.NoError(t, err)

		tr := NewTransfer()
		m, err := tr.StartSend(tmpFile, false)

		require.NoError(t, err)
		assert.NotNil(t, m)
		assert.Equal(t, "testfile.txt", m.Filename)
		assert.Equal(t, int64(len(content)), m.Size)
		assert.Equal(t, StateMetadata, tr.state)
	})

	t.Run("calculates checksum when requested", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		content := []byte("hello world")
		err := os.WriteFile(tmpFile, content, 0600)
		require.NoError(t, err)

		tr := NewTransfer()
		m, err := tr.StartSend(tmpFile, true)

		require.NoError(t, err)
		assert.NotEmpty(t, m.Checksum)
		assert.Equal(t, 64, len(m.Checksum)) // SHA256 hex
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		tr := NewTransfer()
		_, err := tr.StartSend("/nonexistent/file.txt", false)

		assert.Error(t, err)
		assert.Equal(t, StateFailed, tr.state)
	})
}

func TestTransfer_StartReceive(t *testing.T) {
	t.Run("initializes transfer for valid metadata", func(t *testing.T) {
		tr := NewTransfer()
		m := &Metadata{
			Filename: "test.txt",
			Size:     1024,
		}

		err := tr.StartReceive(m)

		require.NoError(t, err)
		assert.Equal(t, StateTransferring, tr.state)
		assert.NotNil(t, tr.progress)
	})

	t.Run("returns error for invalid metadata", func(t *testing.T) {
		tr := NewTransfer()
		m := &Metadata{
			Filename: "", // Invalid
			Size:     1024,
		}

		err := tr.StartReceive(m)

		assert.Error(t, err)
		assert.Equal(t, StateFailed, tr.state)
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		tr := NewTransfer()
		m := &Metadata{
			Filename: "../../../etc/passwd",
			Size:     100,
		}

		err := tr.StartReceive(m)

		assert.Error(t, err)
		assert.Equal(t, StateFailed, tr.state)
	})
}

func TestTransfer_UpdateProgress(t *testing.T) {
	t.Run("updates progress and returns metrics", func(t *testing.T) {
		tr := NewTransfer()
		err := tr.StartReceive(&Metadata{Filename: "test.txt", Size: 1000})
		require.NoError(t, err)

		metrics := tr.UpdateProgress(500)

		require.NotNil(t, metrics)
		assert.Equal(t, int64(500), metrics.TransferredBytes)
		assert.InDelta(t, 50.0, metrics.Percentage, 0.1)
	})

	t.Run("accumulates progress", func(t *testing.T) {
		tr := NewTransfer()
		err := tr.StartReceive(&Metadata{Filename: "test.txt", Size: 1000})
		require.NoError(t, err)

		tr.UpdateProgress(200)
		tr.UpdateProgress(300)
		metrics := tr.UpdateProgress(250)

		assert.Equal(t, int64(750), metrics.TransferredBytes)
	})
}

func TestTransfer_Complete(t *testing.T) {
	t.Run("marks transfer as complete", func(t *testing.T) {
		tr := NewTransfer()
		err := tr.StartReceive(&Metadata{Filename: "test.txt", Size: 100})
		require.NoError(t, err)
		tr.UpdateProgress(100)

		tr.Complete()

		assert.True(t, tr.IsComplete())
		assert.Equal(t, StateComplete, tr.state)
	})
}

func TestTransfer_Fail(t *testing.T) {
	t.Run("marks transfer as failed with error", func(t *testing.T) {
		tr := NewTransfer()
		err := tr.StartReceive(&Metadata{Filename: "test.txt", Size: 100})
		require.NoError(t, err)

		testErr := assert.AnError
		tr.Fail(testErr)

		assert.True(t, tr.IsFailed())
		assert.Equal(t, StateFailed, tr.state)
		assert.Equal(t, testErr, tr.err)
	})
}

func TestTransfer_Status(t *testing.T) {
	t.Run("returns full status", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		err := os.WriteFile(tmpFile, []byte("hello world"), 0600)
		require.NoError(t, err)

		tr := NewTransfer()
		_, err = tr.StartSend(tmpFile, false)
		require.NoError(t, err)
		tr.MetadataSent()
		tr.UpdateProgress(5)

		status := tr.Status()

		require.NotNil(t, status)
		assert.Equal(t, StateTransferring, status.State)
		assert.NotNil(t, status.Metadata)
		assert.NotNil(t, status.Progress)
		assert.False(t, status.LastUpdate.IsZero())
	})
}

func TestTransfer_SetProgressCallback(t *testing.T) {
	t.Run("callback is called on state changes", func(t *testing.T) {
		var callCount int32

		tr := NewTransfer()
		tr.SetProgressCallback(func(_ *Status) {
			atomic.AddInt32(&callCount, 1)
		})

		err := tr.StartReceive(&Metadata{Filename: "test.txt", Size: 100})
		require.NoError(t, err)
		tr.UpdateProgress(50)
		tr.Complete()

		// Should be called: StartReceive, UpdateProgress, Complete
		assert.GreaterOrEqual(t, atomic.LoadInt32(&callCount), int32(3))
	})
}

func TestTransfer_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent progress updates", func(t *testing.T) {
		tr := NewTransfer()
		err := tr.StartReceive(&Metadata{Filename: "test.txt", Size: 100000})
		require.NoError(t, err)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					tr.UpdateProgress(10)
					_ = tr.Status()
				}
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		// All updates should be reflected
		metrics := tr.GetProgress()
		assert.Equal(t, int64(10000), metrics.TransferredBytes)
	})
}

func TestTransfer_MetadataSent(t *testing.T) {
	t.Run("transitions to transferring state", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		err := os.WriteFile(tmpFile, []byte("hello world"), 0600)
		require.NoError(t, err)

		tr := NewTransfer()
		_, err = tr.StartSend(tmpFile, false)
		require.NoError(t, err)
		assert.Equal(t, StateMetadata, tr.state)

		tr.MetadataSent()

		assert.Equal(t, StateTransferring, tr.state)
	})
}

func TestTransfer_GetMetadata(t *testing.T) {
	t.Run("returns metadata after send start", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		err := os.WriteFile(tmpFile, []byte("hello world"), 0600)
		require.NoError(t, err)

		tr := NewTransfer()
		_, err = tr.StartSend(tmpFile, false)
		require.NoError(t, err)

		m := tr.GetMetadata()

		assert.NotNil(t, m)
		assert.Equal(t, "testfile.txt", m.Filename)
	})

	t.Run("returns nil before initialization", func(t *testing.T) {
		tr := NewTransfer()
		assert.Nil(t, tr.GetMetadata())
	})
}

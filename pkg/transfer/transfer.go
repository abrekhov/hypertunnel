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
	"fmt"
	"sync"
	"time"
)

// State represents the current state of a file transfer.
type State int

const (
	// StateIdle indicates no transfer is in progress.
	StateIdle State = iota
	// StateMetadata indicates metadata is being exchanged.
	StateMetadata
	// StateTransferring indicates data is being transferred.
	StateTransferring
	// StateVerifying indicates checksum is being verified.
	StateVerifying
	// StateComplete indicates transfer completed successfully.
	StateComplete
	// StateFailed indicates transfer failed.
	StateFailed
)

// String returns a human-readable state name.
func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateMetadata:
		return "exchanging metadata"
	case StateTransferring:
		return "transferring"
	case StateVerifying:
		return "verifying"
	case StateComplete:
		return "complete"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Status contains the full status of an ongoing transfer.
type Status struct {
	Error        error
	Metadata     *Metadata
	Progress     *ProgressMetrics
	ExpectedSum  string
	ActualSum    string
	LastUpdate   time.Time
	State        State
	VerifyResult bool
}

// ProgressCallback is called periodically with transfer status updates.
type ProgressCallback func(status *Status)

// Transfer manages a file transfer with progress tracking and checksum verification.
type Transfer struct {
	err      error
	metadata *Metadata
	progress *Progress
	callback ProgressCallback
	mu       sync.RWMutex
	state    State
}

// NewTransfer creates a new Transfer instance.
func NewTransfer() *Transfer {
	return &Transfer{
		state: StateIdle,
	}
}

// SetProgressCallback sets a callback that will be called with status updates.
func (t *Transfer) SetProgressCallback(callback ProgressCallback) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.callback = callback
}

// StartSend initializes a transfer for sending a file.
// It creates the metadata and progress tracker.
func (t *Transfer) StartSend(path string, withChecksum bool) (*Metadata, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var err error
	if withChecksum {
		t.metadata, err = MetadataFromFileWithChecksum(path)
	} else {
		t.metadata, err = MetadataFromFile(path)
	}
	if err != nil {
		t.state = StateFailed
		t.err = err
		return nil, err
	}

	t.progress = NewProgress(t.metadata.Size)
	t.state = StateMetadata
	t.notifyCallback()

	return t.metadata, nil
}

// StartReceive initializes a transfer for receiving a file with the given metadata.
func (t *Transfer) StartReceive(metadata *Metadata) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := metadata.Validate(); err != nil {
		t.state = StateFailed
		t.err = err
		return err
	}

	t.metadata = metadata
	t.progress = NewProgress(metadata.Size)
	t.state = StateTransferring
	t.notifyCallback()

	return nil
}

// MetadataSent transitions to transferring state after metadata is sent.
func (t *Transfer) MetadataSent() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = StateTransferring
	t.notifyCallback()
}

// UpdateProgress updates the progress with the number of bytes transferred.
// Returns the current progress metrics.
func (t *Transfer) UpdateProgress(bytes int64) *ProgressMetrics {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.progress != nil {
		t.progress.Update(bytes)
	}
	t.notifyCallback()

	if t.progress != nil {
		return t.progress.Metrics()
	}
	return nil
}

// Complete marks the transfer as complete.
func (t *Transfer) Complete() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = StateComplete
	t.notifyCallback()
}

// Fail marks the transfer as failed with the given error.
func (t *Transfer) Fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = StateFailed
	t.err = err
	t.notifyCallback()
}

// Status returns the current transfer status.
func (t *Transfer) Status() *Status {
	t.mu.RLock()
	defer t.mu.RUnlock()

	status := &Status{
		State:      t.state,
		Metadata:   t.metadata,
		Error:      t.err,
		LastUpdate: time.Now(),
	}

	if t.progress != nil {
		status.Progress = t.progress.Metrics()
	}

	return status
}

// GetProgress returns the current progress metrics.
func (t *Transfer) GetProgress() *ProgressMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.progress != nil {
		return t.progress.Metrics()
	}
	return nil
}

// GetMetadata returns the transfer metadata.
func (t *Transfer) GetMetadata() *Metadata {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.metadata
}

// IsComplete returns true if the transfer is complete.
func (t *Transfer) IsComplete() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state == StateComplete
}

// IsFailed returns true if the transfer has failed.
func (t *Transfer) IsFailed() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state == StateFailed
}

// notifyCallback sends a status update to the registered callback.
// Must be called with mutex held.
func (t *Transfer) notifyCallback() {
	if t.callback == nil {
		return
	}

	status := &Status{
		State:      t.state,
		Metadata:   t.metadata,
		Error:      t.err,
		LastUpdate: time.Now(),
	}

	if t.progress != nil {
		status.Progress = t.progress.Metrics()
	}

	t.callback(status)
}

// PrintProgressLine prints a single-line progress update suitable for terminal output.
func PrintProgressLine(status *Status) {
	if status.Progress == nil {
		fmt.Printf("\rState: %s", status.State)
		return
	}

	p := status.Progress
	speed := FormatSpeed(p.BytesPerSecond)
	transferred := FormatSize(p.TransferredBytes)
	total := FormatSize(p.TotalBytes)
	eta := FormatDuration(time.Duration(p.ETASeconds * float64(time.Second)))

	fmt.Printf("\r[%.1f%%] %s / %s | %s | ETA: %s     ",
		p.Percentage, transferred, total, speed, eta)
}

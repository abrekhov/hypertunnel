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

// Package transfer provides file transfer utilities including progress tracking,
// checksum verification, and metadata handling for HyperTunnel P2P transfers.
package transfer

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Progress tracks the progress of a file transfer operation.
// It is safe for concurrent use.
type Progress struct {
	StartTime        time.Time // When the transfer started
	TotalBytes       int64     // Total bytes to transfer
	TransferredBytes int64     // Bytes transferred so far (atomic)
}

// ProgressMetrics contains a snapshot of transfer metrics.
type ProgressMetrics struct {
	TotalBytes       int64   // Total bytes to transfer
	TransferredBytes int64   // Bytes transferred so far
	Percentage       float64 // Completion percentage (0-100)
	BytesPerSecond   float64 // Current transfer speed
	ETASeconds       float64 // Estimated time remaining in seconds
	ElapsedSeconds   float64 // Time elapsed since start
}

// NewProgress creates a new Progress tracker for a transfer of the given size.
func NewProgress(totalBytes int64) *Progress {
	return &Progress{
		TotalBytes:       totalBytes,
		TransferredBytes: 0,
		StartTime:        time.Now(),
	}
}

// Update adds the given number of bytes to the transfer count.
// This method is safe for concurrent use.
func (p *Progress) Update(bytesTransferred int64) {
	atomic.AddInt64(&p.TransferredBytes, bytesTransferred)
}

// Percentage returns the completion percentage (0-100).
// Returns 0 if total is 0, and caps at 100 if transferred exceeds total.
func (p *Progress) Percentage() float64 {
	if p.TotalBytes == 0 {
		return 0.0
	}
	transferred := atomic.LoadInt64(&p.TransferredBytes)
	pct := float64(transferred) / float64(p.TotalBytes) * 100.0
	if pct > 100.0 {
		return 100.0
	}
	return pct
}

// Speed returns the current transfer speed in bytes per second.
func (p *Progress) Speed() float64 {
	transferred := atomic.LoadInt64(&p.TransferredBytes)
	if transferred == 0 {
		return 0.0
	}
	elapsed := time.Since(p.StartTime).Seconds()
	if elapsed <= 0 {
		return 0.0
	}
	return float64(transferred) / elapsed
}

// ETA returns the estimated time remaining until transfer completion.
// Returns 0 if already complete or cannot estimate.
func (p *Progress) ETA() time.Duration {
	if p.TotalBytes == 0 {
		return 0
	}
	transferred := atomic.LoadInt64(&p.TransferredBytes)
	if transferred >= p.TotalBytes {
		return 0
	}

	speed := p.Speed()
	if speed <= 0 {
		return 0
	}

	remaining := p.TotalBytes - transferred
	seconds := float64(remaining) / speed
	return time.Duration(seconds * float64(time.Second))
}

// Elapsed returns the time elapsed since the transfer started.
func (p *Progress) Elapsed() time.Duration {
	return time.Since(p.StartTime)
}

// IsComplete returns true if the transfer is complete (all bytes transferred).
func (p *Progress) IsComplete() bool {
	if p.TotalBytes == 0 {
		return true
	}
	return atomic.LoadInt64(&p.TransferredBytes) >= p.TotalBytes
}

// Metrics returns a snapshot of all transfer metrics at once.
func (p *Progress) Metrics() *ProgressMetrics {
	transferred := atomic.LoadInt64(&p.TransferredBytes)
	elapsed := time.Since(p.StartTime).Seconds()

	var pct, speed, eta float64

	if p.TotalBytes > 0 {
		pct = float64(transferred) / float64(p.TotalBytes) * 100.0
		if pct > 100.0 {
			pct = 100.0
		}
	}

	if elapsed > 0 && transferred > 0 {
		speed = float64(transferred) / elapsed
		if transferred < p.TotalBytes {
			remaining := p.TotalBytes - transferred
			eta = float64(remaining) / speed
		}
	}

	return &ProgressMetrics{
		TotalBytes:       p.TotalBytes,
		TransferredBytes: transferred,
		Percentage:       pct,
		BytesPerSecond:   speed,
		ETASeconds:       eta,
		ElapsedSeconds:   elapsed,
	}
}

// FormatSpeed formats a speed value (bytes per second) into a human-readable string.
func FormatSpeed(bytesPerSecond float64) string {
	return FormatSize(int64(bytesPerSecond)) + "/s"
}

// FormatSize formats a byte size into a human-readable string.
func FormatSize(bytes int64) string {
	const (
		KB = 1000
		MB = 1000 * KB
		GB = 1000 * MB
		TB = 1000 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatDuration formats a duration into HH:MM:SS format.
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

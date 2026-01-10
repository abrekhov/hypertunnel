/*
Copyright © 2025 Anton Brekhov <anton@abrekhov.ru>

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

import "time"

// Metrics represents a snapshot of transfer progress for logging or UI.
type Metrics struct {
	TotalBytes       int64
	TransferredBytes int64
	BytesPerSecond   float64
	ETA              time.Duration
	Elapsed          time.Duration
}

// Progress tracks transfer progress for a single send/receive operation.
type Progress struct {
	TotalBytes       int64
	TransferredBytes int64
	StartTime        time.Time
}

// NewProgress creates a new progress tracker for a transfer.
func NewProgress(totalBytes int64) *Progress {
	return &Progress{
		TotalBytes: totalBytes,
		StartTime:  time.Now(),
	}
}

// Add increments the transfer progress by n bytes.
func (p *Progress) Add(n int) {
	if n <= 0 {
		return
	}
	p.TransferredBytes += int64(n)
}

// Snapshot returns current progress metrics.
func (p *Progress) Snapshot(now time.Time) Metrics {
	elapsed := now.Sub(p.StartTime)
	var bytesPerSecond float64
	if elapsed > 0 {
		bytesPerSecond = float64(p.TransferredBytes) / elapsed.Seconds()
	}
	var eta time.Duration
	if p.TotalBytes > 0 && bytesPerSecond > 0 {
		remaining := float64(p.TotalBytes-p.TransferredBytes) / bytesPerSecond
		if remaining < 0 {
			remaining = 0
		}
		eta = time.Duration(remaining * float64(time.Second))
	}
	return Metrics{
		TotalBytes:       p.TotalBytes,
		TransferredBytes: p.TransferredBytes,
		BytesPerSecond:   bytesPerSecond,
		ETA:              eta,
		Elapsed:          elapsed,
	}
}

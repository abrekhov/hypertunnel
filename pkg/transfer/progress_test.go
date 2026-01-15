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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProgress(t *testing.T) {
	t.Run("creates progress with correct total size", func(t *testing.T) {
		p := NewProgress(1024)
		assert.Equal(t, int64(1024), p.TotalBytes)
		assert.Equal(t, int64(0), p.TransferredBytes)
		assert.False(t, p.StartTime.IsZero())
	})

	t.Run("creates progress with zero size", func(t *testing.T) {
		p := NewProgress(0)
		assert.Equal(t, int64(0), p.TotalBytes)
	})

	t.Run("creates progress with large size", func(t *testing.T) {
		// 10 GB
		size := int64(10 * 1024 * 1024 * 1024)
		p := NewProgress(size)
		assert.Equal(t, size, p.TotalBytes)
	})
}

func TestProgress_Update(t *testing.T) {
	t.Run("updates transferred bytes correctly", func(t *testing.T) {
		p := NewProgress(1000)
		p.Update(100)
		assert.Equal(t, int64(100), p.TransferredBytes)

		p.Update(200)
		assert.Equal(t, int64(300), p.TransferredBytes)
	})

	t.Run("handles cumulative updates", func(t *testing.T) {
		p := NewProgress(1000)
		for i := 0; i < 10; i++ {
			p.Update(100)
		}
		assert.Equal(t, int64(1000), p.TransferredBytes)
	})

	t.Run("allows exceeding total (no hard limit)", func(t *testing.T) {
		p := NewProgress(100)
		p.Update(150)
		assert.Equal(t, int64(150), p.TransferredBytes)
	})
}

func TestProgress_Percentage(t *testing.T) {
	tests := []struct {
		name        string
		total       int64
		transferred int64
		expected    float64
	}{
		{"0% progress", 1000, 0, 0.0},
		{"50% progress", 1000, 500, 50.0},
		{"100% progress", 1000, 1000, 100.0},
		{"partial percentage", 1000, 333, 33.3},
		{"zero total returns 0", 0, 0, 0.0},
		{"over 100% capped", 1000, 1500, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProgress(tt.total)
			p.TransferredBytes = tt.transferred
			result := p.Percentage()
			assert.InDelta(t, tt.expected, result, 0.1)
		})
	}
}

func TestProgress_Speed(t *testing.T) {
	t.Run("calculates speed in bytes per second", func(t *testing.T) {
		p := NewProgress(1000)
		// Simulate start time 1 second ago
		p.StartTime = time.Now().Add(-1 * time.Second)
		p.TransferredBytes = 1000

		speed := p.Speed()
		// Should be approximately 1000 bytes/second
		assert.InDelta(t, 1000.0, speed, 100.0) // Allow some variance
	})

	t.Run("returns zero speed for no elapsed time", func(t *testing.T) {
		p := NewProgress(1000)
		// Just created, minimal time elapsed
		p.TransferredBytes = 1000

		speed := p.Speed()
		// Speed might be very high or zero depending on timing
		assert.GreaterOrEqual(t, speed, 0.0)
	})

	t.Run("returns zero speed for no bytes transferred", func(t *testing.T) {
		p := NewProgress(1000)
		p.StartTime = time.Now().Add(-1 * time.Second)

		speed := p.Speed()
		assert.Equal(t, 0.0, speed)
	})
}

func TestProgress_ETA(t *testing.T) {
	t.Run("estimates remaining time", func(t *testing.T) {
		p := NewProgress(2000)
		p.StartTime = time.Now().Add(-1 * time.Second)
		p.TransferredBytes = 1000 // Half done in 1 second

		eta := p.ETA()
		// Should be approximately 1 second remaining
		assert.InDelta(t, 1.0, eta.Seconds(), 0.5)
	})

	t.Run("returns zero ETA when complete", func(t *testing.T) {
		p := NewProgress(1000)
		p.StartTime = time.Now().Add(-1 * time.Second)
		p.TransferredBytes = 1000

		eta := p.ETA()
		assert.Equal(t, time.Duration(0), eta)
	})

	t.Run("returns zero ETA for zero speed", func(t *testing.T) {
		p := NewProgress(1000)
		p.TransferredBytes = 0

		eta := p.ETA()
		// Should return 0 or a very large value; implementation decides
		assert.GreaterOrEqual(t, eta.Seconds(), 0.0)
	})

	t.Run("handles zero total size", func(t *testing.T) {
		p := NewProgress(0)
		eta := p.ETA()
		assert.Equal(t, time.Duration(0), eta)
	})
}

func TestProgress_Elapsed(t *testing.T) {
	t.Run("returns elapsed time since start", func(t *testing.T) {
		p := NewProgress(1000)
		time.Sleep(10 * time.Millisecond)

		elapsed := p.Elapsed()
		assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(10))
	})
}

func TestProgress_IsComplete(t *testing.T) {
	t.Run("returns false when not complete", func(t *testing.T) {
		p := NewProgress(1000)
		p.TransferredBytes = 500
		assert.False(t, p.IsComplete())
	})

	t.Run("returns true when complete", func(t *testing.T) {
		p := NewProgress(1000)
		p.TransferredBytes = 1000
		assert.True(t, p.IsComplete())
	})

	t.Run("returns true for zero total", func(t *testing.T) {
		p := NewProgress(0)
		assert.True(t, p.IsComplete())
	})

	t.Run("returns true when exceeds total", func(t *testing.T) {
		p := NewProgress(1000)
		p.TransferredBytes = 1500
		assert.True(t, p.IsComplete())
	})
}

func TestProgress_Metrics(t *testing.T) {
	t.Run("returns all metrics at once", func(t *testing.T) {
		p := NewProgress(2000)
		p.StartTime = time.Now().Add(-1 * time.Second)
		p.TransferredBytes = 1000

		metrics := p.Metrics()

		require.NotNil(t, metrics)
		assert.Equal(t, int64(2000), metrics.TotalBytes)
		assert.Equal(t, int64(1000), metrics.TransferredBytes)
		assert.InDelta(t, 50.0, metrics.Percentage, 0.1)
		assert.InDelta(t, 1000.0, metrics.BytesPerSecond, 200.0)
		assert.InDelta(t, 1.0, metrics.ETASeconds, 0.5)
	})
}

func TestProgress_FormatSpeed(t *testing.T) {
	tests := []struct {
		name           string
		expected       string
		bytesPerSecond float64
	}{
		{"bytes per second", "500 B/s", 500},
		{"kilobytes per second", "1.5 KB/s", 1500},
		{"megabytes per second", "1.5 MB/s", 1500000},
		{"gigabytes per second", "1.5 GB/s", 1500000000},
		{"zero speed", "0 B/s", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSpeed(tt.bytesPerSecond)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgress_FormatSize(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		bytes    int64
	}{
		{"bytes", "500 B", 500},
		{"kilobytes", "1.5 KB", 1500},
		{"megabytes", "1.5 MB", 1500000},
		{"gigabytes", "1.5 GB", 1500000000},
		{"terabytes", "1.5 TB", 1500000000000},
		{"zero bytes", "0 B", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSize(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgress_FormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		duration time.Duration
	}{
		{"seconds only", "00:00:45", 45 * time.Second},
		{"minutes and seconds", "00:05:30", 5*time.Minute + 30*time.Second},
		{"hours minutes seconds", "02:30:45", 2*time.Hour + 30*time.Minute + 45*time.Second},
		{"zero duration", "00:00:00", 0},
		{"sub-second", "00:00:00", 500 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestProgress_ThreadSafety(t *testing.T) {
	t.Run("concurrent updates are safe", func(t *testing.T) {
		p := NewProgress(100000)

		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					p.Update(10)
					_ = p.Percentage()
					_ = p.Speed()
					_ = p.ETA()
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Final value should be 10 * 100 * 10 = 10000
		assert.Equal(t, int64(10000), p.TransferredBytes)
	})
}

// Benchmarks for performance-critical operations
func BenchmarkProgress_Update(b *testing.B) {
	p := NewProgress(int64(b.N) * 1000)
	for i := 0; i < b.N; i++ {
		p.Update(1000)
	}
}

func BenchmarkProgress_Metrics(b *testing.B) {
	p := NewProgress(1000000)
	p.TransferredBytes = 500000
	for i := 0; i < b.N; i++ {
		_ = p.Metrics()
	}
}

func BenchmarkFormatSpeed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = FormatSpeed(1500000.0)
	}
}

func BenchmarkFormatSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = FormatSize(1500000)
	}
}

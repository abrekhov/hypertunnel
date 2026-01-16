/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */
package tui

import (
	"testing"
	"time"
)

func TestNewModel(t *testing.T) {
	m := NewModel(true, "test.txt", 1024)
	if m.state != StateConnection {
		t.Errorf("Expected state to be StateConnection, got %v", m.state)
	}
	if m.connection == nil {
		t.Error("Expected connection model to be initialized")
	}
	if m.transfer == nil {
		t.Error("Expected transfer model to be initialized")
	}
}

func TestConnectionModel(t *testing.T) {
	cm := NewConnectionModel(true, "test.txt", 1024)
	if cm == nil {
		t.Fatal("Expected connection model to be created")
	}
	if !cm.isOffer {
		t.Error("Expected isOffer to be true")
	}
	if cm.filename != "test.txt" {
		t.Errorf("Expected filename to be 'test.txt', got %s", cm.filename)
	}
	if cm.filesize != 1024 {
		t.Errorf("Expected filesize to be 1024, got %d", cm.filesize)
	}
}

func TestTransferModel(t *testing.T) {
	tm := NewTransferModel("test.txt", 1024)
	if tm == nil {
		t.Fatal("Expected transfer model to be created")
	}
	if tm.filename != "test.txt" {
		t.Errorf("Expected filename to be 'test.txt', got %s", tm.filename)
	}
	if tm.filesize != 1024 {
		t.Errorf("Expected filesize to be 1024, got %d", tm.filesize)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		expected string
		input    int64
	}{
		{"0 B", 0},
		{"1023 B", 1023},
		{"1.0 KB", 1024},
		{"1.5 KB", 1536},
		{"1.0 MB", 1048576},
		{"1.0 GB", 1073741824},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		expected string
		input    time.Duration
	}{
		{"00:30", 30 * time.Second},
		{"01:30", 90 * time.Second},
		{"01:00:00", 3600 * time.Second},
		{"01:01:01", 3661 * time.Second},
	}

	for _, tt := range tests {
		result := formatDuration(tt.input)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

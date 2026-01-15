/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TransferModel represents the transfer screen
type TransferModel struct {
	startTime   time.Time
	filename    string
	status      string
	progress    progress.Model
	eta         time.Duration
	filesize    int64
	transferred int64
	speed       float64
	width       int
	height      int
}

// NewTransferModel creates a new transfer screen model
func NewTransferModel(filename string, filesize int64) *TransferModel {
	p := progress.New(progress.WithDefaultGradient())
	p.Width = 60

	return &TransferModel{
		filename:  filename,
		filesize:  filesize,
		progress:  p,
		startTime: time.Now(),
		width:     80,
		height:    24,
		status:    "Transferring...",
	}
}

// Init initializes the transfer model
func (m *TransferModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the transfer screen
func (m *TransferModel) Update(msg tea.Msg) (TransferModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TransferProgressMsg:
		m.transferred = msg.BytesTransferred
		m.speed = msg.Speed
		m.eta = msg.ETA
		return *m, nil

	case TransferStatusMsg:
		m.status = msg.Status
		return *m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Update progress bar width
		if m.width > 20 {
			m.progress.Width = min(m.width-20, 60)
		}
		return *m, nil
	}

	return *m, nil
}

// View renders the transfer screen
func (m *TransferModel) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		MarginTop(1).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("📦 File Transfer in Progress"))
	b.WriteString("\n\n")

	// File info
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	b.WriteString(infoStyle.Render(fmt.Sprintf("File: %s", m.filename)))
	b.WriteString("\n")
	b.WriteString(infoStyle.Render(fmt.Sprintf("Size: %s", formatBytes(m.filesize))))
	b.WriteString("\n\n")

	// Progress bar
	var percent float64
	if m.filesize > 0 {
		percent = float64(m.transferred) / float64(m.filesize)
	}
	b.WriteString(m.progress.ViewAs(percent))
	b.WriteString("\n\n")

	// Stats
	statsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14"))

	b.WriteString(statsStyle.Render(fmt.Sprintf("Transferred: %s / %s (%.1f%%)",
		formatBytes(m.transferred),
		formatBytes(m.filesize),
		percent*100)))
	b.WriteString("\n")

	if m.speed > 0 {
		b.WriteString(statsStyle.Render(fmt.Sprintf("Speed: %s/s", formatBytes(int64(m.speed)))))
		b.WriteString("\n")
	}

	if m.eta > 0 {
		b.WriteString(statsStyle.Render(fmt.Sprintf("ETA: %s", formatDuration(m.eta))))
		b.WriteString("\n")
	}

	elapsed := time.Since(m.startTime)
	b.WriteString(statsStyle.Render(fmt.Sprintf("Elapsed: %s", formatDuration(elapsed))))
	b.WriteString("\n\n")

	// Status
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	b.WriteString(statusStyle.Render(m.status))

	return b.String()
}

// Messages

// TransferProgressMsg updates the transfer progress
type TransferProgressMsg struct {
	BytesTransferred int64
	Speed            float64
	ETA              time.Duration
}

// TransferStatusMsg updates the transfer status
type TransferStatusMsg struct {
	Status string
}

// Helper functions

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

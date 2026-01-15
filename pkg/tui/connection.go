/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConnectionModel represents the connection screen
type ConnectionModel struct {
	spinner  spinner.Model
	filename string
	localSDP string
	status   string
	filesize int64
	width    int
	height   int
	isOffer  bool
}

// NewConnectionModel creates a new connection screen model
func NewConnectionModel(isOffer bool, filename string, filesize int64) *ConnectionModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &ConnectionModel{
		isOffer:  isOffer,
		filename: filename,
		filesize: filesize,
		spinner:  s,
		status:   "Initializing...",
		width:    80,
		height:   24,
	}
}

// Init initializes the connection model
func (m *ConnectionModel) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles messages for the connection screen
func (m *ConnectionModel) Update(msg tea.Msg) (ConnectionModel, tea.Cmd) {
	switch msg := msg.(type) {
	case LocalSDPMsg:
		m.localSDP = msg.SDP
		m.status = "Waiting for remote signal..."
		return *m, nil

	case ConnectionStatusMsg:
		m.status = msg.Status
		return *m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return *m, cmd
	}

	return *m, nil
}

// View renders the connection screen
func (m *ConnectionModel) View() string {
	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		MarginTop(1).
		MarginBottom(1)

	b.WriteString(titleStyle.Render("🚀 HyperTunnel - P2P File Transfer"))
	b.WriteString("\n\n")

	// Mode and file info
	infoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("7"))

	mode := "Receiver"
	if m.isOffer {
		mode = "Sender"
	}
	b.WriteString(infoStyle.Render(fmt.Sprintf("Mode: %s", mode)))
	b.WriteString("\n")

	if m.filename != "" {
		fileInfo := fmt.Sprintf("File: %s", m.filename)
		if m.filesize > 0 {
			fileInfo += fmt.Sprintf(" (%s)", formatBytes(m.filesize))
		}
		b.WriteString(infoStyle.Render(fileInfo))
		b.WriteString("\n\n")
	} else {
		b.WriteString("\n")
	}

	// Status with spinner
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	b.WriteString(m.spinner.View())
	b.WriteString(" ")
	b.WriteString(statusStyle.Render(m.status))
	b.WriteString("\n\n")

	// Local SDP box
	if m.localSDP != "" {
		boxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(1, 2).
			Width(min(m.width-4, 78))

		labelStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("14"))

		sdpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))

		var sdpBox strings.Builder
		sdpBox.WriteString(labelStyle.Render("Your connection signal:"))
		sdpBox.WriteString("\n\n")
		sdpBox.WriteString(sdpStyle.Render(m.localSDP))

		b.WriteString(boxStyle.Render(sdpBox.String()))
		b.WriteString("\n\n")

		helpStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Italic(true)

		b.WriteString(helpStyle.Render("Copy the above signal and send it to the other peer."))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Then paste their signal when prompted."))
	}

	return b.String()
}

// Messages

// LocalSDPMsg contains the local SDP signal
type LocalSDPMsg struct {
	SDP string
}

// ConnectionStatusMsg updates the connection status
type ConnectionStatusMsg struct {
	Status string
}

// Helper functions

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

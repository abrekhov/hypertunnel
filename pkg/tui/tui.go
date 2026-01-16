/*
 *   Copyright (c) 2021 Anton Brekhov
 *   All rights reserved.
 */
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// State represents the current state of the TUI
type State int

const (
	StateConnection State = iota
	StateTransfer
	StateDone
	StateError
)

// Model is the main Bubble Tea model
type Model struct {
	connection *ConnectionModel
	transfer   *TransferModel
	err        error
	width      int
	height     int
	state      State
}

// NewModel creates a new TUI model
func NewModel(isOffer bool, filename string, filesize int64) Model {
	return Model{
		state:      StateConnection,
		connection: NewConnectionModel(isOffer, filename, filesize),
		transfer:   NewTransferModel(filename, filesize),
		width:      80,
		height:     24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.connection != nil {
			m.connection.width = msg.Width
			m.connection.height = msg.Height
		}
		if m.transfer != nil {
			m.transfer.width = msg.Width
			m.transfer.height = msg.Height
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case ConnectionReadyMsg:
		m.state = StateTransfer
		return m, m.transfer.Init()

	case TransferCompleteMsg:
		m.state = StateDone
		return m, tea.Quit

	case ErrorMsg:
		m.state = StateError
		m.err = msg.Err
		return m, tea.Quit
	}

	// Delegate to the appropriate state handler
	var cmd tea.Cmd
	switch m.state {
	case StateConnection:
		if m.connection != nil {
			*m.connection, cmd = m.connection.Update(msg)
		}
	case StateTransfer:
		if m.transfer != nil {
			*m.transfer, cmd = m.transfer.Update(msg)
		}
	case StateDone, StateError:
		// Terminal states, no further updates needed
	}

	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	switch m.state {
	case StateConnection:
		if m.connection != nil {
			return m.connection.View()
		}
	case StateTransfer:
		if m.transfer != nil {
			return m.transfer.View()
		}
	case StateDone:
		return m.doneView()
	case StateError:
		return m.errorView()
	}
	return ""
}

func (m Model) doneView() string {
	s := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")).
		MarginTop(1).
		MarginBottom(1)

	return s.Render("✓ Transfer complete!")
}

func (m Model) errorView() string {
	s := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("9")).
		MarginTop(1).
		MarginBottom(1)

	return s.Render(fmt.Sprintf("✗ Error: %v", m.err))
}

// Messages

// ConnectionReadyMsg is sent when the connection is established
type ConnectionReadyMsg struct{}

// TransferCompleteMsg is sent when the transfer is complete
type TransferCompleteMsg struct{}

// ErrorMsg is sent when an error occurs
type ErrorMsg struct {
	Err error
}

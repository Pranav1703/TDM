package tui

import (
	"fmt"
	"log"
	"os"
	"shareIt/internal/server"
	"shareIt/internal/utils"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// This message is specific to the TUI's initialization process.
type ProgramInitMsg struct {
	Program *tea.Program
}

// Define different focus states for our application panes.
const (
	peers_focus = iota
	uploads_focus
	downloads_focus
)

var (
	// Styling for focused and unfocused panes.
	focusedStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")) // A bright magenta
	unfocusedStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")) // A dim gray
)

// mainModel is the top-level model for our application.
type mainModel struct {
	peers        sectionModel
	uploads      sectionModel
	downloads    sectionModel
	input        textinput.Model
	focus        int // To track which pane is focused
	width        int
	height       int
	peerList     []string
	selectedPeer int               // Index of the currently selected peer
	transferLog  map[string]string // Map to store transfer progress strings
	program      *tea.Program      // To send messages from spawned goroutines
}

// sectionModel represents one of the three panes in the UI.
type sectionModel struct {
	title    string
	viewport viewport.Model
	focused  bool
}

func (m *mainModel) SetProgram(p *tea.Program) {
	m.program = p
}

// newSection creates a new section with a given title.
func newSection(title string) sectionModel {
	vp := viewport.New(0, 0)
	vp.SetContent("Scanning for peers...")
	return sectionModel{
		title:    title,
		viewport: vp,
	}
}

// setSize updates the size of the section's viewport.
func (m *sectionModel) setSize(w, h int) {
	style := m.getStyle()
	m.viewport.Width = w - style.GetHorizontalFrameSize()
	m.viewport.Height = h - style.GetVerticalFrameSize() - 1 // -1 for title
}

// getStyle returns the appropriate style based on the focus state.
func (m *sectionModel) getStyle() lipgloss.Style {
	if m.focused {
		return focusedStyle
	}
	return unfocusedStyle
}

// View renders the section, including its title and content.
func (m sectionModel) View() string {
	style := m.getStyle()
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 1)

	if m.focused {
		titleStyle = titleStyle.Background(lipgloss.Color("#6A5ACD"))
	} else {
		titleStyle = titleStyle.Background(lipgloss.Color("#555"))
	}

	title := titleStyle.Render(m.title)
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.viewport.View())
	return style.Render(content)
}

// InitialModel now returns a pointer to mainModel (*mainModel).
func InitialModel() *mainModel {
	ti := textinput.New()
	ti.Placeholder = "Enter file path to upload..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 20

	m := mainModel{
		peers:        newSection("PEERS"),
		uploads:      newSection("UPLOADS"),
		downloads:    newSection("DOWNLOADS"),
		input:        ti,
		focus:        uploads_focus,
		selectedPeer: 0,
		transferLog:  make(map[string]string),
	}
	m.uploads.focused = true
	m.uploads.viewport.SetContent("Enter a file path and press Enter to send to the selected peer.")
	m.downloads.viewport.SetContent("Waiting for incoming files...")
	return &m
}

// Init now uses a pointer receiver for consistency.
func (m *mainModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all incoming messages and updates the model accordingly.
func (m *mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// ProgramInitMsg case is no longer needed since we use SetProgram.

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		topRowHeight := m.height * 2 / 3
		bottomRowHeight := m.height - topRowHeight
		leftColWidth := m.width / 2
		rightColWidth := m.width - leftColWidth
		m.peers.setSize(leftColWidth, topRowHeight)
		m.uploads.setSize(rightColWidth, topRowHeight-1)
		m.downloads.setSize(m.width, bottomRowHeight)
		m.input.Width = rightColWidth - focusedStyle.GetHorizontalFrameSize() - 2

	case utils.PeersUpdatedMsg:
		m.peerList = msg.Peers
		if m.selectedPeer >= len(m.peerList) {
			m.selectedPeer = 0 // Reset selection if it's out of bounds
		}
		m.updatePeersView()

	case utils.FileTransferMsg:
		key := fmt.Sprintf("%s-%s", msg.Direction, msg.Filename)
		m.transferLog[key] = fmt.Sprintf("%s: %s %.2f%% (%s)", msg.Direction, msg.Filename, msg.Progress, msg.Rate)

		var uploads, downloads []string
		for k, v := range m.transferLog {
			if strings.HasPrefix(k, "Sending") {
				uploads = append(uploads, v)
			} else {
				downloads = append(downloads, v)
			}
		}
		m.uploads.viewport.SetContent(strings.Join(uploads, "\n"))
		m.downloads.viewport.SetContent(strings.Join(downloads, "\n"))

	case utils.LogMsg:
		log.Println("TUI Log:", msg.Message)

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		// Handle navigation in the peers list
		case "up", "k":
			if m.focus == peers_focus && len(m.peerList) > 0 {
				m.selectedPeer--
				if m.selectedPeer < 0 {
					m.selectedPeer = len(m.peerList) - 1
				}
				m.updatePeersView()
			}
		case "down", "j":
			if m.focus == peers_focus && len(m.peerList) > 0 {
				m.selectedPeer++
				if m.selectedPeer >= len(m.peerList) {
					m.selectedPeer = 0
				}
				m.updatePeersView()
			}

		case "tab":
			m.focus = (m.focus + 1) % 3
			m.peers.focused = m.focus == peers_focus
			m.uploads.focused = m.focus == uploads_focus
			m.downloads.focused = m.focus == downloads_focus
			if m.focus == uploads_focus {
				m.input.Focus()
			} else {
				m.input.Blur()
			}
			return m, nil

		case "enter":
			if m.focus == uploads_focus {
				filePath := m.input.Value()
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					log.Printf("File does not exist: %s", filePath)
					return m, nil
				}

				if len(m.peerList) > 0 && m.selectedPeer < len(m.peerList) {
					// Send the file to the currently selected peer.
					peerAddr := m.peerList[m.selectedPeer]
					log.Printf("Initiating send of %s to %s", filePath, peerAddr)
					if m.program != nil {
						go server.SendFile(filePath, peerAddr, m.program)
					} else {
						log.Println("TUI program not initialized, cannot send file.")
					}
				} else {
					log.Println("No peers found to send file to.")
				}
				m.input.Reset()
			}
		}
	}

	// Update the focused component
	switch m.focus {
	case peers_focus:
		m.peers.viewport, cmd = m.peers.viewport.Update(msg)
		cmds = append(cmds, cmd)
	case uploads_focus:
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
		m.uploads.viewport, cmd = m.uploads.viewport.Update(msg)
		cmds = append(cmds, cmd)
	case downloads_focus:
		m.downloads.viewport, cmd = m.downloads.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updatePeersView is a helper function to render the list of peers with a selection indicator.
func (m *mainModel) updatePeersView() {
	if len(m.peerList) > 0 {
		var s strings.Builder
		selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)

		for i, peer := range m.peerList {
			if i == m.selectedPeer {
				s.WriteString(selectedStyle.Render("> "+peer) + "\n")
			} else {
				s.WriteString("  " + peer + "\n")
			}
		}
		m.peers.viewport.SetContent(s.String())
	} else {
		m.selectedPeer = 0
		m.peers.viewport.SetContent("Scanning for peers...")
	}
}

// View now uses a pointer receiver for consistency.
func (m *mainModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	uploadsWithInput := lipgloss.JoinVertical(
		lipgloss.Left,
		m.uploads.View(),
		m.input.View(),
	)

	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.peers.View(),
		uploadsWithInput,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		m.downloads.View(),
	)
}


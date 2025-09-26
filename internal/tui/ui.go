package tui

import (

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
	peers     sectionModel
	uploads   sectionModel
	downloads sectionModel
	input     textinput.Model
	focus     int // To track which pane is focused
	width     int
	height    int
}

// sectionModel represents one of the three panes in the UI.
type sectionModel struct {
	title    string
	viewport viewport.Model
	focused  bool
}

// newSection creates a new section with a given title.
func newSection(title string) sectionModel {
	vp := viewport.New(0, 0)
	vp.SetContent("No content yet.")
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

	// Render the title bar
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

	// Join the title and the viewport content
	content := lipgloss.JoinVertical(lipgloss.Left, title, m.viewport.View())

	return style.Render(content)
}

// initialModel sets up the initial state of our application.
func initialModel() mainModel {
	// Create the text input component
	ti := textinput.New()
	ti.Placeholder = "Enter file path to upload..."
	ti.Focus() // Start with the input focused
	ti.CharLimit = 256
	ti.Width = 20

	m := mainModel{
		peers:     newSection("PEERS"),
		uploads:   newSection("UPLOADS"),
		downloads: newSection("DOWNLOADS"),
		input:     ti,
		focus:     uploads_focus, // Start focus on the uploads pane
	}
	m.uploads.focused = true
	return m
}

// Init is the first command that is run when the program starts.
func (m mainModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles all incoming messages and updates the model accordingly.
func (m mainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// Handle key presses
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit

		case "tab":
			// Cycle focus between the panes
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
				// Add the input value to the uploads content
				newContent := m.uploads.viewport.View() + "\n" + m.input.Value()
				if m.uploads.viewport.View() == "No content yet." {
					newContent = m.input.Value()
				}
				m.uploads.viewport.SetContent(newContent)
				m.input.Reset()
				m.uploads.viewport.GotoBottom()
			}
		}

	// Handle window resize events
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate heights for the sections
		// Top row gets 2/3 of the height, bottom gets 1/3
		topRowHeight := m.height * 2 / 3
		bottomRowHeight := m.height - topRowHeight

		// Calculate widths for the top row sections
		leftColWidth := m.width / 2
		rightColWidth := m.width - leftColWidth

		// Set the size for each section
		// Subtract 1 from height for the input bar in the uploads section
		m.peers.setSize(leftColWidth, topRowHeight)
		m.uploads.setSize(rightColWidth, topRowHeight-1)
		m.downloads.setSize(m.width, bottomRowHeight)
		m.input.Width = rightColWidth - focusedStyle.GetHorizontalFrameSize() - 2
	}

	// Update the focused component
	switch m.focus {
	case peers_focus:
		m.peers.viewport, cmd = m.peers.viewport.Update(msg)
		cmds = append(cmds, cmd)
	case uploads_focus:
		// Also update the text input
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

// View renders the entire UI.
func (m mainModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	// Combine the uploads view with the text input
	uploadsWithInput := lipgloss.JoinVertical(
		lipgloss.Left,
		m.uploads.View(),
		m.input.View(),
	)

	// Join the top two sections horizontally
	topRow := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.peers.View(),
		uploadsWithInput,
	)

	// Stack the top row and the bottom section vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		topRow,
		m.downloads.View(),
	)
}

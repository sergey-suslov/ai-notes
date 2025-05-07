package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/sergey-suslov/ai-notes/store"
	"github.com/sergey-suslov/ai-notes/util"
)

// notesModel lets the user browse and select notes to inject.
// notesModel lets the user browse and select notes to inject or view.
type notesModel struct {
	notes    []*store.Note
	cursor   int
	selected *store.Note
	action   string // "inject" or "view"
}

// viewModel displays a single note in read-only mode.
// viewExitMsg signals exiting the note view.
type (
	viewExitMsg struct{}
	// viewModel holds the note to display.
	viewModel struct {
		note     *store.Note
		viewport viewport.Model
		ws       tea.WindowSizeMsg
	}
)

// newViewModel creates a viewModel for the given note.
func newViewModel(note *store.Note, initialWindopwSize tea.WindowSizeMsg) viewModel {
	vp := viewport.New(initialWindopwSize.Width-2, initialWindopwSize.Height-4)
	vp.YPosition = 0
	vp.MouseWheelEnabled = true

	vp.SetContent(note.Body)
	return viewModel{note: note, viewport: vp, ws: initialWindopwSize}
}

// Init does nothing for viewModel.
func (m viewModel) Init() tea.Cmd { return nil }

// Update exits view on any key press.
func (m viewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ws = msg
		m.viewport.Width = util.Max(0, msg.Width-2)
		m.viewport.Height = util.Max(0, msg.Height-4)

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC:
			return m, func() tea.Msg { return viewExitMsg{} }
		}
	}
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	return m, vpCmd
}

// View renders the note title and body.
func (m viewModel) View() string {
	// m.ws = tea.WindowSizeMsg{Width: 171, Height: 20}
	width := util.Max(0, util.Min(int(180), m.ws.Width-2))

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)
	var b strings.Builder

	f, _ := r.Render(m.note.Body)

	// Update viewport dimensions based on window size
	m.viewport.Width = util.Max(0, m.ws.Width-2)
	m.viewport.Height = util.Max(0, m.ws.Height-4)
	m.viewport.SetContent(f)

	b.WriteString("Viewing Note: " + m.note.Title + "\n\n")
	b.WriteString(m.viewport.View() + "\n\n")
	b.WriteString("Press any key to return...")
	return b.String()
}

// newNotesModel constructs a notesModel from stored notes.
// newNotesModel constructs a notesModel from stored notes.
func newNotesModel(notes []*store.Note) *notesModel {
	return &notesModel{notes: notes, action: ""}
}

// Init is required by Bubble Tea; no initial command.
func (m *notesModel) Init() tea.Cmd {
	return nil
}

// Update handles navigation and selection.
func (m *notesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.notes)-1 {
				m.cursor++
			}
		case tea.KeyRunes:
			// handle 'a' for inject
			if len(msg.Runes) > 0 && msg.Runes[0] == 'a' {
				m.action = "inject"
				m.selected = m.notes[m.cursor]
				return m, tea.Quit
			}
		case tea.KeyEnter:
			// view note
			m.action = "view"
			m.selected = m.notes[m.cursor]
			return m, tea.Quit
		case tea.KeyEsc, tea.KeyCtrlC:
			// cancel and return to chat
			m.selected = nil
			m.action = ""
			return m, nil
		}
	}
	return m, nil
}

// View renders the list of notes.
func (m *notesModel) View() string {
	var b strings.Builder
	b.WriteString("Select a note (↑/↓, Enter to view, a to inject, esc to cancel):\n\n")
	for i, note := range m.notes {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf("%s %s (%s)\n", cursor, note.Title, note.CreatedAt.Format("2006-01-02 15:04:05")))
	}
	return b.String()
}


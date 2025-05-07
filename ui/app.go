package ui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	openaiclient "github.com/sergey-suslov/ai-notes/openai"
	"github.com/sergey-suslov/ai-notes/store"
)

// screen identifiers
const (
	screenSelect = iota
	screenChat
	screenNotes
	screenView
)

// AppModel is the top-level Bubble Tea model managing multiple screens.
type AppModel struct {
	client    *openaiclient.Client
	sessions  []*store.Session
	selection *selectionModel

	// active session and chat state
	session *store.Session
	chat    model

	// notes browser and note viewer
	notes *notesModel
	view  *viewModel

	screen int

	windowSize tea.WindowSizeMsg
}

// NewAppModel creates the application model with loaded sessions.
func NewAppModel(client *openaiclient.Client, sessions []*store.Session) *AppModel {
	return &AppModel{
		client:    client,
		sessions:  sessions,
		selection: newSelectionModel(sessions),
		screen:    screenSelect,
	}
}

// Init does nothing; focus is managed by sub-models.
func (m *AppModel) Init() tea.Cmd {
	return nil
}

// Update dispatches messages to the current screen's model.
func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case screenSelect:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.windowSize = msg
		}
		// delegate to selectionModel
		newSel, cmd := m.selection.Update(msg)
		m.selection = newSel.(*selectionModel)
		// if a session was picked, move to chat
		if m.selection.selectedSession != nil {
			m.session = m.selection.selectedSession
			m.chat = NewModel(m.client, m.session, m.windowSize)
			m.screen = screenChat
			return m, m.chat.Init()
		}
		// allow quitting
		if k, ok := msg.(tea.KeyMsg); ok && (k.Type == tea.KeyCtrlC || k.Type == tea.KeyEsc) {
			return m, tea.Quit
		}
		return m, cmd

	case screenChat:
		// global keybindings
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.Type {
			case tea.KeyCtrlL:
				notes, err := store.LoadNotes()
				if err != nil {
					m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error loading notes: " + err.Error()})
					return m, nil
				}
				m.notes = newNotesModel(notes)
				m.screen = screenNotes
				return m, nil
			case tea.KeyCtrlC, tea.KeyEsc:
				return m, tea.Quit
			}
		}
		newChat, cmd := m.chat.Update(msg)
		m.chat = newChat.(model)
		return m, cmd

	case screenNotes:
		// browse notes
		newNotes, cmd := m.notes.Update(msg)
		m.notes = newNotes.(*notesModel)
		// if a note was selected
		if sel := m.notes.selected; sel != nil {
			switch m.notes.action {
			case "inject":
				m.session.Chat = append(m.session.Chat, store.Message{Role: "system", Content: sel.Body})
				m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: fmt.Sprintf("Injected notes: %s", sel.Title)})
				m.screen = screenChat
				m.notes = nil
				return m, nil
			case "view":
				vm := newViewModel(sel, m.windowSize)
				m.view = &vm
				m.screen = screenView
				return m, nil
			}
		}
		// exit notes view
		if k, ok := msg.(tea.KeyMsg); ok && (k.Type == tea.KeyCtrlC || k.Type == tea.KeyEsc) {
			m.screen = screenChat
			return m, nil
		}
		return m, cmd

	case screenView:
		// note view
		// delegate to viewModel
		newViewModel, cmd := m.view.Update(msg)
		// update viewModel pointer
		vm := newViewModel.(viewModel)
		m.view = &vm
		// exit view on custom message
		if _, ok := msg.(viewExitMsg); ok {
			m.screen = screenNotes
			m.notes.action = ""
			return m, nil
		}
		return m, cmd
	}
	return m, nil
}

// View renders the UI for the current screen.
func (m *AppModel) View() string {
	switch m.screen {
	case screenSelect:
		return m.selection.View()
	case screenChat:
		return m.chat.View()
	case screenNotes:
		return m.notes.View()
	case screenView:
		return m.view.View()
	default:
		return ""
	}
}

// Run initializes everything and starts the Bubble Tea program.
func Run() error {
	client, err := openaiclient.NewClient()
	if err != nil {
		return fmt.Errorf("creating OpenAI client: %w", err)
	}
	sessions, err := store.LoadSessions()
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}
	app := NewAppModel(client, sessions)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err = p.Run()
	// save the session if one was active
	if app.session != nil {
		if serr := app.session.Save(); serr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save session: %v\n", serr)
		}
	}
	return err
}

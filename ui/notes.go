package ui

import (
   "fmt"
   "strings"

   tea "github.com/charmbracelet/bubbletea"
   "github.com/sergey-suslov/ai-notes/store"
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
type viewModel struct {
   note *store.Note
}

// newViewModel creates a viewModel for the given note.
func newViewModel(note *store.Note) viewModel {
   return viewModel{note: note}
}

// Init does nothing for viewModel.
func (m viewModel) Init() tea.Cmd { return nil }

// Update exits on any key press.
func (m viewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
   if _, ok := msg.(tea.KeyMsg); ok {
       return m, tea.Quit
   }
   return m, nil
}

// View renders the note title and body.
func (m viewModel) View() string {
   var b strings.Builder
   b.WriteString("Viewing Note: " + m.note.Title + "\n\n")
   b.WriteString(m.note.Body + "\n\n")
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
           // cancel
           m.selected = nil
           return m, tea.Quit
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
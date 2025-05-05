package ui

import (
   "fmt"
   "strings"

   tea "github.com/charmbracelet/bubbletea"
   "github.com/sergey-suslov/ai-notes/store"
)

// notesModel lets the user browse and select notes to inject.
type notesModel struct {
   notes    []*store.Note
   cursor   int
   selected *store.Note
}

// newNotesModel constructs a notesModel from stored notes.
func newNotesModel(notes []*store.Note) *notesModel {
   return &notesModel{notes: notes}
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
           // handle 'a' for add/inject
           if len(msg.Runes) > 0 && msg.Runes[0] == 'a' {
               m.selected = m.notes[m.cursor]
               return m, tea.Quit
           }
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
   b.WriteString("Select a note to inject (↑/↓, a to inject, esc to cancel):\n\n")
   for i, note := range m.notes {
       cursor := " "
       if m.cursor == i {
           cursor = ">"
       }
       b.WriteString(fmt.Sprintf("%s %s (%s)\n", cursor, note.Title, note.CreatedAt.Format("2006-01-02 15:04:05")))
   }
   return b.String()
}
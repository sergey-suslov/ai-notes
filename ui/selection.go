package ui

import (
   "fmt"
   "strings"

   tea "github.com/charmbracelet/bubbletea"
   "github.com/sergey-suslov/ai-notes/store"
)

// selectionModel handles choosing between new or existing sessions.
type selectionModel struct {
   sessions        []*store.Session
   cursor          int
   selectedSession *store.Session
}

// newSelectionModel constructs a selection model with existing sessions.
func newSelectionModel(sessions []*store.Session) *selectionModel {
   return &selectionModel{sessions: sessions}
}

// Init does nothing.
func (m *selectionModel) Init() tea.Cmd {
   return nil
}

// Update handles up/down navigation and selection.
func (m *selectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
   switch msg := msg.(type) {
   case tea.KeyMsg:
       switch msg.Type {
       case tea.KeyRunes:
           // delete session
           if len(msg.Runes) > 0 && msg.Runes[0] == 'd' {
               // only delete existing sessions
               if m.cursor > 0 && m.cursor <= len(m.sessions) {
                   idx := m.cursor - 1
                   sess := m.sessions[idx]
                   // remove file
                   _ = sess.Delete()
                   // remove from list
                   m.sessions = append(m.sessions[:idx], m.sessions[idx+1:]...)
                   // adjust cursor
                   if m.cursor > len(m.sessions) {
                       m.cursor = len(m.sessions)
                   }
               }
               return m, nil
           }
       case tea.KeyUp:
           if m.cursor > 0 {
               m.cursor--
           }
       case tea.KeyDown:
           if m.cursor < len(m.sessions) {
               m.cursor++
           }
       case tea.KeyEnter:
           if m.cursor == 0 {
               // New session
               m.selectedSession = store.NewSession()
           } else {
               // Resume existing session
               m.selectedSession = m.sessions[m.cursor-1]
           }
           return m, tea.Quit
       }
   }
   return m, nil
}

// View renders the menu of sessions.
func (m *selectionModel) View() string {
   var b strings.Builder
   b.WriteString("Select a session (↑/↓, Enter to select, d to delete, esc to cancel):\n\n")
   // Option 0: new session
   cursor := " "
   if m.cursor == 0 {
       cursor = ">"
   }
   b.WriteString(fmt.Sprintf("%s New Session\n", cursor))
   // Existing sessions
   for i, s := range m.sessions {
       prefix := " "
       if m.cursor == i+1 {
           prefix = ">"
       }
       b.WriteString(fmt.Sprintf("%s %s (%s)\n", prefix, s.ID, s.CreatedAt.Format("2006-01-02 15:04:05")))
   }
   return b.String()
}
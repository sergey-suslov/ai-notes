package ui

import (
   "context"
   "fmt"
   "os"
   "strings"

   "github.com/charmbracelet/bubbles/textinput"
   tea "github.com/charmbracelet/bubbletea"
   goopenai "github.com/sashabaranov/go-openai"

   openaiclient "github.com/sergey-suslov/ai-notes/openai"
   "github.com/sergey-suslov/ai-notes/store"
)

// model holds the state for the chat UI.
type model struct {
   client  *openaiclient.Client
   session *store.Session
   input   textinput.Model
}

// aiMsg wraps the AI's response content.
type aiMsg string

// errMsg wraps errors from async commands.
type errMsg struct{ err error }

// NewModel initializes the TUI model with  client and session.
func NewModel(client *openaiclient.Client, session *store.Session) model {
   ti := textinput.New()
   ti.Placeholder = "Type a message"
   ti.Focus()
   ti.CharLimit = 256
   ti.Width = 50

   // If this is a new session (no prior messages), add a welcome prompt
   if len(session.Chat) == 0 {
       session.Chat = append(session.Chat, store.Message{Role: "assistant", Content: "Welcome to AI Notes!"})
   }
   return model{client: client, session: session, input: ti}
}

// Init runs any initial IO; we only need blinking cursor.
func (m model) Init() tea.Cmd {
   return textinput.Blink
}

// Update handles key presses and async messages.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
   switch msg := msg.(type) {
   case aiMsg:
       // append AI reply
       m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: string(msg)})
       return m, nil
   case errMsg:
       m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error: " + msg.err.Error()})
       return m, nil
   case tea.KeyMsg:
       switch msg.Type {
       case tea.KeyCtrlC, tea.KeyEsc:
           return m, tea.Quit
       case tea.KeyEnter:
           userInput := m.input.Value()
           if strings.TrimSpace(userInput) == "" {
               return m, nil
           }
           // record user message
           m.session.Chat = append(m.session.Chat, store.Message{Role: "user", Content: userInput})
           m.input.Reset()
           // call AI
           return m, m.getCompletionCmd()
       }
   }
   // update input field
   var cmd tea.Cmd
   m.input, cmd = m.input.Update(msg)
   return m, cmd
}

// View renders the chat history and the input field.
func (m model) View() string {
   var b strings.Builder
   for _, msg := range m.session.Chat {
       var prefix string
       switch msg.Role {
       case "user":
           prefix = "You: "
       case "assistant":
           prefix = "AI: "
       default:
           prefix = msg.Role + ": "
       }
       b.WriteString(prefix + msg.Content + "\n")
   }
   b.WriteString("\n" + m.input.View())
   return b.String()
}

// getCompletionCmd builds a tea.Cmd that queries the OpenAI API with the full session context.
func (m model) getCompletionCmd() tea.Cmd {
   return func() tea.Msg {
       ctx := context.Background()
       // convert stored chat to openai messages
       msgs := make([]goopenai.ChatCompletionMessage, len(m.session.Chat))
       for i, cm := range m.session.Chat {
           var role string
           if cm.Role == "assistant" {
               role = goopenai.ChatMessageRoleAssistant
           } else {
               role = goopenai.ChatMessageRoleUser
           }
           msgs[i] = goopenai.ChatCompletionMessage{Role: role, Content: cm.Content}
       }
       resp, err := m.client.ChatCompletion(ctx, msgs, "gpt-3.5-turbo")
       if err != nil {
           return errMsg{err}
       }
       return aiMsg(resp)
   }
}

// Run starts the session selection, then chat TUI, and saves the session on exit.
func Run() error {
   client, err := openaiclient.NewClient()
   if err != nil {
       return fmt.Errorf("creating OpenAI client: %w", err)
   }

   // Load existing sessions
   sessions, err := store.LoadSessions()
   if err != nil {
       return fmt.Errorf("loading sessions: %w", err)
   }

   // Selection TUI: choose new or existing session
   selModel := newSelectionModel(sessions)
   p1 := tea.NewProgram(selModel)
   m1, err := p1.Run()
   if err != nil {
       return err
   }
   // Extract selected session
   sm, ok := m1.(*selectionModel)
   if !ok || sm.selectedSession == nil {
       return fmt.Errorf("no session selected")
   }
   session := sm.selectedSession

   // Launch chat UI
   p2 := tea.NewProgram(NewModel(client, session))
   _, err = p2.Run()
   // always attempt to save session
   if saveErr := session.Save(); saveErr != nil {
       fmt.Fprintf(os.Stderr, "warning: failed to save session: %v\n", saveErr)
   }
   return err
}
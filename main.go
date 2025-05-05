package main

import (
   "context"
   "fmt"
   "os"
   "strings"

   tea "github.com/charmbracelet/bubbletea"
   "github.com/charmbracelet/bubbles/textinput"

   goopenai "github.com/sashabaranov/go-openai"
   openaiclient "github.com/sergey-suslov/ai-notes/openai"
)

// model defines the TUI state
type model struct {
   client      *openaiclient.Client
   input       textinput.Model
   chatHistory []string
}

// custom message types for async results
type aiMsg string
type errMsg struct{ err error }

// initialModel returns an initial model with a focused input, welcome message, and OpenAI client
func initialModel(client *openaiclient.Client) model {
	ti := textinput.New()
	ti.Placeholder = "Type a message"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50

   return model{
       client:      client,
       input:       ti,
       chatHistory: []string{"Welcome to AI Notes!"},
   }
}

// Init is called when the program starts
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles incoming messages (keys, AI responses, etc.)
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
   switch msg := msg.(type) {
   case aiMsg:
       m.chatHistory = append(m.chatHistory, "AI: "+string(msg))
       return m, nil
   case errMsg:
       m.chatHistory = append(m.chatHistory, "Error: "+msg.err.Error())
       return m, nil
   case tea.KeyMsg:
       switch msg.Type {
       case tea.KeyCtrlC, tea.KeyEsc:
           return m, tea.Quit
       case tea.KeyEnter:
           input := m.input.Value()
           if strings.TrimSpace(input) == "" {
               return m, nil
           }
           m.chatHistory = append(m.chatHistory, "You: "+input)
           messages := []goopenai.ChatCompletionMessage{
               {Role: goopenai.ChatMessageRoleUser, Content: input},
           }
           m.input.Reset()
           return m, getCompletionCmd(m.client, messages)
       }
   }

   // Let the text input component handle all other messages
   var cmd tea.Cmd
   m.input, cmd = m.input.Update(msg)
   return m, cmd
}

// View renders the UI
func (m model) View() string {
	chat := strings.Join(m.chatHistory, "\n")
	return fmt.Sprintf("%s\n\n%s", chat, m.input.View())
}

// getCompletionCmd returns a command that calls the OpenAI API asynchronously.
func getCompletionCmd(client *openaiclient.Client, messages []goopenai.ChatCompletionMessage) tea.Cmd {
   return func() tea.Msg {
       ctx := context.Background()
       resp, err := client.ChatCompletion(ctx, messages, "gpt-3.5-turbo")
       if err != nil {
           return errMsg{err}
       }
       return aiMsg(resp)
   }
}

func main() {
   client, err := openaiclient.NewClient()
   if err != nil {
       fmt.Fprintf(os.Stderr, "Error creating OpenAI client: %v\n", err)
       os.Exit(1)
   }
   p := tea.NewProgram(initialModel(client))
   if _, err := p.Run(); err != nil {
       fmt.Fprintf(os.Stderr, "Error: %v\n", err)
       os.Exit(1)
   }
}


package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	goopenai "github.com/sashabaranov/go-openai"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/muesli/reflow/wordwrap"
	openaiclient "github.com/sergey-suslov/ai-notes/openai"
	"github.com/sergey-suslov/ai-notes/store"
)

// model holds the state for the chat UI.
type model struct {
	client   *openaiclient.Client
	session  *store.Session
	input    textinput.Model
	viewport viewport.Model

	windowSize tea.WindowSizeMsg
}

// aiMsg wraps the AI's response content.
type aiMsg string

// errMsg wraps errors from async commands.
// errMsg wraps errors from async commands.
type (
	errMsg struct{ err error }
	// noteMsg wraps generated note summary and file path.
	noteMsg struct{ Summary, Path string }
	// noteErr wraps errors from note generation or saving.
	noteErr struct{ err error }
)

// NewModel initializes the TUI model with  client and session.
func NewModel(client *openaiclient.Client, session *store.Session, initialWindopwSize tea.WindowSizeMsg) model {
	ti := textinput.New()
	ti.Placeholder = "Type a message"
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = initialWindopwSize.Width - 2

	// If this is a new session (no prior messages), add a welcome prompt
	if len(session.Chat) == 0 {
		session.Chat = append(session.Chat, store.Message{Role: "assistant", Content: "Welcome to AI Notes!"})
	}
	vp := viewport.New(initialWindopwSize.Width-2, initialWindopwSize.Height-2)
	vp.YPosition = 0
	vp.MouseWheelEnabled = true

	return model{
		client: client, session: session, input: ti,
		viewport:   vp,
		windowSize: initialWindopwSize,
	}
}

// Init runs any initial IO; we only need blinking cursor.
func (m model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles key presses and async messages.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.viewport.YPosition = 0
		// set viewport content width to terminal width minus border padding
		m.viewport.Width = msg.Width - 2
		// expand viewport to full screen height, accounting for border overhead
		m.viewport.Height = msg.Height - 2

	case noteMsg:
		// append summary to chat
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: msg.Summary})
		// inform about saved file
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: fmt.Sprintf("Notes saved to %s", msg.Path)})
		return m, nil
	case noteErr:
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error generating notes: " + msg.err.Error()})
		return m, nil
	case aiMsg:
		// append AI reply
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: string(msg)})
		return m, nil
	case errMsg:
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error: " + msg.err.Error()})
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlL:
			// list and inject notes
			notes, err := store.LoadNotes()
			if err != nil {
				m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error loading notes: " + err.Error()})
				return m, nil
			}
			nm := newNotesModel(notes)
			m2, err := tea.NewProgram(nm).Run()
			if err != nil {
				m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error opening notes browser: " + err.Error()})
				return m, nil
			}
			sel, ok := m2.(*notesModel)
			if ok && sel.selected != nil {
				switch sel.action {
				case "inject":
					// inject selected note into chat as a system message
					m.session.Chat = append(m.session.Chat, store.Message{Role: "system", Content: sel.selected.Body})
					m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: fmt.Sprintf("Injected notes: %s", sel.selected.Title)})
				case "view":
					// view selected note in a modal
					if _, err := tea.NewProgram(newViewModel(sel.selected)).Run(); err != nil {
						m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error viewing note: " + err.Error()})
					}
				}
			}
			return m, nil
		case tea.KeyCtrlN:
			// trigger note generation
			m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Generating notes..."})
			return m, m.getNotesCmd()
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
    // let viewport handle scrolling and other viewport-related events
    var vpCmd tea.Cmd
    m.viewport, vpCmd = m.viewport.Update(msg)
    // update input field
    var inputCmd tea.Cmd
    m.input, inputCmd = m.input.Update(msg)
    // combine viewport and input commands
    return m, tea.Batch(vpCmd, inputCmd)
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
	// wrap content to viewport width to prevent horizontal overflow
	wrapped := wordwrap.String(b.String(), m.viewport.Width)
	m.viewport.SetContent(wrapped)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Render(m.viewport.View())
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
		resp, err := m.client.ChatCompletion(ctx, msgs, "gpt-4o")
		if err != nil {
			return errMsg{err}
		}
		return aiMsg(resp)
	}
}

// getNotesCmd builds a tea.Cmd that generates bullet-point notes and saves them.
func (m model) getNotesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// start with a system prompt
		sys := goopenai.ChatCompletionMessage{Role: goopenai.ChatMessageRoleSystem, Content: "Please summarize the following conversation into concise bullet-point notes."}
		msgs := make([]goopenai.ChatCompletionMessage, len(m.session.Chat)+1)
		msgs[0] = sys
		for i, cm := range m.session.Chat {
			role := goopenai.ChatMessageRoleUser
			if cm.Role == "assistant" {
				role = goopenai.ChatMessageRoleAssistant
			}
			msgs[i+1] = goopenai.ChatCompletionMessage{Role: role, Content: cm.Content}
		}
		// get summary
		summary, err := m.client.ChatCompletion(ctx, msgs, "gpt-4o")
		if err != nil {
			return noteErr{err}
		}
		// save note
		note := store.NewNote(m.session.ID, summary)
		path, err := note.Save()
		if err != nil {
			return noteErr{err}
		}
		return noteMsg{Summary: summary, Path: path}
	}
}

// Run is provided by the app package (ui/app.go).
// Use ui/app.go's Run when starting the application.

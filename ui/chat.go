package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	goopenai "github.com/sashabaranov/go-openai"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/muesli/reflow/wordwrap"
	openaiclient "github.com/sergey-suslov/ai-notes/openai"
	"github.com/sergey-suslov/ai-notes/store"
	"github.com/sergey-suslov/ai-notes/util"
)

var BodyStyle = lipgloss.NewStyle().Margin(1, 2)

// model holds the state for the chat UI.
type model struct {
	client   *openaiclient.Client
	session  *store.Session
	input    textarea.Model
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

// NewModel initializes the TUI model with  client and session
func NewModel(client *openaiclient.Client, session *store.Session, initialWindopwSize tea.WindowSizeMsg) model {
	ti := textarea.New()
	ti.Placeholder = "Type a message"
	ti.Focus()
	ti.CharLimit = 1000
	ti.SetWidth(initialWindopwSize.Width - 2)

	// If this is a new session (no prior messages), add a welcome prompt
	if len(session.Chat) == 0 {
		session.Chat = append(session.Chat, store.Message{Role: "assistant", Content: "Welcome to AI Notes!"})
	}
	vp := viewport.New(initialWindopwSize.Width-2, initialWindopwSize.Height-4)
	vp.YPosition = 0
	vp.MouseWheelEnabled = true

	m := model{
		client: client, session: session, input: ti,
		viewport:   vp,
		windowSize: initialWindopwSize,
	}
	wrapped := m.getChatString()
	vp.SetContent(wrapped)
	vp.GotoBottom()
	m.viewport = vp

	return m
}

func (m *model) defaultBodyMargin() (int, int) { //nolint:exhaustive
	return BodyStyle.GetFrameSize()
}

func (m *model) getChatString() string {
	userStyle := lipgloss.NewStyle().Bold(true).Padding(1, 1).Margin(1, 2).Background(lipgloss.Color("#105fa8"))
	aiStyle := lipgloss.NewStyle().Bold(false).Margin(1, 0).Border(lipgloss.NormalBorder(), true, false)
	_, v := m.defaultBodyMargin()

	width := util.Max(0, util.Min(int(180), m.viewport.Width-v*2))

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width),
	)

	var b strings.Builder
	for _, msg := range m.session.Chat {
		// var prefix string
		wrapped := wordwrap.String(msg.Content, m.viewport.Width-6)
		switch msg.Role {
		case "user":
			b.WriteString(userStyle.Render(wrapped))
		case "assistant":
			content, _ := r.Render(msg.Content)
			b.WriteString(aiStyle.Render(content))
		default:
			b.WriteString(aiStyle.Render(wrapped))
		}
		// b.WriteString(prefixStyle.Render(prefix) + messageStyle.Render(msg.Content+"\n"))
	}
	return b.String()
}

// Init runs any initial IO; we only need blinking cursor.
func (m model) Init() tea.Cmd {
	return textarea.Blink
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
		m.viewport.Height = msg.Height - 4

	case noteMsg:
		// append summary to chat
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: msg.Summary})
		// inform about saved file
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: fmt.Sprintf("Notes saved to %s", msg.Path)})
		m.viewport.SetContent(m.getChatString())
		m.viewport.GotoBottom()

		return m, nil
	case noteErr:
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error generating notes: " + msg.err.Error()})
		return m, nil
	case aiMsg:
		// append AI reply
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: string(msg)})
		m.viewport.SetContent(m.getChatString())
		m.viewport.GotoBottom()
		return m, nil
	case errMsg:
		m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Error: " + msg.err.Error()})
		return m, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlN:
			// trigger note generation
			m.session.Chat = append(m.session.Chat, store.Message{Role: "assistant", Content: "Generating notes..."})
			m.viewport.SetContent(m.getChatString())
			m.viewport.GotoBottom()

			return m, m.getNotesCmd()
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyCtrlS:
			userInput := m.input.Value()
			if strings.TrimSpace(userInput) == "" {
				return m, nil
			}
			// record user message
			m.session.Chat = append(m.session.Chat, store.Message{Role: "user", Content: userInput})
			m.input.Reset()
			m.viewport.SetContent(m.getChatString())
			m.viewport.GotoBottom()

			// call AI
			return m, m.getCompletionCmd()
		}
	}
	// let viewport handle scrolling and other viewport-related events
	// b.WriteString("\n" + m.input.View())
	// wrap content to viewport width to prevent horizontal overflow
	m.viewport.SetContent(m.getChatString())

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
	chat := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		Render(m.viewport.View())
	b.WriteString(chat)
	b.WriteString("\n")
	b.WriteString(m.input.View())
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
		resp, err := m.client.ChatCompletion(ctx, msgs, "gpt-4o-mini")
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
		summary, err := m.client.ChatCompletion(ctx, msgs, "gpt-4o-mini")
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

package store

import (
   "encoding/json"
   "fmt"
   "os"
   "path/filepath"
   "sort"
   "time"
)

const (
   baseDirName     = ".ai-notes"
   sessionsDirName = "sessions"
)

// Message represents a single chat message with role (user or assistant) and content.
type Message struct {
   Role    string `json:"role"`
   Content string `json:"content"`
}

// Session holds the metadata and chat history for a conversation.
type Session struct {
   ID        string    `json:"id"`
   CreatedAt time.Time `json:"created_at"`
   Chat      []Message `json:"chat"`
}

// NewSession creates a new session with a time-based ID and current timestamp.
func NewSession() *Session {
   now := time.Now()
   id := now.Format("20060102T150405")
   return &Session{
       ID:        id,
       CreatedAt: now,
       Chat:      make([]Message, 0),
   }
}

// Save writes the session as JSON to ~/.ai-notes/sessions/{ID}.json.
func (s *Session) Save() error {
   dir, err := sessionsDir()
   if err != nil {
       return err
   }
   // ensure directory exists
   if err := os.MkdirAll(dir, 0o755); err != nil {
       return fmt.Errorf("creating sessions dir: %w", err)
   }
   path := filepath.Join(dir, s.filename())
   f, err := os.Create(path)
   if err != nil {
       return fmt.Errorf("creating session file: %w", err)
   }
   defer f.Close()
   enc := json.NewEncoder(f)
   enc.SetIndent("", "  ")
   if err := enc.Encode(s); err != nil {
       return fmt.Errorf("encoding session JSON: %w", err)
   }
   return nil
}

// LoadSessions reads all session JSON files from ~/.ai-notes/sessions and returns them.
func LoadSessions() ([]*Session, error) {
   dir, err := sessionsDir()
   if err != nil {
       return nil, err
   }
   files, err := os.ReadDir(dir)
   if err != nil {
       // If the directory doesn't exist, return empty
       if os.IsNotExist(err) {
           return []*Session{}, nil
       }
       return nil, fmt.Errorf("reading sessions dir: %w", err)
   }
   var sessions []*Session
   for _, fi := range files {
       if fi.IsDir() || filepath.Ext(fi.Name()) != ".json" {
           continue
       }
       path := filepath.Join(dir, fi.Name())
       data, err := os.ReadFile(path)
       if err != nil {
           return nil, fmt.Errorf("reading session file %s: %w", fi.Name(), err)
       }
       var s Session
       if err := json.Unmarshal(data, &s); err != nil {
           return nil, fmt.Errorf("parsing session JSON %s: %w", fi.Name(), err)
       }
       sessions = append(sessions, &s)
   }
   // sort sessions by CreatedAt descending (newest first)
   sort.Slice(sessions, func(i, j int) bool {
       return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
   })
   return sessions, nil
}

// sessionsDir returns the full path to the sessions directory (~/.ai-notes/sessions).
func sessionsDir() (string, error) {
   home, err := os.UserHomeDir()
   if err != nil {
       return "", fmt.Errorf("could not determine home directory: %w", err)
   }
   base := filepath.Join(home, baseDirName)
   return filepath.Join(base, sessionsDirName), nil
}

// filename returns the filename for the session: {ID}.json
func (s *Session) filename() string {
   return s.ID + ".json"
}
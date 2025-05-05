package store

import (
   "fmt"
   "os"
   "path/filepath"
   "sort"
   "strings"
   "time"
)

const (
   notesDirName = "notes"
)

// Note represents a saved summary of a session.
type Note struct {
   ID        string    // unique identifier for note (timestamp)
   SessionID string    // ID of the session this note was generated from
   Title     string    // human-readable title for the note
   Body      string    // content of the note (summary)
   CreatedAt time.Time // when the note was created
}

// notesDir returns the full path to notes directory (~/.ai-notes/notes).
func notesDir() (string, error) {
   home, err := os.UserHomeDir()
   if err != nil {
       return "", fmt.Errorf("could not determine home directory: %w", err)
   }
   base := filepath.Join(home, baseDirName)
   return filepath.Join(base, notesDirName), nil
}

// LoadNotes reads all markdown notes from ~/.ai-notes/notes and returns them sorted by CreatedAt desc.
func LoadNotes() ([]*Note, error) {
   dir, err := notesDir()
   if err != nil {
       return nil, err
   }
   files, err := os.ReadDir(dir)
   if err != nil {
       if os.IsNotExist(err) {
           return []*Note{}, nil
       }
       return nil, fmt.Errorf("reading notes dir: %w", err)
   }
   var notes []*Note
   for _, fi := range files {
       if fi.IsDir() || filepath.Ext(fi.Name()) != ".md" {
           continue
       }
       path := filepath.Join(dir, fi.Name())
       data, err := os.ReadFile(path)
       if err != nil {
           return nil, fmt.Errorf("reading note file %s: %w", fi.Name(), err)
       }
       text := string(data)
       // split title and body
       parts := strings.SplitN(text, "\n", 2)
       title := strings.TrimPrefix(parts[0], "# ")
       body := ""
       if len(parts) > 1 {
           body = strings.TrimSpace(parts[1])
       }
       // parse ID and SessionID from title: Notes-{sessionID}-{id}
       titleParts := strings.SplitN(title, "-", 3)
       var sessionID, id string
       var createdAt time.Time
       if len(titleParts) == 3 && titleParts[0] == "Notes" {
           sessionID = titleParts[1]
           id = titleParts[2]
           t, err := time.Parse("20060102T150405", id)
           if err == nil {
               createdAt = t
           }
       } else {
           // fallback: use filename (without ext) as ID
           id = strings.TrimSuffix(fi.Name(), ".md")
           t, err := time.Parse("20060102T150405", id)
           if err == nil {
               createdAt = t
           }
       }
       note := &Note{
           ID:        id,
           SessionID: sessionID,
           Title:     title,
           Body:      body,
           CreatedAt: createdAt,
       }
       notes = append(notes, note)
   }
   // sort newest first
   sort.Slice(notes, func(i, j int) bool {
       return notes[i].CreatedAt.After(notes[j].CreatedAt)
   })
   return notes, nil
}

// NewNote creates a new Note for a given session ID with the provided body.
func NewNote(sessionID, body string) *Note {
   now := time.Now()
   id := now.Format("20060102T150405")
   title := fmt.Sprintf("Notes-%s-%s", sessionID, id)
   return &Note{
       ID:        id,
       SessionID: sessionID,
       Title:     title,
       Body:      body,
       CreatedAt: now,
   }
}

// Save writes the note as a markdown file to ~/.ai-notes/notes/{ID}.md.
// Returns the full file path or an error.
func (n *Note) Save() (string, error) {
   home, err := os.UserHomeDir()
   if err != nil {
       return "", fmt.Errorf("could not determine home directory: %w", err)
   }
   base := filepath.Join(home, baseDirName)
   dir := filepath.Join(base, notesDirName)
   if err := os.MkdirAll(dir, 0o755); err != nil {
       return "", fmt.Errorf("creating notes dir: %w", err)
   }
   filename := n.ID + ".md"
   path := filepath.Join(dir, filename)
   f, err := os.Create(path)
   if err != nil {
       return "", fmt.Errorf("creating note file: %w", err)
   }
   defer f.Close()
   // write markdown: title and body
   _, err = fmt.Fprintf(f, "# %s\n\n%s", n.Title, n.Body)
   if err != nil {
       return "", fmt.Errorf("writing note file: %w", err)
   }
   return path, nil
}
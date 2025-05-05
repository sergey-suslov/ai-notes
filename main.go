package main

import (
   "fmt"
   "os"
   "github.com/sergey-suslov/ai-notes/ui"
)

func main() {
   if err := ui.Run(); err != nil {
       fmt.Fprintf(os.Stderr, "Error: %v\n", err)
       os.Exit(1)
   }
}
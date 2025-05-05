package openai

import (
   "context"
   "fmt"
   "os"

   openai "github.com/sashabaranov/go-openai"
)

// Client wraps the OpenAI API client.
type Client struct {
   c *openai.Client
}

// NewClient creates a new OpenAI API client using the OPENAI_API_KEY environment variable.
func NewClient() (*Client, error) {
   apiKey := os.Getenv("OPENAI_API_KEY")
   if apiKey == "" {
       return nil, fmt.Errorf("environment variable OPENAI_API_KEY is not set")
   }
   cli := openai.NewClient(apiKey)
   return &Client{c: cli}, nil
}

// ChatCompletion sends a list of messages to the OpenAI Chat Completion API and returns the response content.
// model is the model name to use, e.g. "gpt-3.5-turbo".
func (c *Client) ChatCompletion(ctx context.Context, messages []openai.ChatCompletionMessage, model string) (string, error) {
   req := openai.ChatCompletionRequest{
       Model:    model,
       Messages: messages,
   }
   resp, err := c.c.CreateChatCompletion(ctx, req)
   if err != nil {
       return "", err
   }
   if len(resp.Choices) == 0 {
       return "", fmt.Errorf("no choices returned from OpenAI")
   }
   return resp.Choices[0].Message.Content, nil
}
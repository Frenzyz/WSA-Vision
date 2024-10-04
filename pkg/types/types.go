///H:\WSA\pkg\types\types.go
package types

// CombinedPrompt is the structured LLM response that contains natural language and shell commands
type CombinedPrompt struct {
	NLResponse string   `json:"nlResponse"`
	Commands   []string `json:"commands"`
}

// PromptMessage represents a message in the chat history.
type PromptMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatData represents the data sent to the LLM API
type ChatData struct {
	Model    string          `json:"model"`
	Messages []PromptMessage `json:"messages"`
	Format   string          `json:"format"`
	Stream   bool            `json:"stream"`
}

// LLMResponse represents the response from the LLM API when streaming is disabled
type LLMResponse struct {
	Model     string     `json:"model"`
	CreatedAt string     `json:"created_at"`
	Message   LLMMessage `json:"message"`
	// Include other fields as necessary
}

type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

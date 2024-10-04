///H:\WSA\pkg\assistant\get_shell_command.go
package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"strings"

	"WSAVision/pkg/types"
)

// GetShellCommand generates commands from the LLM based on user input, chat history, optional error context, and command type
func GetShellCommand(userInput string, chatHistory []types.PromptMessage, errorContext string, isInstallation bool) (*types.CombinedPrompt, error) {
	// Path to the system index file
	indexFilePath := "system_index.txt"

	// Load the system index
	systemIndex, err := LoadSystemIndex(indexFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load system index: %w", err)
	}

	// Get the current user's username
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}
	username := currentUser.Username

	// Sanitize username if it contains domain information (e.g., DOMAIN\username)
	if strings.Contains(username, "\\") {
		parts := strings.Split(username, "\\")
		username = parts[len(parts)-1]
	}

	// Summarize the system index to include only key directories with the actual username
	summarizedIndex := summarizeSystemIndex(systemIndex, username)

	// Construct system prompt with error context if available
	systemPrompt := "You are an assistant that generates shell commands for a Windows-based operating system. " +
		"Your responses must always be in the JSON format: { \"nlResponse\": \"\", \"commands\": [\"command1\", \"command2\"] }. " +
		"Your primary role is to take user input and generate appropriate and syntactically correct commands based on the following rules: " +
		"1. For package installation tasks, generate standard PowerShell or CMD commands without any prefixes. " +
		"2. For all other general automation tasks such as interacting with GUI applications, moving windows, typing text, etc., generate complete and valid AutoHotkey scripts prefixed with 'AUTOHOTKEY:'. " +
		"3. Do not include any dangerous commands like 'Remove-Item -Recurse -Force' or 'shutdown', and provide intelligent warnings if such commands are requested. " +
		"4. Ensure that all AutoHotkey scripts are complete and valid, including necessary commands to activate windows and send keystrokes. " +
		"5. Before creating a directory, check if it already exists to avoid errors using correct PowerShell syntax. " +
		"6. Ensure that all commands are single-line (except for AutoHotkey scripts, which can be multi-line) and avoid unnecessary line breaks or escape characters.\n\n" +
		"**Example for Opening Notepad and Writing Text:**\n" +
		"```json\n" +
		"{\n" +
		"  \"nlResponse\": \"Opening Notepad and writing your note.\",\n" +
		"  \"commands\": [\n" +
		"    \"AUTOHOTKEY:Run, notepad.exe\",\n" +
		"    \"AUTOHOTKEY:WinWaitActive, Untitled - Notepad\",\n" +
		"    \"AUTOHOTKEY:Send, This is a test note.\"\n" +
		"  ]\n" +
		"}\n" +
		"```\n\n"

	if errorContext != "" {
		systemPrompt += "The previous command failed with the following error: \"" + sanitizeError(errorContext) + "\". " +
			"Please provide an improved command to address this error.\n\n"
	}

	// Append the summarized system index to the prompt
	systemPrompt += "Here is a summary of key system directories:\n" + summarizedIndex

	systemMessage := types.PromptMessage{
		Role:    "system",
		Content: systemPrompt,
	}

	// Build chat history including system message
	messages := []types.PromptMessage{systemMessage}
	messages = append(messages, chatHistory...)

	// Prepare LLM request data
	chatData := types.ChatData{
		Model:    os.Getenv("LLM_MODEL"), // Model name from environment variable
		Messages: messages,
		Format:   "json",
		Stream:   false, // Disable streaming
	}

	if chatData.Model == "" {
		chatData.Model = "llama3.2" // Default model
	}

	// Make LLM API call
	body, err := json.Marshal(chatData)
	if err != nil {
		return nil, fmt.Errorf("error marshaling chat data: %w", err)
	}

	// Get API endpoint from environment variable or use default
	apiEndpoint := os.Getenv("LLM_API_ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = "http://localhost:11434/api/chat"
	}

	response, err := http.Post(apiEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error making LLM API request: %w", err)
	}
	defer response.Body.Close()

	// Read the entire response body
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading LLM response body: %w", err)
	}

	// Log the prompt and response for debugging
	fmt.Printf("LLM Prompt Sent:\n%s\n", string(body))
	fmt.Printf("LLM Response Received:\n%s\n", string(respBody))

	// Define the struct to unmarshal the response
	var llmResponse types.LLMResponse
	err = json.Unmarshal(respBody, &llmResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w\nResponse body: %s", err, string(respBody))
	}

	assistantMessage := llmResponse.Message.Content

	// Now parse assistantMessage into CombinedPrompt
	var combinedPrompt types.CombinedPrompt
	err = json.Unmarshal([]byte(assistantMessage), &combinedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse assistant's message as CombinedPrompt: %w\nMessage content: %s", err, assistantMessage)
	}

	// Validate the commands
	if len(combinedPrompt.Commands) == 0 {
		return nil, fmt.Errorf("no commands generated by LLM")
	}

	// Additional validation of commands
	for _, cmd := range combinedPrompt.Commands {
		if isDangerousCommand(cmd) {
			return nil, fmt.Errorf("LLM generated a dangerous command: %s", cmd)
		}
	}

	return &combinedPrompt, nil
}

// summarizeSystemIndex creates a summary of key directories from the system index using the actual username
func summarizeSystemIndex(systemIndex string, username string) string {
	// Split the system index into lines
	lines := strings.Split(systemIndex, "\n")
	// Define key directories to include in the summary with actual username
	keyDirs := []string{
		"C:\\Program Files",
		"C:\\Program Files (x86)",
		fmt.Sprintf("C:\\Users\\%s", username),
		"C:\\Windows",
		"C:\\ProgramData",
	}

	var summaryBuilder strings.Builder
	for _, dir := range keyDirs {
		for _, line := range lines {
			// Match directories exactly or ensure they start with the key directory
			if strings.HasPrefix(line, dir) {
				summaryBuilder.WriteString(line + "\n")
				break
			}
		}
	}

	return summaryBuilder.String()
}

// sanitizeError removes sensitive information from error messages before sending to LLM
func sanitizeError(errorMsg string) string {
	// Implement any necessary sanitization, such as removing file paths or sensitive data
	// For simplicity, we'll just trim whitespace here
	return strings.TrimSpace(errorMsg)
}
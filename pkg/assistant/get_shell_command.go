package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	"WSA/pkg/types"
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
    systemPrompt := "You are an AI assistant that helps generate Windows PowerShell commands to achieve user tasks. " +
        "For starting applications, always use the format 'start appname' (e.g., 'start notepad', 'start spotify'). " +
        "Do not include file paths, extensions, or any additional parameters. " +
        "Based on the user's input and system information, provide the necessary commands wrapped in a JSON object. " +
        "Ensure the commands are compatible with PowerShell and do not include any dangerous operations. " +
        "Do not include any additional text or explanations.\n\n" +
        "Response format strictly as follows:\n" +
        "```json\n" +
        "{\n" +
        "  \"nlResponse\": \"Your natural language response to the user.\",\n" +
        "  \"commands\": [\n" +
        "    \"First command\",\n" +
        "    \"Second command\"\n" +
        "  ],\n" +
        "  \"visionNeeded\": false // Set to true if vision is needed, else false.\n" +
        "}\n" +
        "```\n" +
        "Ensure that the JSON is properly formatted and contains no syntax errors. " +
        "Ensure that in the JSON output, all backslashes in file paths are properly escaped as '\\\\'. " +
        "**Do not deviate from the 'start appname' format when starting applications.**"

    if errorContext != "" {
        systemPrompt += "\n\nNote: The previous command failed with the following error: \"" + sanitizeError(errorContext) + "\". " +
            "Please provide an improved command to address this error."
    }

    // Append the summarized system index to the prompt
    systemPrompt += "\n\nHere is a summary of key system directories:\n" + summarizedIndex

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
        Stream:   false, // Disable streaming
    }

    if chatData.Model == "" {
        chatData.Model = "llama3.2" // Default model
    }

    // Make LLM API call to Ollama
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

    // Log the assistant's message separately
    fmt.Printf("Assistant's Message Content:\n%s\n", assistantMessage)

    // Escape backslashes in the assistant's message
    assistantMessage = escapeBackslashesInJSON(assistantMessage)

    // Now parse assistantMessage into CombinedPrompt
    var combinedPrompt types.CombinedPrompt
    err = json.Unmarshal([]byte(assistantMessage), &combinedPrompt)
    if err != nil {
        // Attempt to extract JSON from the assistant's message
        extractedJSON, extractErr := ExtractJSON(assistantMessage)
        if extractErr != nil {
            return nil, fmt.Errorf("failed to parse assistant's message as CombinedPrompt: %w\nMessage content: %s", err, assistantMessage)
        }

        // Retry unmarshaling with the extracted JSON
        err = json.Unmarshal([]byte(extractedJSON), &combinedPrompt)
        if err != nil {
            return nil, fmt.Errorf("failed to parse extracted JSON as CombinedPrompt: %w\nExtracted JSON: %s", err, extractedJSON)
        }
    }

    // Post-process commands to correct any deviations
    for i, cmd := range combinedPrompt.Commands {
        combinedPrompt.Commands[i] = fixStartCommand(cmd)
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

    // Check if vision is needed and ask the user for permission
    if combinedPrompt.VisionNeeded {
        fmt.Println("The assistant requires access to the vision model to proceed. Do you allow this? (yes/no)")
        var userResponse string
        fmt.Print("> ")
        fmt.Scanln(&userResponse)
        if strings.ToLower(userResponse) != "yes" {
            return nil, fmt.Errorf("user denied access to the vision model")
        }
    }

    return &combinedPrompt, nil
}

// fixStartCommand ensures that the command uses the correct 'start appname' format
func fixStartCommand(cmd string) string {
    // Regular expression to match 'start' commands
    re := regexp.MustCompile(`(?i)^start\s+(.+?)(\.exe)?$`)
    matches := re.FindStringSubmatch(cmd)
    if len(matches) > 1 {
        appName := matches[1]
        // Remove any file paths or backslashes
        appName = filepath.Base(appName)
        appName = strings.TrimSuffix(appName, ".exe")
        return fmt.Sprintf("start %s", appName)
    }
    return cmd
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

// escapeBackslashesInJSON escapes backslashes in JSON strings to ensure valid JSON parsing.
func escapeBackslashesInJSON(s string) string {
    // Replace single backslashes with double backslashes
    return strings.ReplaceAll(s, `\`, `\\`)
}

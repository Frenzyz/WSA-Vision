package assistant

import (
	"WSA/pkg/settings"
	"WSA/pkg/types"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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

	// Load settings to get the default browser
	settingsData, err := settings.LoadSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}

	// Include the default browser in the system prompt or use it in command post-processing
	defaultBrowser := settingsData.DefaultBrowser

	// Construct system prompt with error context if available
	systemPrompt := "You are an AI assistant that helps generate macOS Terminal commands to achieve user tasks. " +
		"For starting applications, always use the format 'open -a appname' (e.g., 'open -a TextEdit', 'open -a Spotify'). " +
		"**For closing applications, use the command 'osascript -e \"quit app \\\"AppName\\\"\"' (e.g., 'osascript -e \"quit app \\\"Spotify\\\"\"').** " +
		"Do not include file paths, extensions, or any additional parameters. " +
		"Based on the user's input and system information, provide the necessary commands wrapped in **a single JSON object only**. " +
		"Ensure the commands are compatible with macOS and do not include any dangerous operations. " +
		"Do not include any additional text or explanations.\n\n" +
		"**Response format strictly as follows (do not include any text outside this JSON):**\n" +
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
		"**Ensure that in the JSON output, all special characters, especially double quotes, are properly escaped using backslashes as per JSON format.** " +
		"**Do not include multiple JSON objects or arrays.** " +
		"**When closing applications, ensure the command follows the specified 'osascript' format.**"

	// Example: Include default browser in the system prompt
	systemPrompt += "\n\nNote: The user's default browser is " + defaultBrowser + "."

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

	// Clean and fix the assistant's message before parsing
	assistantMessage = cleanAssistantMessage(assistantMessage)

	// Escape backslashes in the assistant's message
	assistantMessage = escapeBackslashesInJSON(assistantMessage)

	// Attempt to extract JSON from the assistant's message
	extractedJSON, extractErr := ExtractJSON(assistantMessage)
	if extractErr != nil {
		return nil, fmt.Errorf("failed to extract JSON from assistant's message: %w\nMessage content: %s", extractErr, assistantMessage)
	}

	// Now parse extractedJSON into CombinedPrompt
	var combinedPrompt types.CombinedPrompt
	err = json.Unmarshal([]byte(extractedJSON), &combinedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse extracted JSON as CombinedPrompt: %w\nExtracted JSON: %s", err, extractedJSON)
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

// fixStartCommand ensures that the command uses the correct format
func fixStartCommand(cmd string) string {
	// Check for 'open -a' commands with a URL
	reOpenAURL := regexp.MustCompile(`(?i)^open\s+-a\s+("[^"]+"|\S+)\s+("[^"]+"|\S+)$`)
	matchesOpenAURL := reOpenAURL.FindStringSubmatch(cmd)
	if len(matchesOpenAURL) > 2 {
		appName := matchesOpenAURL[1]
		url := matchesOpenAURL[2]
		appName = strings.Trim(appName, `"`)
		appName = filepath.Base(appName)
		appName = strings.TrimSuffix(appName, ".app")
		return fmt.Sprintf(`open -a "%s" "%s"`, appName, url)
	}

	// Check for 'open -a' commands without a URL
	reOpenA := regexp.MustCompile(`(?i)^open\s+-a\s+("[^"]+"|\S+)$`)
	matchesOpenA := reOpenA.FindStringSubmatch(cmd)
	if len(matchesOpenA) > 1 {
		appName := matchesOpenA[1]
		appName = strings.Trim(appName, `"`)
		appName = filepath.Base(appName)
		appName = strings.TrimSuffix(appName, ".app")
		return fmt.Sprintf(`open -a "%s"`, appName)
	}

	// Check for 'open' commands with URLs
	reOpenURL := regexp.MustCompile(`(?i)^open\s+("[^"]+"|\S+)$`)
	matchesOpenURL := reOpenURL.FindStringSubmatch(cmd)
	if len(matchesOpenURL) > 1 {
		url := matchesOpenURL[1]
		url = strings.Trim(url, `"`)
		return fmt.Sprintf(`open "%s"`, url)
	}

	// Check for 'osascript -e' commands
	reOsa := regexp.MustCompile(`(?i)^osascript\s+-e\s+(.+)$`)
	matchesOsa := reOsa.FindStringSubmatch(cmd)
	if len(matchesOsa) > 1 {
		script := matchesOsa[1]
		// Ensure that the script is properly quoted
		script = strings.Trim(script, `"`)
		return fmt.Sprintf(`osascript -e "%s"`, script)
	}

	// Allow other safe commands
	return cmd
}

// cleanAssistantMessage fixes improperly escaped quotes in the assistant's message
func cleanAssistantMessage(message string) string {
	// Remove code fences and language tags
	message = strings.TrimSpace(message)
	message = strings.TrimPrefix(message, "```json")
	message = strings.TrimPrefix(message, "```")
	message = strings.TrimSuffix(message, "```")

	// Replace any occurrences of \" with \\"
	message = strings.ReplaceAll(message, `\"`, `\\"`)

	// Ensure that inner quotes in commands are properly escaped
	re := regexp.MustCompile(`"commands":\s*\[\s*([^\]]+)\s*\]`)
	message = re.ReplaceAllStringFunc(message, func(match string) string {
		// Find the commands array
		cmdRe := regexp.MustCompile(`"commands":\s*\[\s*(.*)\s*\]`)
		cmdMatches := cmdRe.FindStringSubmatch(match)
		if len(cmdMatches) > 1 {
			commandsStr := cmdMatches[1]
			// Split commands
			commands := splitJSONStrings(commandsStr)
			// Properly escape inner double quotes
			for i, cmd := range commands {
				cmd = strings.Trim(cmd, `"`)
				cmd = strings.ReplaceAll(cmd, `\"`, `"`)
				cmd = strings.ReplaceAll(cmd, `"`, `\"`)
				commands[i] = `"` + cmd + `"`
			}
			return `"commands": [` + strings.Join(commands, ", ") + `]`
		}
		return match
	})
	return message
}

// splitJSONStrings splits a string of JSON string literals into a slice
func splitJSONStrings(s string) []string {
	var result []string
	var current strings.Builder
	escaped := false
	inString := false
	for _, r := range s {
		switch r {
		case '"':
			if !escaped {
				inString = !inString
			}
			current.WriteRune(r)
			escaped = false
		case '\\':
			if escaped {
				current.WriteRune(r)
				escaped = false
			} else {
				escaped = true
				current.WriteRune(r)
			}
		case ',':
			if inString {
				current.WriteRune(r)
			} else {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			}
			escaped = false
		default:
			current.WriteRune(r)
			escaped = false
		}
	}
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

// summarizeSystemIndex creates a summary of key directories from the system index using the actual username
func summarizeSystemIndex(systemIndex string, username string) string {
	// Split the system index into lines
	lines := strings.Split(systemIndex, "\n")
	// Define key directories to include in the summary with actual username
	var keyDirs []string

	if runtime.GOOS == "windows" {
		keyDirs = []string{
			"C:\\Program Files",
			"C:\\Program Files (x86)",
			fmt.Sprintf("C:\\Users\\%s", username),
			"C:\\Windows",
			"C:\\ProgramData",
		}
	} else {
		keyDirs = []string{
			"/Applications",
			"/System",
			"/Users/" + username,
			"/Library",
			"/usr",
		}
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

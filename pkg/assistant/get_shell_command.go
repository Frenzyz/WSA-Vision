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
	// Check if this is a simple app control request that we can handle intelligently
	fmt.Printf("Checking smart app control for: '%s'\n", userInput)
	if smartCommand, err := HandleSmartAppControl(userInput); err == nil {
		fmt.Printf("Smart app control succeeded: %s\n", smartCommand)
		return &types.CombinedPrompt{
			NLResponse:   fmt.Sprintf("I'll %s for you.", userInput),
			Commands:     []string{smartCommand},
			VisionNeeded: false,
		}, nil
	} else {
		fmt.Printf("Smart app control failed: %v\n", err)
	}
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

	// Do not alter escape sequences; keep assistant JSON as-is

	// Attempt to extract JSON from the assistant's message
	extractedJSON, extractErr := ExtractJSON(assistantMessage)
	if extractErr != nil {
		return nil, fmt.Errorf("failed to extract JSON from assistant's message: %w\nMessage content: %s", extractErr, assistantMessage)
	}

	// Clean common JSON issues like trailing commas
	extractedJSON = removeTrailingCommas(extractedJSON)

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

	// If the user's intent is to close/quit an app, keep only valid quit commands
	if isCloseIntent(userInput) {
		filtered := make([]string, 0, len(combinedPrompt.Commands))
		for _, cmd := range combinedPrompt.Commands {
			if isValidQuitCommand(cmd) {
				filtered = append(filtered, cmd)
			}
		}
		combinedPrompt.Commands = filtered
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

// HandleSmartAppControl handles simple app control requests intelligently
func HandleSmartAppControl(userInput string) (string, error) {
	input := strings.TrimSpace(userInput)
	fmt.Printf("HandleSmartAppControl called with: '%s'\n", input)
	
	// Check if it's a quit/close intent
	if IsQuitIntent(input) {
		fmt.Printf("Detected quit intent\n")
		appName := ExtractAppNameFromIntent(input)
		fmt.Printf("Extracted app name: '%s'\n", appName)
		if appName != "" {
			return GetSmartQuitCommand(appName)
		}
	}
	
	// Check if it's an open/start intent
	if IsOpenIntent(input) {
		fmt.Printf("Detected open intent\n")
		appName := ExtractAppNameFromIntent(input)
		fmt.Printf("Extracted app name: '%s'\n", appName)
		if appName != "" {
			return GetSmartOpenCommand(appName)
		}
	}
	
	fmt.Printf("No smart app control match found\n")
	return "", fmt.Errorf("not a simple app control request")
}

// fixStartCommand ensures that the command uses the correct format
func fixStartCommand(cmd string) string {
	// Normalize 'open -a' commands (handle app names with spaces or backslash-escaped spaces)
	reOpenAAny := regexp.MustCompile(`(?i)^open\s+-a\s+(.+)$`)
	matchesOpenAAny := reOpenAAny.FindStringSubmatch(cmd)
	if len(matchesOpenAAny) > 1 {
		appPart := strings.TrimSpace(matchesOpenAAny[1])
		// Strip surrounding quotes if present
		appPart = strings.Trim(appPart, `"`)
		appPart = strings.Trim(appPart, "'")
		// Replace backslash-escaped spaces with real spaces
		appPart = strings.ReplaceAll(appPart, `\\ `, " ")
		appPart = strings.ReplaceAll(appPart, `\ `, " ")
		// Remove trailing .app if included
		appPart = filepath.Base(appPart)
		appPart = strings.TrimSuffix(appPart, ".app")
		// Collapse multiple spaces
		appPart = strings.Join(strings.Fields(appPart), " ")
		return fmt.Sprintf(`open -a "%s"`, appPart)
	}

	// Check for 'open' commands with URLs
	reOpenURL := regexp.MustCompile(`(?i)^open\s+("[^"]+"|\S+)$`)
	matchesOpenURL := reOpenURL.FindStringSubmatch(cmd)
	if len(matchesOpenURL) > 1 {
		url := matchesOpenURL[1]
		url = strings.Trim(url, `"`)
		return fmt.Sprintf(`open "%s"`, url)
	}

	// Normalize 'osascript -e' commands; prefer single quotes around the script
	reOsa := regexp.MustCompile(`(?i)^osascript\s+-e\s+(.+)$`)
	matchesOsa := reOsa.FindStringSubmatch(cmd)
	if len(matchesOsa) > 1 {
		script := strings.TrimSpace(matchesOsa[1])
		// Strip only one outer pair of matching quotes if present
		if len(script) >= 2 {
			if (script[0] == '"' && script[len(script)-1] == '"') || (script[0] == '\'' && script[len(script)-1] == '\'') {
				script = script[1 : len(script)-1]
			}
		}
		// Unescape any JSON-escaped quotes \" to "
		script = strings.ReplaceAll(script, `\\"`, `"`)
		script = strings.ReplaceAll(script, `\"`, `"`)
		// If script is a quit command, normalize to quit app "Name"
		lower := strings.ToLower(script)
		if strings.Contains(lower, "quit app") || strings.Contains(lower, "to quit") {
			// Try to extract the app name inside quotes
			name := ""
			if m := regexp.MustCompile(`"([^"]+)"`).FindStringSubmatch(script); len(m) > 1 {
				name = m[1]
			}
			if name == "" {
				// fallback: last word
				parts := strings.Fields(script)
				if len(parts) > 0 {
					name = parts[len(parts)-1]
				}
			}
			if name != "" {
				script = fmt.Sprintf("quit app \"%s\"", name)
			}
		}
		// Escape single quotes for sh single-quoted string: ' -> '\''
		script = strings.ReplaceAll(script, "'", `'"'"'`)
		return fmt.Sprintf("osascript -e '%s'", script)
	}

	// Allow other safe commands
	return cmd
}

// cleanAssistantMessage fixes improperly escaped quotes in the assistant's message
func cleanAssistantMessage(message string) string {
	// Remove code fences and language tags only; avoid altering escapes
	message = strings.TrimSpace(message)
	message = strings.TrimPrefix(message, "```json")
	message = strings.TrimPrefix(message, "```")
	message = strings.TrimSuffix(message, "```")
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

// removeTrailingCommas removes trailing commas before closing brackets/braces in JSON.
func removeTrailingCommas(s string) string {
	// , ] -> ] and , } -> }
	reCommaBeforeBracket := regexp.MustCompile(`,\s*]`)
	s = reCommaBeforeBracket.ReplaceAllString(s, "]")
	reCommaBeforeBrace := regexp.MustCompile(`,\s*}`)
	s = reCommaBeforeBrace.ReplaceAllString(s, "}")
	return s
}

// isCloseIntent returns true if the user's input is about closing/quitting an app.
func isCloseIntent(userInput string) bool {
	in := strings.ToLower(userInput)
	return strings.Contains(in, "close") || strings.Contains(in, "quit") || strings.Contains(in, "exit") || strings.Contains(in, "stop")
}

// isValidQuitCommand checks if a command cleanly quits a macOS app via osascript.
func isValidQuitCommand(cmd string) bool {
	c := strings.TrimSpace(cmd)
	// Accept our normalized single-quoted AppleScript
	if regexp.MustCompile(`(?i)^osascript\s+-e\s+'quit app "[^"]+"'$`).MatchString(c) {
		return true
	}
	// Accept common double-quoted variant (may come from LLM)
	if regexp.MustCompile(`(?i)^osascript\s+-e\s+\"quit app \\\"[^\"]+\\\"\"$`).MatchString(c) {
		return true
	}
	return false
}

package assistant

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"
    "regexp"

    "WSA/pkg/goalengine"
    "WSA/pkg/types"
)

// ExtractJSON attempts to find and return the first JSON object or array in the input string.
func ExtractJSON(input string) (string, error) {
    // Regular expression to match JSON objects or arrays
    jsonRegex := regexp.MustCompile(`\{[\s\S]*\}|\[[\s\S]*\]`)
    matches := jsonRegex.FindAllString(input, -1)
    if len(matches) == 0 {
        return "", fmt.Errorf("no JSON object or array found in the input")
    }
    return matches[0], nil
}

// GenerateTasksFromGoal breaks down a high-level goal into tasks using the LLM
func GenerateTasksFromGoal(goalDescription string) ([]*goalengine.Task, error) {
    // Prepare the system prompt
    systemPrompt := "You are an assistant that helps break down high-level goals into actionable tasks for a Windows-based operating system. " +
        "Please provide a JSON array of tasks with descriptions only. Do not include any additional text, explanations, or commentary.\n\n" +
        "Response format strictly as follows:\n" +
        "```json\n" +
        "[\n" +
        "  { \"description\": \"First task description\" },\n" +
        "  { \"description\": \"Second task description\" }\n" +
        "]\n" +
        "```\n" +
        "Ensure that the JSON is properly formatted and contains no syntax errors. **Do not deviate from this format under any circumstances.**"

    // Prepare the user message with the goal description
    userMessage := "Goal: " + goalDescription

    messages := []types.PromptMessage{
        {
            Role:    "system",
            Content: systemPrompt,
        },
        {
            Role:    "user",
            Content: userMessage,
        },
    }

    // Prepare LLM request data
    chatData := types.ChatData{
        Model:    os.Getenv("LLM_MODEL"), // Model name from environment variable
        Messages: messages,
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
    fmt.Printf("LLM Request Sent:\n%s\n", string(body))
    fmt.Printf("LLM Response Received:\n%s\n", string(respBody)) // Detailed logging

    // Parse the response
    var llmResponse types.LLMResponse
    err = json.Unmarshal(respBody, &llmResponse)
    if err != nil {
        // Attempt to extract JSON from the response body
        extractedJSON, extractErr := ExtractJSON(string(respBody))
        if extractErr != nil {
            return nil, fmt.Errorf("failed to decode LLM response: %w\nResponse body: %s", err, string(respBody))
        }

        // Retry unmarshaling with the extracted JSON
        err = json.Unmarshal([]byte(extractedJSON), &llmResponse)
        if err != nil {
            return nil, fmt.Errorf("failed to parse extracted JSON as LLM response: %w\nExtracted JSON: %s", err, extractedJSON)
        }
    }

    assistantMessage := llmResponse.Message.Content

    // Log the assistant's message separately
    fmt.Printf("Assistant's Message Content:\n%s\n", assistantMessage)

    // Now parse assistantMessage into a list of tasks
    var tasks []struct {
        Description string `json:"description"`
    }
    err = json.Unmarshal([]byte(assistantMessage), &tasks)
    if err != nil {
        // Attempt to extract JSON from the assistant's message
        extractedJSON, extractErr := ExtractJSON(assistantMessage)
        if extractErr != nil {
            return nil, fmt.Errorf("failed to parse assistant's message as tasks: %w\nMessage content: %s", err, assistantMessage)
        }

        // Retry unmarshaling with the extracted JSON
        err = json.Unmarshal([]byte(extractedJSON), &tasks)
        if err != nil {
            return nil, fmt.Errorf("failed to parse extracted JSON as tasks: %w\nExtracted JSON: %s", err, extractedJSON)
        }
    }

    // Convert to goalengine.Task
    var goalTasks []*goalengine.Task
    for _, t := range tasks {
        goalTasks = append(goalTasks, &goalengine.Task{
            Description: t.Description,
            Status:      goalengine.Pending,
            MaxRetries:  3,
        })
    }

    return goalTasks, nil
}

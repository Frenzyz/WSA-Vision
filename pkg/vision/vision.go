package vision

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "os"

    "WSA/pkg/types"
)

// ProcessImage uses LLava via Ollama to process an image and generate a description or extract information
func ProcessImage(imagePath string, question string) (string, error) {
    imageData, err := os.ReadFile(imagePath)
    if err != nil {
        return "", fmt.Errorf("failed to read image file: %w", err)
    }

    // Encode image in base64
    imageBase64 := base64.StdEncoding.EncodeToString(imageData)

    // Prepare the request payload
    payload := map[string]interface{}{
        "model":  "llava",
        "prompt": question,
        "images": []string{imageBase64},
    }

    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return "", fmt.Errorf("failed to marshal payload: %w", err)
    }

    // Send request to Ollama API
    apiEndpoint := os.Getenv("LLM_API_ENDPOINT")
    if apiEndpoint == "" {
        apiEndpoint = "http://localhost:11434/api/generate"
    }

    response, err := http.Post(apiEndpoint, "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
        return "", fmt.Errorf("error making Ollama API request: %w", err)
    }
    defer response.Body.Close()

    // Read the response
    respBody, err := io.ReadAll(response.Body)
    if err != nil {
        return "", fmt.Errorf("error reading Ollama response body: %w", err)
    }

    // Parse the response
    var ollamaResponse types.OllamaResponse
    err = json.Unmarshal(respBody, &ollamaResponse)
    if err != nil {
        return "", fmt.Errorf("failed to decode Ollama response: %w\nResponse body: %s", err, string(respBody))
    }

    return ollamaResponse.Response, nil
}

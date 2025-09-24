package vision

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// AnalyzeWithImages sends a prompt and one or more base64-encoded images to Ollama using
// a multimodal model (defaults to a gemma3 vision-capable variant) and returns the text response.
func AnalyzeWithImages(prompt string, imagesBase64 []string, model string) (string, error) {
	if model == "" {
		// Default to a gemma3 vision-capable model name
		model = os.Getenv("LLM_MODEL")
		if model == "" {
			model = "gemma3:12b"
		}
	}

	// Ollama generate endpoint; see https://github.com/ollama/ollama/blob/main/docs/api.md
	apiEndpoint := os.Getenv("LLM_API_ENDPOINT")
	if apiEndpoint == "" {
		apiEndpoint = "http://localhost:11434/api/generate"
	}

	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"images": imagesBase64,
		"stream": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal vision payload: %w", err)
	}

	resp, err := http.Post(apiEndpoint, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read ollama response: %w", err)
	}

	var parsed struct {
		Response string `json:"response"`
		Model    string `json:"model"`
	}
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w; body=%s", err, string(respBytes))
	}

	return parsed.Response, nil
}

// AnalyzeImagePaths reads image files, base64-encodes them, and calls AnalyzeWithImages.
func AnalyzeImagePaths(prompt string, imagePaths []string, model string) (string, error) {
	images := make([]string, 0, len(imagePaths))
	for _, p := range imagePaths {
		data, err := os.ReadFile(p)
		if err != nil {
			return "", fmt.Errorf("failed to read image %s: %w", p, err)
		}
		images = append(images, base64.StdEncoding.EncodeToString(data))
	}
	return AnalyzeWithImages(prompt, images, model)
}

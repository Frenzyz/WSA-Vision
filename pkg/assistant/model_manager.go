package assistant

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OllamaModel represents a model from Ollama API
type OllamaModel struct {
	Name       string `json:"name"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
}

// OllamaModelsResponse represents the response from Ollama models API
type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

// GetAvailableModels fetches available models from Ollama
func GetAvailableModels() ([]string, error) {
	// Try to get models from Ollama API
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API returned status %d", resp.StatusCode)
	}

	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	var modelNames []string
	for _, model := range modelsResp.Models {
		// Clean up model names (remove :latest suffix if present)
		name := model.Name
		if strings.HasSuffix(name, ":latest") {
			name = strings.TrimSuffix(name, ":latest")
		}
		modelNames = append(modelNames, name)
	}

	// If no models found, return default ones
	if len(modelNames) == 0 {
		return []string{"llama3.2", "gemma3:12b", "gpt-oss:20b"}, nil
	}

	return modelNames, nil
}

// SetDefaultModel updates the default model for command generation
func SetDefaultModel(modelName string) error {
	// This could be extended to persist the model choice
	// For now, we'll just validate that the model exists
	models, err := GetAvailableModels()
	if err != nil {
		return err
	}

	for _, model := range models {
		if model == modelName || model+":latest" == modelName {
			return nil
		}
	}

	return fmt.Errorf("model '%s' not found in available models", modelName)
}
package assistant

import (
	"fmt"
	"os/exec"
	"strings"
)

// PullModel ensures that the specified models are available
func PullModel(modelName string) error {
	modelsToPull := []string{"llama3.2", "llava"}

	if modelName != "" {
		modelsToPull = []string{modelName}
	}

	for _, model := range modelsToPull {
		fmt.Printf("Checking if %s model needs to be pulled...\n", model)

		// Sanitize model name
		safeModelName := strings.ReplaceAll(model, "\"", "")
		safeModelName = strings.ReplaceAll(safeModelName, ";", "")
		safeModelName = strings.TrimSpace(safeModelName)

		// Check if the model is already available
		cmd := exec.Command("ollama", "list")
		output, err := cmd.CombinedOutput()
		if err != nil {
			// If ollama is not installed or not running, skip pulling to avoid fatal error
			fmt.Printf("Ollama not available, skipping model checks and pulls. Error: %v\n", err)
			return nil
		}

		if strings.Contains(string(output), safeModelName) {
			fmt.Printf("Model %s is already available.\n", safeModelName)
			continue
		}

		// Pull the model
		fmt.Printf("Pulling %s model...\n", safeModelName)
		cmd = exec.Command("ollama", "pull", safeModelName)
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to pull %s model: %v\n", safeModelName, err)
			fmt.Printf("Output: %s\n", output)
			// Do not fail hard; continue without vision model
			return nil
		}

		fmt.Printf("Successfully pulled %s model.\n", safeModelName)
	}

	return nil
}

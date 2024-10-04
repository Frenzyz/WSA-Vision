///H:\WSA\pkg\assistant\model_downloader.go
package assistant

import (
	"fmt"
	"os/exec"
	"strings"
)

// PullModel attempts to download the specified model if it doesn't already exist
func PullModel(modelName string) error {
	if modelName == "" {
		modelName = "llama3.2" // Default model name
	}
	fmt.Printf("Checking if %s model needs to be pulled...\n", modelName)

	// Sanitize modelName to prevent command injection
	safeModelName := strings.ReplaceAll(modelName, "\"", "")
	safeModelName = strings.ReplaceAll(safeModelName, ";", "")
	safeModelName = strings.TrimSpace(safeModelName)

	// Check if the model is already available
	cmd := exec.Command("ollama", "list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error listing models: %v\nOutput: %s", err, string(output))
	}

	if strings.Contains(string(output), safeModelName) {
		fmt.Printf("Model %s is already available.\n", safeModelName)
		return nil
	}

	// Pull the model
	fmt.Printf("Pulling %s model...\n", safeModelName)
	cmd = exec.Command("ollama", "pull", safeModelName)
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to pull %s model: %v\n", safeModelName, err)
		fmt.Printf("Output: %s\n", output)
		return err
	}

	fmt.Printf("Successfully pulled %s model.\n", safeModelName)
	return nil
}

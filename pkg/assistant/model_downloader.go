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
            return fmt.Errorf("error listing models: %v\nOutput: %s", err, string(output))
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
            return err
        }

        fmt.Printf("Successfully pulled %s model.\n", safeModelName)
    }

    return nil
}

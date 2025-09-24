package goalengine

import (
	"WSA/pkg/assistant"
	"fmt"
	"log"
)

// ProcessGoal processes a user goal and returns commands
func ProcessGoal(goal string, systemContext map[string]interface{}) ([]string, error) {
	log.Printf("Processing goal: %s", goal)

	// Always use Ollama for command generation
	commands, _, err := assistant.GenerateCommandsWithOllama(goal, "")
	if err != nil {
		return nil, fmt.Errorf("ollama generation failed: %v", err)
	}
	return commands, nil
}

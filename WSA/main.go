package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"WSA/pkg/assistant"
	"WSA/pkg/goalengine"
	"WSA/pkg/logging"
	"WSA/pkg/settings" // New import for settings management
	"WSA/pkg/types"
)

func main() {
	logging.SetupLogging()

	// Ensure the models are loaded
	err := assistant.PullModel("")
	if err != nil {
		log.Fatalf("Failed to load models: %v", err)
	}

	// Generate system index if it doesn't exist
	indexFilePath := "system_index.txt"
	if _, err := os.Stat(indexFilePath); os.IsNotExist(err) {
		err = assistant.GenerateSystemIndex(indexFilePath)
		if err != nil {
			log.Fatalf("Failed to generate system index: %v", err)
		}
		log.Println("System index generation completed successfully.")
	}

	// Load or initialize system settings
	settingsData, err := settings.LoadSettings()
	if err != nil {
		fmt.Printf("Failed to load settings: %v\n", err)
		log.Printf("Failed to load settings: %v\n", err)
		return
	}

	// Prompt the user to confirm default browser if not already set
	if settingsData.DefaultBrowser == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Please enter your default browser (e.g., Safari, Google Chrome, Firefox):")
		fmt.Print("> ")
		browserInput, _ := reader.ReadString('\n')
		settingsData.DefaultBrowser = strings.TrimSpace(browserInput)
		err = settingsData.SaveSettings()
		if err != nil {
			fmt.Printf("Failed to save settings: %v\n", err)
			log.Printf("Failed to save settings: %v\n", err)
			return
		}
	}

	fmt.Println("Enter a high-level goal (e.g., 'set up a development environment', 'organize my files', 'ch' to clear history, or 'exit' to quit):")

	reader := bufio.NewReader(os.Stdin)
	var chatHistory []types.PromptMessage // Initialize chat history

	for {
		fmt.Print("> ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			log.Printf("Error reading input: %v\n", err)
			continue
		}
		userInput = strings.TrimSpace(userInput)

		if strings.ToLower(userInput) == "exit" {
			fmt.Println("Goodbye!")
			break
		}

		if strings.ToLower(userInput) == "ch" {
			chatHistory = nil // Clear chat history
			fmt.Println("Chat history cleared.")
			continue
		}

		// Directly handle 'close app' commands
		if strings.HasPrefix(strings.ToLower(userInput), "close ") {
			appName := strings.TrimSpace(userInput[6:])
			if appName != "" {
				err := assistant.CloseApplication(appName)
				if err != nil {
					fmt.Printf("Failed to close %s: %v\n", appName, err)
					log.Printf("Failed to close %s: %v\n", appName, err)
				} else {
					fmt.Printf("%s has been closed successfully.\n", appName)
				}
				continue
			}
		}

		// Initialize GoalEngine
		goal := &goalengine.Goal{
			Description:  userInput,
			Tasks:        []*goalengine.Task{},
			CurrentState: &goalengine.State{}, // Initialize with current state
			DesiredState: &goalengine.State{}, // Define desired state
		}

		// Generate tasks from the high-level goal
		tasks, err := assistant.GenerateTasksFromGoal(goal.Description)
		if err != nil {
			fmt.Printf("Failed to generate tasks: %v\n", err)
			log.Printf("Failed to generate tasks: %v\n", err)
			continue
		}
		goal.Tasks = tasks

		// Start processing the goal
		processGoal(goal, &chatHistory)
	}
}

func processGoal(goal *goalengine.Goal, chatHistory *[]types.PromptMessage) {
	if len(goal.Tasks) == 0 {
		fmt.Println("No tasks generated. Exiting goal processing.")
		log.Println("No tasks generated. Exiting goal processing.")
		return
	}

	for !goal.IsGoalAchieved() {
		for _, task := range goal.Tasks {
			if task.Status == goalengine.Pending {
				// Process the task
				executeTask(task, chatHistory)
			}
		}
		// Update the goal's current state
		err := goal.UpdateCurrentState()
		if err != nil {
			fmt.Printf("Error updating current state: %v\n", err)
			log.Printf("Error updating current state: %v\n", err)
		} else {
			fmt.Printf("Current State Updated: CPU Usage: %.2f%%, Memory Usage: %.2f%%\n",
				goal.CurrentState.CPUUsage, goal.CurrentState.MemoryUsage)
			log.Printf("Current State Updated: CPU Usage: %.2f%%, Memory Usage: %.2f%%\n",
				goal.CurrentState.CPUUsage, goal.CurrentState.MemoryUsage)
		}
	}

	// After processing, check for failed tasks
	var failedTasks []string
	for _, task := range goal.Tasks {
		if task.Status == goalengine.Failed {
			failedTasks = append(failedTasks, task.Description)
		}
	}

	if len(failedTasks) > 0 {
		fmt.Println("Some tasks could not be completed:")
		for _, desc := range failedTasks {
			fmt.Printf("- %s\n", desc)
		}
	} else {
		fmt.Println("All tasks completed successfully!")
	}
}

func executeTask(task *goalengine.Task, chatHistory *[]types.PromptMessage) {
	task.Attempt++
	task.Status = goalengine.InProgress

	// Add user input to chat history
	*chatHistory = append(*chatHistory, types.PromptMessage{
		Role:    "user",
		Content: task.Description,
	})

	// Get commands for the task
	combinedPrompt, err := assistant.GetShellCommand(task.Description, *chatHistory, task.Feedback, isInstallationCommand(task.Description))
	if err != nil {
		fmt.Printf("Error getting commands for task '%s': %v\n", task.Description, err)
		log.Printf("Error getting commands for task '%s': %v\n", task.Description, err)
		task.Status = goalengine.Failed
		task.Feedback = err.Error()
		logging.LogTaskExecution(task)
		return
	}

	task.Commands = combinedPrompt.Commands

	// Add assistant's response to chat history
	*chatHistory = append(*chatHistory, types.PromptMessage{
		Role:    "assistant",
		Content: combinedPrompt.NLResponse,
	})

	success := true

	// Use vision model if needed
	if combinedPrompt.VisionNeeded {
		err := assistant.UseVisionModel(task.Description)
		if err != nil {
			fmt.Printf("Error using vision model for task '%s': %v\n", task.Description, err)
			log.Printf("Error using vision model for task '%s': %v\n", task.Description, err)
			success = false
			task.Feedback = err.Error()
		}
	}

	// Execute commands
	for _, command := range task.Commands {
		command = strings.TrimSpace(command)
		if command == "" {
			fmt.Printf("Skipping empty or invalid command.\n")
			continue
		}
		err := assistant.ExecuteShellCommand(command)
		if err != nil {
			fmt.Printf("Error executing command '%s': %v\n", command, err)
			log.Printf("Error executing command '%s': %v\n", command, err)
			success = false
			task.Feedback = err.Error()
			break
		}
	}

	if success {
		task.Status = goalengine.Completed
	} else {
		if task.Attempt < task.MaxRetries {
			// Retry the task with improved commands
			executeTask(task, chatHistory)
		} else {
			task.Status = goalengine.Failed
		}
	}

	logging.LogTaskExecution(task)
}

func isInstallationCommand(input string) bool {
	installationKeywords := []string{
		"install", "setup", "download", "add", "configure", "deploy",
	}
	inputLower := strings.ToLower(input)
	for _, keyword := range installationKeywords {
		if strings.Contains(inputLower, keyword) {
			return true
		}
	}
	return false
}

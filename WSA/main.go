package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"WSA/pkg/assistant"
	"WSA/pkg/goalengine"
	"WSA/pkg/logging"
	"WSA/pkg/settings"
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
	_, err = settings.LoadSettings()
	if err != nil {
		fmt.Printf("Failed to load settings: %v\n", err)
		log.Printf("Failed to load settings: %v\n", err)
		return
	}

	// Start HTTP server
	http.HandleFunc("/execute", executeHandler)
	http.HandleFunc("/settings", settingsHandler)
	fmt.Println("Server started at http://localhost:8080")
	log.Println("Server started at http://localhost:8080")
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// Handler for executing commands
func executeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Goal      string `json:"goal"`
		UseVision bool   `json:"useVision"`
	}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	goalDescription := strings.TrimSpace(req.Goal)
	if goalDescription == "" {
		http.Error(w, "Goal cannot be empty", http.StatusBadRequest)
		return
	}

	// Initialize GoalEngine
	goal := &goalengine.Goal{
		Description:  goalDescription,
		Tasks:        []*goalengine.Task{},
		CurrentState: &goalengine.State{}, // Initialize with current state
		DesiredState: &goalengine.State{}, // Define desired state
		Logs:         []string{},
		UseVision:    req.UseVision,
	}

	// Generate tasks from the high-level goal
	tasks, err := assistant.GenerateTasksFromGoal(goal.Description)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate tasks: %v", err), http.StatusInternalServerError)
		return
	}
	goal.Tasks = tasks

	var chatHistory []types.PromptMessage // Initialize chat history

	// Process the goal
	processGoal(goal, &chatHistory)

	// Prepare response
	response := struct {
		Message string   `json:"message"`
		Logs    []string `json:"logs"`
	}{
		Message: "Goal processed successfully",
		Logs:    goal.Logs,
	}
	json.NewEncoder(w).Encode(response)
}

// Handler for getting and setting settings
func settingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return current settings
		settingsData, err := settings.LoadSettings()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load settings: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(settingsData)
	case http.MethodPost:
		// Update settings
		var settingsData settings.Settings
		err := json.NewDecoder(r.Body).Decode(&settingsData)
		if err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		err = settingsData.SaveSettings()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save settings: %v", err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(settingsData)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func processGoal(goal *goalengine.Goal, chatHistory *[]types.PromptMessage) {
	if len(goal.Tasks) == 0 {
		log.Println("No tasks generated. Exiting goal processing.")
		goal.Logs = append(goal.Logs, "No tasks generated. Exiting goal processing.")
		return
	}

	for !goal.IsGoalAchieved() {
		for _, task := range goal.Tasks {
			if task.Status == goalengine.Pending {
				// Process the task
				executeTask(task, chatHistory, goal)
			}
		}
		//// Update the goal's current state
		//err := goal.UpdateCurrentState()
		//if err != nil {
		//	log.Printf("Error updating current state: %v\n", err)
		//	goal.Logs = append(goal.Logs, fmt.Sprintf("Error updating current state: %v", err))
		//} else {
		//	log.Printf("Current State Updated: CPU Usage: %.2f%%, Memory Usage: %.2f%%\n",
		//		goal.CurrentState.CPUUsage, goal.CurrentState.MemoryUsage)
		//	goal.Logs = append(goal.Logs, fmt.Sprintf("Current State Updated: CPU Usage: %.2f%%, Memory Usage: %.2f%%",
		//		goal.CurrentState.CPUUsage, goal.CurrentState.MemoryUsage))
		//}
	}

	// After processing, check for failed tasks
	var failedTasks []string
	for _, task := range goal.Tasks {
		if task.Status == goalengine.Failed {
			failedTasks = append(failedTasks, task.Description)
		}
	}

	if len(failedTasks) > 0 {
		log.Println("Some tasks could not be completed:")
		goal.Logs = append(goal.Logs, "Some tasks could not be completed:")
		for _, desc := range failedTasks {
			log.Printf("- %s\n", desc)
			goal.Logs = append(goal.Logs, fmt.Sprintf("- %s", desc))
		}
	} else {
		log.Println("All tasks completed successfully!")
		goal.Logs = append(goal.Logs, "All tasks completed successfully!")
	}
}

func executeTask(task *goalengine.Task, chatHistory *[]types.PromptMessage, goal *goalengine.Goal) {
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
		log.Printf("Error getting commands for task '%s': %v\n", task.Description, err)
		task.Status = goalengine.Failed
		task.Feedback = err.Error()
		goal.Logs = append(goal.Logs, fmt.Sprintf("Error getting commands for task '%s': %v", task.Description, err))
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

	// Use vision model if needed and allowed
	if combinedPrompt.VisionNeeded && goal.UseVision {
		err := assistant.UseVisionModel(task.Description)
		if err != nil {
			log.Printf("Error using vision model for task '%s': %v\n", task.Description, err)
			success = false
			task.Feedback = err.Error()
			goal.Logs = append(goal.Logs, fmt.Sprintf("Error using vision model for task '%s': %v", task.Description, err))
		}
	} else if combinedPrompt.VisionNeeded && !goal.UseVision {
		log.Printf("Vision model required but not enabled for task '%s'\n", task.Description)
		success = false
		task.Feedback = "Vision model required but not enabled."
		goal.Logs = append(goal.Logs, fmt.Sprintf("Vision model required but not enabled for task '%s'", task.Description))
	}

	// Execute commands
	for _, command := range task.Commands {
		command = strings.TrimSpace(command)
		if command == "" {
			log.Printf("Skipping empty or invalid command.\n")
			goal.Logs = append(goal.Logs, "Skipping empty or invalid command.")
			continue
		}
		err := assistant.ExecuteShellCommand(command)
		if err != nil {
			log.Printf("Error executing command '%s': %v\n", command, err)
			success = false
			task.Feedback = err.Error()
			goal.Logs = append(goal.Logs, fmt.Sprintf("Error executing command '%s': %v", command, err))
			break
		} else {
			goal.Logs = append(goal.Logs, fmt.Sprintf("Command executed successfully: '%s'", command))
		}
	}

	if success {
		task.Status = goalengine.Completed
		goal.Logs = append(goal.Logs, fmt.Sprintf("Task '%s' completed successfully.", task.Description))
	} else {
		if task.Attempt < task.MaxRetries {
			// Retry the task with improved commands
			goal.Logs = append(goal.Logs, fmt.Sprintf("Retrying task '%s'. Attempt %d.", task.Description, task.Attempt))
			executeTask(task, chatHistory, goal)
		} else {
			task.Status = goalengine.Failed
			goal.Logs = append(goal.Logs, fmt.Sprintf("Task '%s' failed after %d attempts.", task.Description, task.Attempt))
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

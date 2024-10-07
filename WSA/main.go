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
    "WSA/pkg/types"
)

func main() {
    logging.SetupLogging()

    // Ensure the model is loaded
    err := assistant.PullModel("")
    if err != nil {
        log.Fatalf("Failed to load model: %v", err)
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

    fmt.Println("Enter a high-level goal (e.g., 'set up a development environment', 'organize my files', or 'exit' to quit):")

    reader := bufio.NewReader(os.Stdin)

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
        processGoal(goal)
    }
}

func processGoal(goal *goalengine.Goal) {
    // Check if no tasks are available to prevent an infinite loop
    if len(goal.Tasks) == 0 {
        fmt.Println("No tasks generated. Exiting goal processing.")
        log.Println("No tasks generated. Exiting goal processing.")
        return
    }

    for !goal.IsGoalAchieved() {
        for _, task := range goal.Tasks {
            if task.Status == goalengine.Pending {
                // Process the task
                executeTask(task)
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
    fmt.Println("All tasks completed successfully!")
}

func executeTask(task *goalengine.Task) {
    task.Attempt++
    task.Status = goalengine.InProgress

    // Get commands for the task
    chatHistory := []types.PromptMessage{
        {
            Role:    "user",
            Content: task.Description,
        },
    }

    combinedPrompt, err := assistant.GetShellCommand(task.Description, chatHistory, task.Feedback, isInstallationCommand(task.Description))
    if err != nil {
        fmt.Printf("Error getting commands for task '%s': %v\n", task.Description, err)
        log.Printf("Error getting commands for task '%s': %v\n", task.Description, err)
        task.Status = goalengine.Failed
        task.Feedback = err.Error()
        logging.LogTaskExecution(task)
        return
    }

    task.Commands = combinedPrompt.Commands

    success := true
    for _, command := range task.Commands {
        if strings.HasPrefix(command, "AUTOHOTKEY:") {
            // Handle AutoHotkey command
            scriptContent := strings.TrimPrefix(command, "AUTOHOTKEY:")
            scriptFileName := "script.ahk"
            err := assistant.SaveAutoHotkeyScript(scriptContent, scriptFileName)
            if err != nil {
                fmt.Printf("Error saving AutoHotkey script: %v\n", err)
                log.Printf("Error saving AutoHotkey script: %v\n", err)
                success = false
                task.Feedback = err.Error()
                break
            }
            err = assistant.ExecuteAutoHotkeyScript(scriptFileName)
            if err != nil {
                fmt.Printf("Error executing AutoHotkey script: %v\n", err)
                log.Printf("Error executing AutoHotkey script: %v\n", err)
                success = false
                task.Feedback = err.Error()
                break
            }
        } else {
            // Execute shell command
            err := assistant.ExecuteShellCommand(command)
            if err != nil {
                fmt.Printf("Error executing command '%s': %v\n", command, err)
                log.Printf("Error executing command '%s': %v\n", command, err)
                success = false
                task.Feedback = err.Error()
                break
            }
        }
    }

    if success {
        task.Status = goalengine.Completed
    } else {
        if task.Attempt < task.MaxRetries {
            // Retry the task with improved commands
            executeTask(task)
        } else {
            task.Status = goalengine.Failed
        }
    }

    logging.LogTaskExecution(task)
}

// isInstallationCommand determines if the user input is related to installing packages
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

///H:\WSA\WSA\main.go
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"WSAVision/pkg/assistant"
	"WSAVision/pkg/logging"
	"WSAVision/pkg/types"
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

	fmt.Println("Enter a command (e.g., 'open notes and write something in notepad', 'install package xyz', or 'exit' to quit):")

	reader := bufio.NewReader(os.Stdin)
	chatHistory := []types.PromptMessage{} // Stores the conversation history

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

		// Append user message to chat history
		chatHistory = append(chatHistory, types.PromptMessage{
			Role:    "user",
			Content: userInput,
		})

		// Determine if the command is an installation task based on keywords
		isInstallation := isInstallationCommand(userInput)

		// Process input with LLM through assistant package
		errorContext := ""
		if len(chatHistory) > 1 {
			// Get the last assistant response if any
			lastAssistant := chatHistory[len(chatHistory)-1]
			errorContext = lastAssistant.Content
		}

		combinedPrompt, err := assistant.GetShellCommand(userInput, chatHistory, errorContext, isInstallation)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			log.Printf("Error getting shell command: %v\n", err)
			continue
		}

		// Append assistant response to chat history
		chatHistory = append(chatHistory, types.PromptMessage{
			Role:    "assistant",
			Content: combinedPrompt.NLResponse,
		})

		// Process LLM response
		if combinedPrompt != nil {
			fmt.Printf("Natural Language Response: %s\n", combinedPrompt.NLResponse)
			fmt.Printf("Generated Commands: %v\n", combinedPrompt.Commands)

			// Separate shell commands and AutoHotkey commands
			var shellCommands []string
			var autoHotkeyCommands []string

			for _, cmd := range combinedPrompt.Commands {
				// Trim whitespace
				cmd = strings.TrimSpace(cmd)
				if cmd == "" {
					continue // Skip empty commands
				}

				// Determine if it's an AutoHotkey command
				if strings.HasPrefix(cmd, "AUTOHOTKEY:") {
					// Extract the AutoHotkey command
					scriptLine := strings.TrimPrefix(cmd, "AUTOHOTKEY:")
					scriptLine = strings.TrimSpace(scriptLine)
					if scriptLine != "" {
						autoHotkeyCommands = append(autoHotkeyCommands, scriptLine)
					}
				} else {
					// Clean the shell command
					cleanCmd := cleanCommand(cmd)
					if cleanCmd != "" {
						shellCommands = append(shellCommands, cleanCmd)
					}
				}
			}

			var wg sync.WaitGroup
			var mu sync.Mutex

			// Execute shell commands
			for _, cmd := range shellCommands {
				cmd := cmd // capture loop variable
				wg.Add(1)
				go func(command string) {
					defer wg.Done()

					maxRetries := 3
					var execErr error
					errorContext := ""

					for attempt := 1; attempt <= maxRetries; attempt++ {
						fmt.Printf("Executing command (Attempt %d): %s\n", attempt, command)
						log.Printf("Executing command (Attempt %d): %s\n", attempt, command)
						execErr = assistant.ExecuteShellCommand(command)
						if execErr == nil {
							// Command executed successfully
							mu.Lock()
							fmt.Printf("Command '%s' executed successfully on attempt %d.\n", command, attempt)
							log.Printf("Command '%s' executed successfully on attempt %d.\n", command, attempt)
							mu.Unlock()
							return
						}

						// Command failed, collect error context
						mu.Lock()
						fmt.Printf("Error executing command '%s': %v\n", command, execErr)
						log.Printf("Error executing command '%s': %v\n", command, execErr)
						mu.Unlock()

						errorContext = execErr.Error()

						// Get improved command from LLM using error context
						improvedPrompt, err := assistant.GetShellCommand(command, chatHistory, errorContext, isInstallation)
						if err != nil {
							mu.Lock()
							fmt.Printf("Error getting improved command from LLM: %v\n", err)
							log.Printf("Error getting improved command from LLM: %v\n", err)
							mu.Unlock()
							break // Cannot get improved command, exit retry loop
						}

						// Append assistant response to chat history
						chatHistory = append(chatHistory, types.PromptMessage{
							Role:    "assistant",
							Content: improvedPrompt.NLResponse,
						})

						// Append improved commands to generated commands
						for _, improvedCmd := range improvedPrompt.Commands {
							improvedCmd = strings.TrimSpace(improvedCmd)
							if improvedCmd != "" && !strings.HasPrefix(improvedCmd, "AUTOHOTKEY:") {
								command = cleanCommand(improvedCmd)
								fmt.Printf("Retrying with improved command: %s\n", command)
								log.Printf("Retrying with improved command: %s\n", command)
								break // Retry with the first improved command
							}
						}
					}

					if execErr != nil {
						mu.Lock()
						fmt.Printf("Failed to execute command '%s' after %d attempts.\n", command, maxRetries)
						log.Printf("Failed to execute command '%s' after %d attempts.\n", command, maxRetries)
						mu.Unlock()
					}
				}(cmd)
			}

			// Execute AutoHotkey commands if any
			if len(autoHotkeyCommands) > 0 {
				wg.Add(1)
				go func(commands []string) {
					defer wg.Done()

					// Combine AutoHotkey commands into a single script
					scriptContent := strings.Join(commands, "\n")

					// Define script file name
					scriptFileName := "script.ahk"

					// Save the script to a file
					err := assistant.SaveAutoHotkeyScript(scriptContent, scriptFileName)
					if err != nil {
						mu.Lock()
						fmt.Printf("Error saving AutoHotkey script: %v\n", err)
						log.Printf("Error saving AutoHotkey script: %v\n", err)
						mu.Unlock()
						return
					}

					// Execute the script
					err = assistant.ExecuteAutoHotkeyScript(scriptFileName)
					if err != nil {
						mu.Lock()
						fmt.Printf("Error executing AutoHotkey script: %v\n", err)
						log.Printf("Error executing AutoHotkey script: %v\n", err)
						mu.Unlock()
						return
					}

					mu.Lock()
					fmt.Printf("AutoHotkey script '%s' executed successfully.\n", scriptFileName)
					log.Printf("AutoHotkey script '%s' executed successfully.\n", scriptFileName)
					mu.Unlock()
				}(autoHotkeyCommands)
			}

			// Wait for all commands to finish
			wg.Wait()
		} else {
			fmt.Println("No commands generated.")
		}
	}
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

// cleanCommand removes excessive line breaks and escape characters for shell commands
func cleanCommand(cmd string) string {
	// Replace escaped quotes with actual quotes
	cmd = strings.ReplaceAll(cmd, `\"`, `"`)
	// Remove carriage returns
	cmd = strings.ReplaceAll(cmd, "\r", "")
	// Do not modify AutoHotkey commands
	if strings.HasPrefix(cmd, "AUTOHOTKEY:") {
		return ""
	}
	// Replace newlines with spaces for shell commands
	cmd = strings.ReplaceAll(cmd, "\n", " ")
	// Remove redundant spaces
	cmd = strings.Join(strings.Fields(cmd), " ")
	return cmd
}

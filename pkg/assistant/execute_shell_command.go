package assistant

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ExecuteShellCommand runs the shell command on the system after validation.
// It returns an error if the command execution fails.
func ExecuteShellCommand(command string) error {
	// Sanitize and validate the command
	if isDangerousCommand(command) {
		return fmt.Errorf("dangerous command detected and blocked: %s", command)
	}

	// Prevent execution of empty or whitespace commands
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty or whitespace command detected and blocked")
	}

	// Ensure the command does not contain multiple commands separated by ';' or '&&' or '||'
	if strings.Contains(command, ";") || strings.Contains(command, "&&") || strings.Contains(command, "||") {
		return fmt.Errorf("chained commands detected and blocked: %s", command)
	}

	// Ensure the command is not an AutoHotkey command
	if strings.HasPrefix(command, "AUTOHOTKEY:") {
		return fmt.Errorf("AutoHotkey commands are no longer supported.")
	}

	var cmd *exec.Cmd

	// Detect the operating system and set the command accordingly
	if runtime.GOOS == "windows" {
		// Use PowerShell on Windows
		cmd = exec.Command("powershell", "-NoProfile", "-Command", command)
	} else {
		// Use /bin/sh on Unix-like systems (macOS, Linux)
		cmd = exec.Command("/bin/sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error executing command '%s': %v\nOutput:\n%s\n", command, err, string(output))
		return fmt.Errorf("error executing command: %v\nOutput: %s", err, string(output))
	}
	fmt.Printf("Command output for '%s':\n%s\n", command, string(output))
	return nil
}

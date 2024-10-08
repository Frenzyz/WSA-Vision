package assistant

import (
    "fmt"
    "os/exec"
    "strings"
)

// ExecuteShellCommand runs the shell command on the system after validation.
// It returns an error if the command execution fails.
func ExecuteShellCommand(command string) error {
    // Sanitize and validate the command
    if isDangerousCommand(command) {
        return fmt.Errorf("dangerous command detected and blocked: %s", command)
    }

    // Ensure the command is not an AutoHotkey command
    if strings.HasPrefix(command, "AUTOHOTKEY:") {
        return fmt.Errorf("AutoHotkey commands are no longer supported.")
    }

    // Execute the command using PowerShell
    cmd := exec.Command("powershell", "-NoProfile", "-Command", command)
    output, err := cmd.CombinedOutput()
    if err != nil {
        fmt.Printf("Error executing command '%s': %v\nOutput:\n%s\n", command, err, string(output))
        return fmt.Errorf("error executing command: %v\nOutput: %s", err, string(output))
    }
    fmt.Printf("Command output for '%s':\n%s\n", command, string(output))
    return nil
}

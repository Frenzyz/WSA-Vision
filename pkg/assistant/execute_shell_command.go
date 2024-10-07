package assistant

import (
    "fmt"
    "os"
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
        return fmt.Errorf("AutoHotkey commands should be handled separately.")
    }

    // Execute the command using PowerShell
    cmd := exec.Command("powershell", "-NoProfile", "-Command", command)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("error executing command: %v\nOutput: %s", err, string(output))
    }
    fmt.Printf("Command output for '%s':\n%s\n", command, string(output))
    return nil
}

// SaveAutoHotkeyScript saves the AutoHotkey script content to a specified file.
// It returns an error if the operation fails.
func SaveAutoHotkeyScript(script string, fileName string) error {
    err := os.WriteFile(fileName, []byte(script), 0644)
    if err != nil {
        return fmt.Errorf("failed to write AutoHotkey script to file: %v", err)
    }
    return nil
}

// ExecuteAutoHotkeyScript executes an AutoHotkey script from the specified file.
// It returns an error if execution fails.
func ExecuteAutoHotkeyScript(fileName string) error {
    // Check if AutoHotkey.exe is accessible
    _, err := exec.LookPath("AutoHotkey.exe")
    if err != nil {
        return fmt.Errorf("AutoHotkey.exe not found in PATH. Please ensure AutoHotkey is installed and the PATH is set correctly")
    }

    // Execute the script using AutoHotkey
    cmd := exec.Command("AutoHotkey.exe", fileName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("error executing AutoHotkey script: %v\nOutput: %s", err, string(output))
    }

    fmt.Printf("AutoHotkey script executed successfully.\nOutput:\n%s\n", string(output))
    return nil
}

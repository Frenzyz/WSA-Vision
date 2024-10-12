package assistant

import (
	"fmt"
	"os/exec"
)

// CloseApplication closes the specified application using osascript.
func CloseApplication(appName string) error {
	// Construct the command
	cmdStr := fmt.Sprintf(`osascript -e 'quit app "%s"'`, appName)
	// Execute the command
	cmd := exec.Command("/bin/sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error closing application '%s': %v\nOutput:\n%s\n", appName, err, string(output))
		return fmt.Errorf("error closing application '%s': %v\nOutput: %s", appName, err, string(output))
	}
	return nil
}

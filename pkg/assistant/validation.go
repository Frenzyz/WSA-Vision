package assistant

import (
	"regexp"
	"strings"
)

// isDangerousCommand checks if the command contains any dangerous operations.
// It returns true if a dangerous command is detected, otherwise false.
func isDangerousCommand(command string) bool {
	dangerousCommands := []string{
		"rm -rf",
		"shutdown",
		"format",
		"del /f /s /q",
		"erase",
		"sc delete",
		"Reg Delete",
		"bcdedit",
		"diskpart",
		"wmic",
		"cipher",
		"takeown",
		"icacls",
		"powershell",
		"Remove-Item",
		"Stop-Process",
		"sudo",
		"dd",
		"mkfs",
		"chmod 000",
		"chown root",
		"rm -d",
		"rmdir",
		// Removed "killall" and "osascript" from dangerous commands
	}

	lowerCommand := strings.ToLower(command)
	for _, cmd := range dangerousCommands {
		if strings.Contains(lowerCommand, strings.ToLower(cmd)) {
			return true
		}
	}

	// Additional safety check for 'killall' command
	if strings.HasPrefix(lowerCommand, "killall") {
		// Allow 'killall AppName' but block 'killall' without arguments or with dangerous arguments
		killallPattern := regexp.MustCompile(`^killall\s+[\w\s]+$`)
		if !killallPattern.MatchString(lowerCommand) {
			return true
		}
	}

	return false
}

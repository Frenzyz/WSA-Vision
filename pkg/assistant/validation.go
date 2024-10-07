package assistant

import (
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
        "powershell -Command \"(New-Object Net.WebClient).DownloadString",
        "Remove-Item",
        "Stop-Process",
    }

    lowerCommand := strings.ToLower(command)
    for _, cmd := range dangerousCommands {
        if strings.Contains(lowerCommand, strings.ToLower(cmd)) {
            return true
        }
    }
    return false
}

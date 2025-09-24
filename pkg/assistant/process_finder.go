package assistant

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunningApp represents a running application with its details
type RunningApp struct {
	Name     string
	BundleID string
	PID      string
}

// GetRunningApplications retrieves all currently running applications with their bundle IDs
func GetRunningApplications() ([]RunningApp, error) {
	// Use osascript to get running applications with bundle identifiers
	cmd := exec.Command("osascript", "-e", `
		tell application "System Events"
			set appList to {}
			repeat with proc in (every application process whose background only is false)
				try
					set appName to name of proc
					set appBundleID to bundle identifier of proc
					set appPID to unix id of proc
					set end of appList to appName & "|" & appBundleID & "|" & (appPID as string)
				on error
					-- Skip apps without bundle ID
				end try
			end repeat
			return appList
		end tell
	`)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get running applications: %w", err)
	}

	var apps []RunningApp
	lines := strings.Split(strings.TrimSpace(string(output)), ", ")
	
	for _, line := range lines {
		parts := strings.Split(line, "|")
		if len(parts) >= 3 {
			apps = append(apps, RunningApp{
				Name:     strings.TrimSpace(parts[0]),
				BundleID: strings.TrimSpace(parts[1]),
				PID:      strings.TrimSpace(parts[2]),
			})
		}
	}

	return apps, nil
}

// FindBestMatchingApp finds the best matching running application for a given query
func FindBestMatchingApp(query string, apps []RunningApp) *RunningApp {
	query = strings.ToLower(strings.TrimSpace(query))
	
	// First, try exact name match
	for _, app := range apps {
		if strings.ToLower(app.Name) == query {
			return &app
		}
	}
	
	// Then try partial name match
	for _, app := range apps {
		if strings.Contains(strings.ToLower(app.Name), query) {
			return &app
		}
	}
	
	// Try bundle ID match (e.g., "spotify" matches "com.spotify.client")
	for _, app := range apps {
		if strings.Contains(strings.ToLower(app.BundleID), query) {
			return &app
		}
	}
	
	// Try fuzzy matching - remove common words and check
	cleanQuery := removeCommonWords(query)
	for _, app := range apps {
		cleanAppName := removeCommonWords(strings.ToLower(app.Name))
		if strings.Contains(cleanAppName, cleanQuery) || strings.Contains(cleanQuery, cleanAppName) {
			return &app
		}
	}
	
	return nil
}

// removeCommonWords removes common words that might interfere with matching
func removeCommonWords(text string) string {
	commonWords := []string{"app", "application", "the", "a", "an", "client", "desktop"}
	words := strings.Fields(text)
	var filtered []string
	
	for _, word := range words {
		isCommon := false
		for _, common := range commonWords {
			if word == common {
				isCommon = true
				break
			}
		}
		if !isCommon {
			filtered = append(filtered, word)
		}
	}
	
	return strings.Join(filtered, " ")
}

// GetSmartQuitCommand generates an intelligent quit command for a given app name
func GetSmartQuitCommand(appQuery string) (string, error) {
	// Get all running applications
	apps, err := GetRunningApplications()
	if err != nil {
		return "", fmt.Errorf("failed to get running applications: %w", err)
	}
	
	// Find the best matching app
	matchedApp := FindBestMatchingApp(appQuery, apps)
	if matchedApp == nil {
		return "", fmt.Errorf("no running application found matching '%s'", appQuery)
	}
	
	// Generate the quit command using the exact app name
	quitCommand := fmt.Sprintf(`osascript -e 'quit app "%s"'`, matchedApp.Name)
	
	fmt.Printf("Smart match: '%s' -> '%s' (Bundle: %s, PID: %s)\n", 
		appQuery, matchedApp.Name, matchedApp.BundleID, matchedApp.PID)
	
	return quitCommand, nil
}

// GetSmartOpenCommand generates an intelligent open command for a given app name
func GetSmartOpenCommand(appQuery string) (string, error) {
	// For opening apps, we can use Spotlight to find installed applications
	cmd := exec.Command("mdfind", "kMDItemKind == 'Application'")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to search for applications: %w", err)
	}
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var bestMatch string
	
	query := strings.ToLower(strings.TrimSpace(appQuery))
	
	// Look for the best matching application
	for _, line := range lines {
		appPath := strings.TrimSpace(line)
		if appPath == "" {
			continue
		}
		
		// Extract app name from path
		appName := strings.TrimSuffix(strings.Split(appPath, "/")[len(strings.Split(appPath, "/"))-1], ".app")
		
		// Check for exact match first
		if strings.ToLower(appName) == query {
			bestMatch = appName
			break
		}
		
		// Check for partial match
		if strings.Contains(strings.ToLower(appName), query) && bestMatch == "" {
			bestMatch = appName
		}
	}
	
	if bestMatch == "" {
		return "", fmt.Errorf("no installed application found matching '%s'", appQuery)
	}
	
	openCommand := fmt.Sprintf(`open -a "%s"`, bestMatch)
	fmt.Printf("Smart match: '%s' -> '%s'\n", appQuery, bestMatch)
	
	return openCommand, nil
}

// IsQuitIntent determines if the user wants to quit/close an application
func IsQuitIntent(input string) bool {
	quitKeywords := []string{"quit", "close", "exit", "stop", "kill", "terminate", "end"}
	inputLower := strings.ToLower(input)
	
	for _, keyword := range quitKeywords {
		if strings.Contains(inputLower, keyword) {
			return true
		}
	}
	
	return false
}

// IsOpenIntent determines if the user wants to open/start an application
func IsOpenIntent(input string) bool {
	openKeywords := []string{"open", "start", "launch", "run", "execute"}
	inputLower := strings.ToLower(input)
	
	for _, keyword := range openKeywords {
		if strings.Contains(inputLower, keyword) {
			return true
		}
	}
	
	return false
}

// ExtractAppNameFromIntent extracts the application name from user intent
func ExtractAppNameFromIntent(input string) string {
	// Remove common action words
	actionWords := []string{"quit", "close", "exit", "stop", "kill", "terminate", "end", 
		"open", "start", "launch", "run", "execute", "the", "app", "application"}
	
	words := strings.Fields(strings.ToLower(input))
	var appWords []string
	
	for _, word := range words {
		isAction := false
		for _, action := range actionWords {
			if word == action {
				isAction = true
				break
			}
		}
		if !isAction {
			appWords = append(appWords, word)
		}
	}
	
	return strings.Join(appWords, " ")
}
package assistant

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// SystemInfo represents comprehensive system information
type SystemInfo struct {
	Directories  map[string]string   `json:"directories"`
	Applications []string            `json:"applications"`
	Processes    []ProcessInfo       `json:"processes"`
	Environment  map[string]string   `json:"environment"`
	Network      NetworkInfo         `json:"network"`
	Filesystem   FilesystemInfo      `json:"filesystem"`
	HomeDir      string              `json:"homeDir"`
	DocumentsDir string              `json:"documentsDir"`
	DownloadsDir string              `json:"downloadsDir"`
	DesktopDir   string              `json:"desktopDir"`
	PicturesDir  string              `json:"picturesDir"`
	MusicDir     string              `json:"musicDir"`
	VideosDir    string              `json:"videosDir"`
}

type ProcessInfo struct {
	Name string `json:"name"`
	PID  string `json:"pid"`
}

type NetworkInfo struct {
	Interfaces []string `json:"interfaces"`
	Hostname   string   `json:"hostname"`
}

type FilesystemInfo struct {
	Drives []DriveInfo `json:"drives"`
}

type DriveInfo struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Available string `json:"available"`
	Total     string `json:"total"`
}

// GetSystemInfo gathers comprehensive system information
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		Directories: make(map[string]string),
		Environment: make(map[string]string),
	}

	// Get user directories
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %v", err)
	}

	info.HomeDir = currentUser.HomeDir
	info.Directories["home"] = currentUser.HomeDir

	// Get standard directories based on OS
	switch runtime.GOOS {
	case "darwin": // macOS
		info.DocumentsDir = filepath.Join(currentUser.HomeDir, "Documents")
		info.DownloadsDir = filepath.Join(currentUser.HomeDir, "Downloads")
		info.DesktopDir = filepath.Join(currentUser.HomeDir, "Desktop")
		info.PicturesDir = filepath.Join(currentUser.HomeDir, "Pictures")
		info.MusicDir = filepath.Join(currentUser.HomeDir, "Music")
		info.VideosDir = filepath.Join(currentUser.HomeDir, "Movies")
	case "windows":
		info.DocumentsDir = filepath.Join(currentUser.HomeDir, "Documents")
		info.DownloadsDir = filepath.Join(currentUser.HomeDir, "Downloads")
		info.DesktopDir = filepath.Join(currentUser.HomeDir, "Desktop")
		info.PicturesDir = filepath.Join(currentUser.HomeDir, "Pictures")
		info.MusicDir = filepath.Join(currentUser.HomeDir, "Music")
		info.VideosDir = filepath.Join(currentUser.HomeDir, "Videos")
	default: // Linux and others
		info.DocumentsDir = filepath.Join(currentUser.HomeDir, "Documents")
		info.DownloadsDir = filepath.Join(currentUser.HomeDir, "Downloads")
		info.DesktopDir = filepath.Join(currentUser.HomeDir, "Desktop")
		info.PicturesDir = filepath.Join(currentUser.HomeDir, "Pictures")
		info.MusicDir = filepath.Join(currentUser.HomeDir, "Music")
		info.VideosDir = filepath.Join(currentUser.HomeDir, "Videos")
	}

	// Add directories to map
	info.Directories["documents"] = info.DocumentsDir
	info.Directories["downloads"] = info.DownloadsDir
	info.Directories["desktop"] = info.DesktopDir
	info.Directories["pictures"] = info.PicturesDir
	info.Directories["music"] = info.MusicDir
	info.Directories["videos"] = info.VideosDir

	// Get installed applications
	info.Applications = getInstalledApplications()

	// Get running processes
	info.Processes = getRunningProcesses()

	// Get environment variables (filtered for security)
	info.Environment = getFilteredEnvironment()

	// Get network information
	info.Network = getNetworkInfo()

	// Get filesystem information
	info.Filesystem = getFilesystemInfo()

	return info, nil
}

// getInstalledApplications returns a list of installed applications
func getInstalledApplications() []string {
	var apps []string

	switch runtime.GOOS {
	case "darwin":
		// macOS - check /Applications and ~/Applications
		appDirs := []string{"/Applications", filepath.Join(os.Getenv("HOME"), "Applications")}
		for _, dir := range appDirs {
			if entries, err := os.ReadDir(dir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() && strings.HasSuffix(entry.Name(), ".app") {
						appName := strings.TrimSuffix(entry.Name(), ".app")
						apps = append(apps, appName)
					}
				}
			}
		}
	case "windows":
		// Windows - check common program directories
		programDirs := []string{
			"C:\\Program Files",
			"C:\\Program Files (x86)",
		}
		for _, dir := range programDirs {
			if entries, err := os.ReadDir(dir); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						apps = append(apps, entry.Name())
					}
				}
			}
		}
	default:
		// Linux - check /usr/bin and /usr/local/bin
		binDirs := []string{"/usr/bin", "/usr/local/bin"}
		for _, dir := range binDirs {
			if entries, err := os.ReadDir(dir); err == nil {
				for _, entry := range entries {
					if !entry.IsDir() {
						apps = append(apps, entry.Name())
					}
				}
			}
		}
	}

	return apps
}

// getRunningProcesses returns a list of running processes
func getRunningProcesses() []ProcessInfo {
	var processes []ProcessInfo

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("ps", "aux")
	case "windows":
		cmd = exec.Command("tasklist", "/fo", "csv")
	default:
		return processes
	}

	output, err := cmd.Output()
	if err != nil {
		log.Printf("Failed to get processes: %v", err)
		return processes
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || line == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			processes = append(processes, ProcessInfo{
				Name: fields[len(fields)-1], // Last field is usually the command
				PID:  fields[1],             // Second field is usually PID
			})
		}
	}

	// Limit to first 50 processes to avoid overwhelming the response
	if len(processes) > 50 {
		processes = processes[:50]
	}

	return processes
}

// getFilteredEnvironment returns filtered environment variables
func getFilteredEnvironment() map[string]string {
	env := make(map[string]string)
	
	// Only include safe environment variables
	safeVars := []string{
		"PATH", "HOME", "USER", "SHELL", "LANG", "LC_ALL",
		"TMPDIR", "TEMP", "TMP", "PWD", "OLDPWD",
	}

	for _, key := range safeVars {
		if value := os.Getenv(key); value != "" {
			env[key] = value
		}
	}

	return env
}

// getNetworkInfo returns network information
func getNetworkInfo() NetworkInfo {
	info := NetworkInfo{
		Interfaces: []string{},
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	// Get network interfaces (simplified)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("ifconfig")
	case "windows":
		cmd = exec.Command("ipconfig")
	default:
		return info
	}

	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, ":") && !strings.Contains(line, "inet") {
				parts := strings.Split(line, ":")
				if len(parts) > 0 {
					interfaceName := strings.TrimSpace(parts[0])
					if interfaceName != "" {
						info.Interfaces = append(info.Interfaces, interfaceName)
					}
				}
			}
		}
	}

	return info
}

// getFilesystemInfo returns filesystem information
func getFilesystemInfo() FilesystemInfo {
	info := FilesystemInfo{
		Drives: []DriveInfo{},
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd = exec.Command("df", "-h")
	case "windows":
		cmd = exec.Command("wmic", "logicaldisk", "get", "size,freespace,caption")
	default:
		return info
	}

	if output, err := cmd.Output(); err == nil {
		lines := strings.Split(string(output), "\n")
		for i, line := range lines {
			if i == 0 || line == "" {
				continue // Skip header and empty lines
			}

			fields := strings.Fields(line)
			if len(fields) >= 4 {
				drive := DriveInfo{
					Name:      fields[0],
					Total:     fields[1],
					Available: fields[3],
					Type:      "disk",
				}
				info.Drives = append(info.Drives, drive)
			}
		}
	}

	return info
}

// PullModel pulls/loads a model (placeholder implementation)
func PullModel(model string) error {
	log.Printf("Loading model: %s", model)
	// This would integrate with Ollama or other model providers
	return nil
}

// GetAvailableModels returns available models
func GetAvailableModels() ([]string, error) {
	// Default models - this would integrate with Ollama in a real implementation
	models := []string{"llama3.2", "gemma3:12b", "gpt-oss:20b"}
	return models, nil
}

// GenerateSystemIndex generates a system index file
func GenerateSystemIndex(filepath string) error {
	log.Printf("Generating system index at: %s", filepath)
	
	// Get system info
	systemInfo, err := GetSystemInfo()
	if err != nil {
		return fmt.Errorf("failed to get system info: %v", err)
	}

	// Create index content
	indexContent := fmt.Sprintf(`System Index Generated: %s
OS: %s
Home Directory: %s
Documents: %s
Downloads: %s
Desktop: %s

Installed Applications: %d found
Running Processes: %d found
Network Interfaces: %d found
Filesystem Drives: %d found

This index helps WSA Assistant understand your system structure for better automation.
`, 
		systemInfo.HomeDir,
		runtime.GOOS,
		systemInfo.HomeDir,
		systemInfo.DocumentsDir,
		systemInfo.DownloadsDir,
		systemInfo.DesktopDir,
		len(systemInfo.Applications),
		len(systemInfo.Processes),
		len(systemInfo.Network.Interfaces),
		len(systemInfo.Filesystem.Drives),
	)

	// Write to file
	return os.WriteFile(filepath, []byte(indexContent), 0644)
}
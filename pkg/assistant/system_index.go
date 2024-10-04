///H:\WSA\pkg\assistant\system_index.go
package assistant

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GenerateSystemIndex scans all drives and directories to create a system index.
// The index is saved to the specified file path.
func GenerateSystemIndex(indexFilePath string) error {
	var drives []string

	if runtime.GOOS == "windows" {
		// On Windows, scan all drive letters (A-Z)
		for d := 'A'; d <= 'Z'; d++ {
			drive := fmt.Sprintf("%c:\\", d)
			if _, err := os.Stat(drive); err == nil {
				drives = append(drives, drive)
			}
		}
	} else {
		// On Unix-like systems, start from root "/"
		drives = append(drives, "/")
	}

	// Create or truncate the index file
	file, err := os.Create(indexFilePath)
	if err != nil {
		return fmt.Errorf("failed to create index file: %w", err)
	}
	defer file.Close()

	// Channel to handle errors from goroutines
	errorChan := make(chan error, len(drives))
	doneChan := make(chan bool, len(drives))

	// Iterate over each drive and walk through directories
	for _, drive := range drives {
		go func(drivePath string) {
			err := filepath.WalkDir(drivePath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					// Skip directories that can't be accessed
					return nil
				}
				if d.IsDir() {
					// Write the directory path to the file
					_, writeErr := fmt.Fprintln(file, path)
					if writeErr != nil {
						return writeErr
					}
				}
				return nil
			})
			if err != nil {
				errorChan <- fmt.Errorf("error walking directory %s: %w", drivePath, err)
			}
			doneChan <- true
		}(drive)
	}

	// Wait for all goroutines to finish
	for i := 0; i < len(drives); i++ {
		select {
		case err := <-errorChan:
			if err != nil {
				// Log the error and continue
				fmt.Printf("Error generating system index: %v\n", err)
			}
		case <-doneChan:
			// Drive processed successfully
		}
	}

	fmt.Printf("System index successfully generated at %s\n", indexFilePath)
	return nil
}

// LoadSystemIndex reads the system index from the specified file path.
// It returns the index as a single string.
func LoadSystemIndex(indexFilePath string) (string, error) {
	data, err := os.ReadFile(indexFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read system index file: %w", err)
	}
	// Convert to string and replace backslashes with double backslashes for JSON compatibility
	indexContent := strings.ReplaceAll(string(data), "\\", "\\\\")
	return indexContent, nil
}
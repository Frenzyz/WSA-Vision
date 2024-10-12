package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Settings struct {
	DefaultBrowser string `json:"defaultBrowser"`
	// Add other settings fields here as needed
}

const settingsFilePath = "system_settings.json"

// LoadSettings loads the settings from the settings file or creates default settings if the file doesn't exist.
func LoadSettings() (*Settings, error) {
	var settings Settings
	if _, err := os.Stat(settingsFilePath); os.IsNotExist(err) {
		// File doesn't exist, return default settings
		return &settings, nil
	}

	data, err := ioutil.ReadFile(settingsFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}

	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}

	return &settings, nil
}

// SaveSettings saves the settings to the settings file.
func (s *Settings) SaveSettings() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	err = ioutil.WriteFile(settingsFilePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	return nil
}

package settings

import (
	"encoding/json"
	"os"
)

// Settings represents application settings
type Settings struct {
	DefaultModel string `json:"defaultModel"`
	VisionMode   bool   `json:"visionMode"`
	LogLevel     string `json:"logLevel"`
}

// LoadSettings loads settings from file or creates default
func LoadSettings() (*Settings, error) {
	settings := &Settings{
		DefaultModel: "llama3.2",
		VisionMode:   false,
		LogLevel:     "info",
	}

	// Try to load from file
	if data, err := os.ReadFile("settings.json"); err == nil {
		json.Unmarshal(data, settings)
	}

	return settings, nil
}

// SaveSettings saves settings to file
func SaveSettings(settings *Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile("settings.json", data, 0644)
}
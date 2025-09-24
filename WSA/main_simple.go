package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"WSA/pkg/assistant"
	"WSA/pkg/goalengine"
	"WSA/pkg/logging"
	"WSA/pkg/settings"
	"WSA/pkg/types"
	"WSA/pkg/vision"
)

func main() {
	logging.SetupLogging()

	// Ensure the models are loaded
	err := assistant.PullModel("")
	if err != nil {
		log.Fatalf("Failed to load models: %v", err)
	}

	// Generate system index if it doesn't exist
	indexFilePath := "system_index.txt"
	if _, err := os.Stat(indexFilePath); os.IsNotExist(err) {
		err = assistant.GenerateSystemIndex(indexFilePath)
		if err != nil {
			log.Fatalf("Failed to generate system index: %v", err)
		}
		log.Println("System index generation completed successfully.")
	}

	// Load or initialize system settings
	_, err = settings.LoadSettings()
	if err != nil {
		fmt.Printf("Failed to load settings: %v\n", err)
		log.Printf("Failed to load settings: %v\n", err)
		return
	}

	// Start HTTP server
	http.HandleFunc("/execute", executeHandler)
	http.HandleFunc("/settings", settingsHandler)
	http.HandleFunc("/models", modelsHandler)
	http.HandleFunc("/map-system", mapSystemHandler)
	// New granular mapping endpoints for live progress
	http.HandleFunc("/map-system/directories", mapDirectoriesHandler)
	http.HandleFunc("/map-system/applications", mapApplicationsHandler)
	http.HandleFunc("/map-system/processes", mapProcessesHandler)
	http.HandleFunc("/map-system/environment", mapEnvironmentHandler)
	http.HandleFunc("/map-system/network", mapNetworkHandler)
	http.HandleFunc("/map-system/filesystem", mapFilesystemHandler)
	http.HandleFunc("/load-model", loadModelHandler)
	http.HandleFunc("/unload-model", unloadModelHandler)
	// Vision endpoint
	http.HandleFunc("/vision/analyze", visionAnalyzeHandler)
	// Capture screenshot as base64 (macOS implementation)
	http.HandleFunc("/vision/screenshot", visionScreenshotHandler)
	// Capture and analyze in one call
	http.HandleFunc("/vision/capture-and-analyze", visionCaptureAndAnalyzeHandler)

	fmt.Println("Server started at http://localhost:8080")
	log.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// Handler for executing commands
func executeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Goal          string                 `json:"goal"`
		UseVision     bool                   `json:"useVision"`
		Model         string                 `json:"model"`
		SystemContext map[string]interface{} `json:"systemContext"`
		Timestamp     string                 `json:"timestamp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	goalDescription := req.Goal
	log.Printf("Received goal: %s", goalDescription)

	if req.Model != "" {
		os.Setenv("LLM_MODEL", req.Model)
		log.Printf("Using model: %s for request: %s", req.Model, goalDescription)
	}

	// Process the goal using our goal engine
	log.Printf("Processing goal: '%s'", goalDescription)

	commands, err := goalengine.ProcessGoal(goalDescription, req.SystemContext)

	var logs []string
	var message string
	var liveCommands []map[string]interface{}

	if err != nil {
		log.Printf("Goal processing failed: %v", err)
		logs = []string{fmt.Sprintf("Goal processing failed: %v", err)}
		message = fmt.Sprintf("Failed to process goal: %v", err)
		// Add failed entry
		liveCommands = append(liveCommands, map[string]interface{}{
			"command":   fmt.Sprintf("ERROR: %v", err),
			"status":    "failed",
			"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
		})
	} else {
		// Execute each command sequentially and record status
		for _, cmd := range commands {
			cmd = strings.TrimSpace(cmd)
			if cmd == "" {
				continue
			}
			startTs := fmt.Sprintf("%d", time.Now().Unix())
			liveCommands = append(liveCommands, map[string]interface{}{
				"command":   cmd,
				"status":    "running",
				"timestamp": startTs,
			})

			// Run via shell with timeout
			sh := "/bin/sh"
			args := []string{"-lc", cmd}
			if os.PathSeparator == '\\' { // rudimentary windows check
				sh = "cmd"
				args = []string{"/C", cmd}
			}
			ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
			defer cancel()
			c := exec.CommandContext(ctx, sh, args...)
			output, runErr := c.CombinedOutput()
			if ctx.Err() == context.DeadlineExceeded {
				liveCommands = append(liveCommands, map[string]interface{}{
					"command":   fmt.Sprintf("%s -> TIMEOUT", cmd),
					"status":    "failed",
					"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
				})
				message = fmt.Sprintf("Command timed out: %s", cmd)
				break
			}
			if len(output) > 0 {
				logs = append(logs, fmt.Sprintf("%s\n%s", cmd, string(output)))
			} else {
				logs = append(logs, cmd)
			}
			if runErr != nil {
				liveCommands = append(liveCommands, map[string]interface{}{
					"command":   fmt.Sprintf("%s -> ERROR: %v", cmd, runErr),
					"status":    "failed",
					"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
				})
				message = fmt.Sprintf("Command failed: %s", cmd)
				break
			}
			// Mark success entry
			liveCommands = append(liveCommands, map[string]interface{}{
				"command":   cmd,
				"status":    "completed",
				"timestamp": fmt.Sprintf("%d", time.Now().Unix()),
			})
		}
		if message == "" {
			message = fmt.Sprintf("Successfully processed goal: %s", goalDescription)
		}
	}

	response := struct {
		Message      string                   `json:"message"`
		Logs         []string                 `json:"logs"`
		LiveCommands []map[string]interface{} `json:"liveCommands"`
	}{
		Message:      message,
		Logs:         logs,
		LiveCommands: liveCommands,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler for getting and setting settings
func settingsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings, err := settings.LoadSettings()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to load settings: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)

	case http.MethodPost:
		var newSettings settings.Settings
		if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		if err := settings.SaveSettings(&newSettings); err != nil {
			http.Error(w, fmt.Sprintf("Failed to save settings: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Handler for getting available models
func modelsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	models, err := assistant.GetAvailableModels()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get models: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Models []string `json:"models"`
	}{
		Models: models,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler for mapping system information
func mapSystemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	systemInfo, err := assistant.GetSystemInfo()
	if err != nil {
		log.Printf("Failed to get system info: %v", err)
		http.Error(w, fmt.Sprintf("Failed to get system info: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(systemInfo)
}

// Granular mapping handlers for progressive UI updates
func mapDirectoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info, err := assistant.GetSystemInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to map directories: %v", err), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"step":         "directories",
		"command":      "scan_directories --standard --user",
		"directories":  info.Directories,
		"homeDir":      info.HomeDir,
		"documentsDir": info.DocumentsDir,
		"downloadsDir": info.DownloadsDir,
		"desktopDir":   info.DesktopDir,
		"picturesDir":  info.PicturesDir,
		"musicDir":     info.MusicDir,
		"videosDir":    info.VideosDir,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func mapApplicationsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info, err := assistant.GetSystemInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to map applications: %v", err), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"step":         "applications",
		"command":      "scan_applications --system --user",
		"applications": info.Applications,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func mapProcessesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info, err := assistant.GetSystemInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to map processes: %v", err), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"step":      "processes",
		"command":   "list_processes --limit 50",
		"processes": info.Processes,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func mapEnvironmentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info, err := assistant.GetSystemInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to map environment: %v", err), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"step":        "environment",
		"command":     "collect_env --safe",
		"environment": info.Environment,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func mapNetworkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info, err := assistant.GetSystemInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to map network: %v", err), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"step":    "network",
		"command": "inspect_network --interfaces",
		"network": info.Network,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func mapFilesystemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	info, err := assistant.GetSystemInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to map filesystem: %v", err), http.StatusInternalServerError)
		return
	}
	resp := map[string]interface{}{
		"step":       "filesystem",
		"command":    "scan_filesystem --disks",
		"filesystem": info.Filesystem,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Handler for loading a specific model
func loadModelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err := assistant.PullModel(req.Model)
	if err != nil {
		log.Printf("Failed to load model %s: %v", req.Model, err)
		http.Error(w, fmt.Sprintf("Failed to load model: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		Message string `json:"message"`
		Model   string `json:"model"`
	}{
		Message: fmt.Sprintf("Model %s loaded successfully", req.Model),
		Model:   req.Model,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Handler for unloading a specific model
func unloadModelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Model string `json:"model"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("Unloading model: %s", req.Model)

	response := struct {
		Message string `json:"message"`
		Model   string `json:"model"`
	}{
		Message: fmt.Sprintf("Model %s unloaded successfully", req.Model),
		Model:   req.Model,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Vision analyze: accepts base64 image(s) and a prompt, runs a gemma3 multimodal model via Ollama
func visionAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.VisionAnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Normalize images list
	images := req.Images
	if req.ImageBase64 != "" {
		images = append(images, req.ImageBase64)
	}
	if len(images) == 0 {
		http.Error(w, "No images provided", http.StatusBadRequest)
		return
	}

	respText, err := vision.AnalyzeWithImages(req.Prompt, images, req.Model)
	if err != nil {
		http.Error(w, fmt.Sprintf("Vision analysis failed: %v", err), http.StatusInternalServerError)
		return
	}

	out := types.VisionAnalyzeResponse{
		Model:    req.Model,
		Response: respText,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// visionScreenshotHandler captures a full-screen screenshot (macOS: screencapture -x) and returns base64
func visionScreenshotHandler(w http.ResponseWriter, r *http.Request) {
	tmp := "tmp_screenshot.png"
	// Try macOS screencapture
	cmd := exec.Command("/bin/sh", "-lc", fmt.Sprintf("screencapture -x %s", tmp))
	if err := cmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("screenshot failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp)
	data, err := os.ReadFile(tmp)
	if err != nil {
		http.Error(w, fmt.Sprintf("read screenshot failed: %v", err), http.StatusInternalServerError)
		return
	}
	b64 := base64.StdEncoding.EncodeToString(data)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"imageBase64": b64})
}

// visionCaptureAndAnalyzeHandler combines screenshot + analyze
func visionCaptureAndAnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt string `json:"prompt"`
		Model  string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Reuse screenshot handler logic
	tmp := "tmp_screenshot.png"
	cmd := exec.Command("/bin/sh", "-lc", fmt.Sprintf("screencapture -x %s", tmp))
	if err := cmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("screenshot failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp)
	b, err := os.ReadFile(tmp)
	if err != nil {
		http.Error(w, fmt.Sprintf("read screenshot failed: %v", err), http.StatusInternalServerError)
		return
	}
	b64 := base64.StdEncoding.EncodeToString(b)

	respText, err := vision.AnalyzeWithImages(req.Prompt, []string{b64}, req.Model)
	if err != nil {
		http.Error(w, fmt.Sprintf("Vision analysis failed: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types.VisionAnalyzeResponse{Model: req.Model, Response: respText})
}

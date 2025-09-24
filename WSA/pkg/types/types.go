package types

// Request represents an execution request
type Request struct {
	Goal          string                 `json:"goal"`
	UseVision     bool                   `json:"useVision"`
	Model         string                 `json:"model"`
	SystemContext map[string]interface{} `json:"systemContext"`
	Timestamp     string                 `json:"timestamp"`
}

// Response represents an execution response
type Response struct {
	Message string   `json:"message"`
	Logs    []string `json:"logs"`
	Success bool     `json:"success"`
}

// VisionAnalyzeRequest is the request payload for /vision/analyze
type VisionAnalyzeRequest struct {
	Prompt      string   `json:"prompt"`
	ImageBase64 string   `json:"imageBase64"`
	Images      []string `json:"images"`
	Model       string   `json:"model"`
}

// VisionAnalyzeResponse is the response payload for /vision/analyze
type VisionAnalyzeResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
}

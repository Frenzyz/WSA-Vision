package assistant

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// GenerateCommandsWithOllama asks an Ollama model to turn a natural language goal
// into a list of shell commands. It returns the commands and the raw response text.
func GenerateCommandsWithOllama(goal string, model string) ([]string, string, error) {
	if model == "" {
		model = os.Getenv("LLM_MODEL")
	}
	if model == "" {
		model = "gemma3:12b"
	}

	// Instruction to constrain output to a simple command list
	prompt := fmt.Sprintf(`You are a command-generation assistant.
Given this goal:
"%s"

Output ONLY the shell commands to execute, one per line. No explanations. No numbering. Each line must be a complete command.

macOS + POSIX sh guidelines (CRITICAL):
- Commands must run under /bin/sh (POSIX). DO NOT use Bash/Zsh-only features like: $'..', [[ ]], arrays, process substitution, or read -d.
- Quote paths and globs: use "..." and escape parentheses in find with \( \) and operators -o/-a.
- For filenames with spaces, prefer find ... -print0 | xargs -0 -I {} <cmd> "{}".
- Create directories with mkdir -p; use mv -n to avoid overwriting.
- Use open -a "App Name" to launch GUI apps.
- For UI automation, you MAY return AppleScript when essential: osascript -e 'tell application "App" to ...'.
- Use idempotent and safe commands. Avoid destructive patterns unless asked.

Examples (style, not answers):
- mkdir -p "~/Downloads/images" && mkdir -p "~/Downloads/non_images"
- find "~/Downloads" -type f \( -iname "*.png" -o -iname "*.jpg" -o -iname "*.jpeg" -o -iname "*.gif" -o -iname "*.webp" -o -iname "*.heic" \) -print0 | xargs -0 -I {} mv -n "{}" "~/Downloads/images/"
- find "~/Downloads" -type f ! \( -iname "*.png" -o -iname "*.jpg" -o -iname "*.jpeg" -o -iname "*.gif" -o -iname "*.webp" -o -iname "*.heic" \) -print0 | xargs -0 -I {} mv -n "{}" "~/Downloads/non_images/"

If the goal is unclear, output a single echo explaining what is missing.`, strings.TrimSpace(goal))

	endpoint := os.Getenv("LLM_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434/api/generate"
	}

	body := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var parsed struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, "", err
	}

	// Parse commands line-by-line, strip code fences if present
	text := strings.TrimSpace(parsed.Response)
	// Remove triple backtick code fences optionally with language tag
	text = strings.TrimPrefix(text, "```bash")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.Trim(text, "`")
	lines := strings.Split(text, "\n")
	commands := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Drop bullets or numbering
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		commands = append(commands, line)
	}
	if len(commands) == 0 {
		return nil, parsed.Response, fmt.Errorf("no commands generated")
	}
	return commands, parsed.Response, nil
}

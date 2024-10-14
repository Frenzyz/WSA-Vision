// File: pkg/assistant/vision_integration.go

package assistant

import (
	"WSA/pkg/vision"
	"fmt"
	"github.com/go-vgo/robotgo"
	"github.com/kbinani/screenshot"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
)

// MoveMouse moves the mouse cursor to the specified (x, y) coordinates.
func MoveMouse(x, y int) error {
	fmt.Printf("Moving mouse to (%d, %d)\n", x, y)
	robotgo.Move(x, y)
	return nil
}

// TypeText types the specified text using the keyboard.
func TypeText(text string) error {
	fmt.Printf("Typing text: %s\n", text)
	robotgo.TypeStr(text)
	return nil
}

// CaptureScreenRegion captures a screenshot of the specified region.
// The region is defined by the top-left corner (x, y) and its width and height.
func CaptureScreenRegion(x, y, width, height int, savePath string) error {
	fmt.Printf("Capturing screen region at (%d, %d) with width %d and height %d\n", x, y, width, height)
	img, err := screenshot.Capture(x, y, width, height)
	if err != nil {
		return fmt.Errorf("failed to capture screen region: %w", err)
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for screenshot: %w", err)
	}

	// Save the screenshot as a PNG file
	file, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create screenshot file: %w", err)
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return fmt.Errorf("failed to encode screenshot as PNG: %w", err)
	}

	fmt.Printf("Screenshot saved to %s\n", savePath)
	return nil
}

// ConfirmMousePosition uses the vision model to verify that the mouse is at the correct position.
// It captures a screenshot around the current mouse position and sends it to the vision model for confirmation.
func ConfirmMousePosition(expectedElement string) (bool, error) {
	// Get current mouse position
	x, y := robotgo.Location()
	fmt.Printf("Current mouse position: (%d, %d)\n", x, y)

	// Define the region to capture around the mouse position
	captureWidth := 200
	captureHeight := 100
	captureX := x - captureWidth/2
	captureY := y - captureHeight/2

	// Define the path to save the screenshot
	screenshotPath := "tmp/mouse_position.png"

	// Capture the screen region
	err := CaptureScreenRegion(captureX, captureY, captureWidth, captureHeight, screenshotPath)
	if err != nil {
		return false, fmt.Errorf("failed to capture screen for vision confirmation: %w", err)
	}

	// Process the image using the vision model
	question := fmt.Sprintf("Is the '%s' element present in the captured image?", expectedElement)
	visionResponse, err := vision.ProcessImage(screenshotPath, question)
	if err != nil {
		return false, fmt.Errorf("vision model processing failed: %w", err)
	}

	fmt.Printf("Vision model response: %s\n", visionResponse)

	// Clean up the screenshot file
	err = os.Remove(screenshotPath)
	if err != nil {
		fmt.Printf("Warning: failed to remove temporary screenshot file: %v\n", err)
	}

	// Determine if the expected element is present based on the vision model's response
	// Use regex to handle variations in the response (e.g., "Yes", "yes", "Yes, the element is present.")
	matched, err := regexp.MatchString(`(?i)\byes\b`, visionResponse)
	if err != nil {
		return false, fmt.Errorf("failed to compile regex: %w", err)
	}

	if matched {
		fmt.Println("Vision confirmation: Mouse is at the correct position.")
		return true, nil
	}

	fmt.Println("Vision confirmation: Mouse is NOT at the correct position.")
	return false, nil
}

///H:\WSA\pkg\logging\logging.go
package logging

import (
	"io"
	"log"
	"os"
)

// SetupLogging sets up logging to both a file and standard output
func SetupLogging() {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Failed to open log file, logging to stdout:", err)
		return
	}
	// Log to both file and standard output
	mw := io.MultiWriter(os.Stdout, file)
	log.SetOutput(mw)
	log.Println("Logging started")
}

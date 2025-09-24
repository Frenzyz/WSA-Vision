package logging

import (
	"log"
	"os"
)

// SetupLogging configures logging for the application
func SetupLogging() {
	// Create or open log file
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}

	// Set log output to file
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	
	log.Println("Logging started")
}
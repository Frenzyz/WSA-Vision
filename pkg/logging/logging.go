package logging

import (
    "database/sql"
    "io"
    "log"
    "os"

    _ "modernc.org/sqlite" // Use the pure Go SQLite driver
    "WSA/pkg/goalengine"
)

var db *sql.DB

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

    // Initialize SQLite database using the pure Go driver
    db, err = sql.Open("sqlite", "./app.db")
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }

    // Ping to verify the connection
    if err := db.Ping(); err != nil {
        log.Fatalf("Failed to ping database: %v", err)
    }

    // Create tables if they don't exist
    createTables()
}

func createTables() {
    createTasksTable := `CREATE TABLE IF NOT EXISTS tasks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        description TEXT,
        status TEXT,
        feedback TEXT,
        timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

    _, err := db.Exec(createTasksTable)
    if err != nil {
        log.Fatalf("Failed to create tasks table: %v", err)
    }
}

// LogTaskExecution logs each task's execution details
func LogTaskExecution(task *goalengine.Task) {
    _, err := db.Exec(`INSERT INTO tasks (description, status, feedback) VALUES (?, ?, ?)`,
        task.Description, taskStatusToString(task.Status), task.Feedback)
    if err != nil {
        log.Printf("Failed to log task execution: %v", err)
    }
}

func taskStatusToString(status goalengine.TaskStatus) string {
    switch status {
    case goalengine.Pending:
        return "Pending"
    case goalengine.InProgress:
        return "InProgress"
    case goalengine.Completed:
        return "Completed"
    case goalengine.Failed:
        return "Failed"
    default:
        return "Unknown"
    }
}

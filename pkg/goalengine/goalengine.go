// pkg/goalengine/goalengine.go
package goalengine

import (
    "sync"
)

// Goal represents a high-level objective
type Goal struct {
    Description  string
    Tasks        []*Task
    CurrentState *State
    DesiredState *State
    Mutex        sync.Mutex
}

// Task represents an actionable step towards a goal
type Task struct {
    Description   string
    Commands      []string
    Status        TaskStatus
    Attempt       int
    MaxRetries    int
    Feedback      string
    Dependencies  []*Task
    AssignedAgent string
}

// TaskStatus represents the execution status of a Task
type TaskStatus int

const (
    Pending TaskStatus = iota
    InProgress
    Completed
    Failed
)

// State represents the system state
type State struct {
    // Define relevant system state properties
    // For example, installed packages, running services, etc.
}

// AddTask adds a new task to the goal
func (g *Goal) AddTask(task *Task) {
    g.Mutex.Lock()
    defer g.Mutex.Unlock()
    g.Tasks = append(g.Tasks, task)
}

// UpdateTaskStatus updates the status of a task
func (g *Goal) UpdateTaskStatus(task *Task, status TaskStatus) {
    g.Mutex.Lock()
    defer g.Mutex.Unlock()
    task.Status = status
}

// IsGoalAchieved checks if all tasks are completed
func (g *Goal) IsGoalAchieved() bool {
    g.Mutex.Lock()
    defer g.Mutex.Unlock()
    for _, task := range g.Tasks {
        if task.Status != Completed {
            return false
        }
    }
    return true
}

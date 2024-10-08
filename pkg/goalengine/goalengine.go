package goalengine

import (
    "sync"

    "WSA/pkg/statemonitor"
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
    RunningProcesses []string
    CPUUsage         float64
    MemoryUsage      float64
    // Add more fields as needed
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

// IsGoalAchieved checks if all tasks are completed or failed
func (g *Goal) IsGoalAchieved() bool {
    g.Mutex.Lock()
    defer g.Mutex.Unlock()
    for _, task := range g.Tasks {
        if task.Status == Pending || task.Status == InProgress {
            return false
        }
    }
    return true
}

// UpdateCurrentState updates the current state of the goal
func (g *Goal) UpdateCurrentState() error {
    state, err := statemonitor.GetCurrentState()
    if err != nil {
        return err
    }
    g.CurrentState = &State{
        RunningProcesses: state.RunningProcesses,
        CPUUsage:         state.CPUUsage,
        MemoryUsage:      state.MemoryUsage,
        // Map additional fields from SystemState to State here
    }
    return nil
}

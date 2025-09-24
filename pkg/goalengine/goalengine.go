package goalengine

type TaskStatus int

const (
	Pending TaskStatus = iota
	InProgress
	Completed
	Failed
)

type Task struct {
	Description string
	Status      TaskStatus
	Commands    []string
	Feedback    string
	Attempt     int
	MaxRetries  int
}

type State struct {
	CPUUsage    float64
	MemoryUsage float64
	// Add other system state fields as needed
}

type Goal struct {
	Description  string
	Tasks        []*Task
	CurrentState *State
	DesiredState *State
	Logs         []string
	UseVision    bool
}

func (g *Goal) IsGoalAchieved() bool {
	for _, task := range g.Tasks {
		if task.Status != Completed {
			return false
		}
	}
	return true
}

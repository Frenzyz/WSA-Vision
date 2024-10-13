package goalengine

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

func (g *Goal) UpdateCurrentState() error {
	// Update the current state with real system metrics
	// For example, update CPU and memory usage
	// Implement the logic to get actual system metrics
	return nil
}

package statemonitor

import (
    "github.com/shirou/gopsutil/v3/cpu"
    "github.com/shirou/gopsutil/v3/mem"
    "github.com/shirou/gopsutil/v3/process"
)

type SystemState struct {
    RunningProcesses []string
    CPUUsage         float64
    MemoryUsage      float64
    // ... other state information ...
}

func GetCurrentState() (*SystemState, error) {
    // Get running processes
    processes, err := process.Processes()
    if err != nil {
        return nil, err
    }

    var processNames []string
    for _, p := range processes {
        name, err := p.Name()
        if err == nil {
            processNames = append(processNames, name)
        }
    }

    // Get CPU usage
    cpuUsages, err := cpu.Percent(0, false)
    if err != nil {
        return nil, err
    }
    var cpuUsage float64
    if len(cpuUsages) > 0 {
        cpuUsage = cpuUsages[0]
    }

    // Get Memory usage
    vmStat, err := mem.VirtualMemory()
    if err != nil {
        return nil, err
    }
    memoryUsage := vmStat.UsedPercent

    state := &SystemState{
        RunningProcesses: processNames,
        CPUUsage:         cpuUsage,
        MemoryUsage:      memoryUsage,
        // ... populate other state information ...
    }

    return state, nil
}

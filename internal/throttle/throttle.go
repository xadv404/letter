package throttle

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type Level int

const (
	Normal Level = iota
	Moderate
	High
	Critical
)

func (l Level) String() string {
	switch l {
	case Normal:
		return "NORMAL"
	case Moderate:
		return "MODERATE"
	case High:
		return "HIGH"
	case Critical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

type Snapshot struct {
	Level       Level
	CPUPercent  float64
	RAMPercent  float64
	DelayMS     int
	Depth       int
	PageLimit   int
	Workers     int
	Goroutines  int
	CollectedAt time.Time
}

type Controller struct {
	mu sync.RWMutex

	baseDelayMS   int
	baseDepth     int
	basePageLimit int
	baseWorkers   int

	current Snapshot
}

func New(delayMS, depth, pageLimit, workers int) *Controller {
	c := &Controller{
		baseDelayMS:   delayMS,
		baseDepth:     depth,
		basePageLimit: pageLimit,
		baseWorkers:   workers,
	}
	c.current = Snapshot{
		Level:       Normal,
		DelayMS:     delayMS,
		Depth:       depth,
		PageLimit:   pageLimit,
		Workers:     workers,
		CollectedAt: time.Now(),
	}
	return c
}

func (c *Controller) Refresh(goroutines int) Snapshot {
	cpuVal := readCPU()
	ramVal := readRAM()

	level := classify(cpuVal, ramVal)
	delay, depth, pages, workers := c.apply(level)

	snap := Snapshot{
		Level:       level,
		CPUPercent:  cpuVal,
		RAMPercent:  ramVal,
		DelayMS:     delay,
		Depth:       depth,
		PageLimit:   pages,
		Workers:     workers,
		Goroutines:  goroutines,
		CollectedAt: time.Now(),
	}

	c.mu.Lock()
	c.current = snap
	c.mu.Unlock()
	return snap
}

func (c *Controller) Current() Snapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.current
}

func readCPU() float64 {
	pcts, err := cpu.Percent(200*time.Millisecond, false)
	if err != nil || len(pcts) == 0 {
		return 0
	}
	v := pcts[0]
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func readRAM() float64 {
	info, err := mem.VirtualMemory()
	if err != nil || info == nil {
		return 0
	}
	return info.UsedPercent
}

func classify(cpu, ram float64) Level {
	switch {
	case cpu >= 90 || ram >= 90:
		return Critical
	case cpu >= 75 || ram >= 80:
		return High
	case cpu >= 55 || ram >= 65:
		return Moderate
	default:
		return Normal
	}
}

func (c *Controller) apply(level Level) (delayMS, depth, pageLimit, workers int) {
	delayMS = c.baseDelayMS
	depth = c.baseDepth
	pageLimit = c.basePageLimit
	workers = c.baseWorkers

	switch level {
	case Moderate:
		delayMS *= 2
		depth = max(1, depth-1)
	case High:
		delayMS *= 3
		depth = max(1, depth-1)
		pageLimit = int(float64(pageLimit) * 0.7)
		workers = max(1, workers/2)
	case Critical:
		delayMS = min(delayMS*4, 2000)
		depth = 1
		pageLimit = max(10, pageLimit/4)
		workers = 1
	}
	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

package throttle

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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

	prevIdle  uint64
	prevTotal uint64
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
	cpuVal := c.readCPU()
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

func (c *Controller) readCPU() float64 {
	idle, total, ok := readProcStat()
	if !ok || c.prevTotal == 0 {
		c.prevIdle, c.prevTotal = idle, total
		return 0
	}
	idleDelta := float64(idle - c.prevIdle)
	totalDelta := float64(total - c.prevTotal)
	c.prevIdle, c.prevTotal = idle, total
	if totalDelta <= 0 {
		return 0
	}
	usage := (1.0 - idleDelta/totalDelta) * 100
	if usage < 0 {
		return 0
	}
	if usage > 100 {
		return 100
	}
	return usage
}

func readProcStat() (idle, total uint64, ok bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return 0, 0, false
	}
	fields := strings.Fields(sc.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, 0, false
		}
		total += v
		if i == 4 {
			idle = v
		}
	}
	return idle, total, true
}

func readRAM() float64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	var total, available float64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			total = parseKB(line)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			available = parseKB(line)
		}
	}
	if total == 0 {
		return 0
	}
	used := total - available
	return (used / total) * 100
}

func parseKB(line string) float64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, _ := strconv.ParseFloat(fields[1], 64)
	return v
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

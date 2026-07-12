package monitor

import (
	"time"

	"github.com/xadv404/letter/internal/params"
	"github.com/xadv404/letter/internal/throttle"
)

// UISnapshot is a thread-safe copy of dashboard state for the desktop UI.
type UISnapshot struct {
	Elapsed    time.Duration
	Phase      int
	PhaseLabel string
	CPU        float64
	RAM        float64
	Throttle   string
	Workers    int
	DelayMS    int
	Depth      int
	PageLimit  int
	Goroutines int
	Keywords   int
	Params     int
	Accepted   int
	Rejected   int
	Domains    []DomainStatus
	Decisions  []params.FilterDecision
	DorkPreview string
	Running    bool
}

func (d *Dashboard) Snapshot(
	phase int,
	phaseLabel string,
	snap throttle.Snapshot,
	decisions []params.FilterDecision,
	accepted, rejected int,
	dorkPreview string,
	running bool,
) UISnapshot {
	d.mu.RLock()
	defer d.mu.RUnlock()

	domains := make([]DomainStatus, 0, len(d.domains))
	for _, st := range d.domains {
		domains = append(domains, *st)
	}

	decs := make([]params.FilterDecision, len(decisions))
	copy(decs, decisions)

	return UISnapshot{
		Elapsed:     time.Since(d.started).Round(time.Second),
		Phase:       phase,
		PhaseLabel:  phaseLabel,
		CPU:         snap.CPUPercent,
		RAM:         snap.RAMPercent,
		Throttle:    snap.Level.String(),
		Workers:     snap.Workers,
		DelayMS:     snap.DelayMS,
		Depth:       snap.Depth,
		PageLimit:   snap.PageLimit,
		Goroutines:  snap.Goroutines,
		Keywords:    d.keywordsFound,
		Params:      d.paramsFound,
		Accepted:    accepted,
		Rejected:    rejected,
		Domains:     domains,
		Decisions:   decs,
		DorkPreview: dorkPreview,
		Running:     running,
	}
}

func (d *Dashboard) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.started = time.Now()
	d.domains = map[string]*DomainStatus{}
	d.keywordsFound = 0
	d.paramsFound = 0
}

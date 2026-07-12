package monitor

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/xadv404/letter/internal/params"
	"github.com/xadv404/letter/internal/throttle"
)

type DomainStatus struct {
	Domain   string
	Pages    int
	Errors   int
	Finished bool
}

type TimePoint struct {
	At       time.Time
	Keywords int
	Params   int
}

type Dashboard struct {
	mu            sync.RWMutex
	started       time.Time
	domains       map[string]*DomainStatus
	keywordsFound int
	paramsFound   int
	series        []TimePoint
}

func New() *Dashboard {
	return &Dashboard{
		started: time.Now(),
		domains: map[string]*DomainStatus{},
	}
}

func (d *Dashboard) Ensure(domain string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.domains[domain]; !ok {
		d.domains[domain] = &DomainStatus{Domain: domain}
	}
}

func (d *Dashboard) UpdateDomain(domain string, pages, errors int, finished bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	st, ok := d.domains[domain]
	if !ok {
		st = &DomainStatus{Domain: domain}
		d.domains[domain] = st
	}
	st.Pages = pages
	st.Errors = errors
	st.Finished = finished
}

func (d *Dashboard) AddKeywords(n int) {
	d.mu.Lock()
	d.keywordsFound += n
	d.mu.Unlock()
}

func (d *Dashboard) RecordSeries(keywords, params int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.series = append(d.series, TimePoint{
		At: time.Now().UTC(), Keywords: keywords, Params: params,
	})
	const maxPoints = 120
	if len(d.series) > maxPoints {
		d.series = d.series[len(d.series)-maxPoints:]
	}
}

func (d *Dashboard) AddParams(n int) {
	d.mu.Lock()
	d.paramsFound += n
	d.mu.Unlock()
}

func (d *Dashboard) Series() []TimePoint {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]TimePoint, len(d.series))
	copy(out, d.series)
	return out
}

func (d *Dashboard) Render(w io.Writer, snap throttle.Snapshot, decisions []params.FilterDecision, accepted, rejected int) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	elapsed := time.Since(d.started).Round(time.Second)
	fmt.Fprintf(w, "\n=== Letter Recon Dashboard ===\n")
	fmt.Fprintf(w, "Elapsed: %s | CPU: %.1f%% | RAM: %.1f%% | Throttle: %s\n",
		elapsed, snap.CPUPercent, snap.RAMPercent, snap.Level)
	fmt.Fprintf(w, "Workers: %d | Delay: %dms | Depth: %d | PageLimit: %d | Goroutines: %d\n",
		snap.Workers, snap.DelayMS, snap.Depth, snap.PageLimit, snap.Goroutines)
	fmt.Fprintf(w, "Keywords: %d | Parameters: %d | Filter accepted/rejected: %d/%d\n\n",
		d.keywordsFound, d.paramsFound, accepted, rejected)

	fmt.Fprintln(w, "Per-domain progress:")
	for _, st := range d.domains {
		status := "running"
		if st.Finished {
			status = "done"
		}
		fmt.Fprintf(w, "  %-40s pages=%4d errors=%3d [%s]\n", st.Domain, st.Pages, st.Errors, status)
	}

	if len(decisions) > 0 {
		fmt.Fprintln(w, "\nLast filter decisions:")
		start := 0
		if len(decisions) > 100 {
			start = len(decisions) - 100
		}
		for _, dec := range decisions[start:] {
			flag := "REJECT"
			if dec.Accepted {
				flag = "ACCEPT"
			}
			fmt.Fprintf(w, "  [%s] %s score=%d tier=%s — %s\n", flag, dec.Param, dec.Score, dec.Tier, dec.Reason)
		}
	}
	fmt.Fprintln(w, strings.Repeat("-", 48))
}

package crawler

import "github.com/xadv404/letter/internal/monitor"

// Events receives live updates for the HTML dashboard.
type Events struct {
	OnSnapshot  func(monitor.UISnapshot)
	OnLog       func(string)
	OnDorksDone func(dorksPath string)
}

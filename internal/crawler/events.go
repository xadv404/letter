package crawler

import "github.com/xadv404/letter/internal/monitor"

// Events receives live updates for the desktop UI.
type Events struct {
	OnSnapshot func(monitor.UISnapshot)
	OnLog      func(string)
}

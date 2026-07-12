package ui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/xadv404/letter/internal/config"
	"github.com/xadv404/letter/internal/crawler"
	"github.com/xadv404/letter/internal/monitor"
)

// App is the Letter Recon desktop UI.
type App struct {
	fyneApp fyne.App
	window  fyne.Window

	domainFile *widget.Entry
	outputDir  *widget.Entry
	depth      *widget.Slider
	pages      *widget.Slider
	workers    *widget.Slider
	delay      *widget.Slider
	minScore   *widget.Slider

	phaseBind    binding.String
	cpuBind      binding.String
	ramBind      binding.String
	throttleBind binding.String
	statsBind    binding.String
	elapsedBind  binding.String
	decisionBind binding.String
	dorkBind     binding.String
	logBind      binding.String

	domainTable *widget.Table

	startBtn  *widget.Button
	pauseBtn  *widget.Button
	resumeBtn *widget.Button
	stopBtn   *widget.Button

	mu         sync.Mutex
	domainRows []monitor.DomainStatus
	running    bool
	engine     *crawler.Engine
	cancel     context.CancelFunc
}

func Run() {
	a := &App{}
	a.fyneApp = app.NewWithID("com.xadv404.letter")
	a.window = a.fyneApp.NewWindow("Letter Recon")
	a.window.Resize(fyne.NewSize(1180, 760))
	a.window.SetContent(a.build())
	a.window.ShowAndRun()
}

func (a *App) build() fyne.CanvasObject {
	a.phaseBind = binding.NewString()
	a.cpuBind = binding.NewString()
	a.ramBind = binding.NewString()
	a.throttleBind = binding.NewString()
	a.statsBind = binding.NewString()
	a.elapsedBind = binding.NewString()
	a.decisionBind = binding.NewString()
	a.dorkBind = binding.NewString()
	a.logBind = binding.NewString()

	_ = a.phaseBind.Set("Idle — ready")
	_ = a.cpuBind.Set("CPU: —")
	_ = a.ramBind.Set("RAM: —")
	_ = a.throttleBind.Set("Throttle: NORMAL")
	_ = a.statsBind.Set("Keywords: 0 | Params: 0 | Filter: 0/0")
	_ = a.elapsedBind.Set("Elapsed: 0s")

	a.domainFile = widget.NewEntry()
	a.domainFile.SetPlaceHolder("domains.txt")
	a.outputDir = widget.NewEntry()
	a.outputDir.SetText("./output")

	a.depth = widget.NewSlider(1, 10)
	a.depth.Step = 1
	a.depth.SetValue(3)
	a.pages = widget.NewSlider(10, 500)
	a.pages.Step = 10
	a.pages.SetValue(100)
	a.workers = widget.NewSlider(1, 32)
	a.workers.Step = 1
	a.workers.SetValue(8)
	a.delay = widget.NewSlider(50, 2000)
	a.delay.Step = 50
	a.delay.SetValue(250)
	a.minScore = widget.NewSlider(50, 100)
	a.minScore.Step = 1
	a.minScore.SetValue(65)

	a.domainTable = widget.NewTable(
		func() (int, int) { return len(a.domainRows) + 1, 4 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				headers := []string{"Domain", "Pages", "Errors", "Status"}
				lbl.SetText(headers[id.Col])
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				return
			}
			row := a.domainRows[id.Row-1]
			switch id.Col {
			case 0:
				lbl.SetText(row.Domain)
			case 1:
				lbl.SetText(strconv.Itoa(row.Pages))
			case 2:
				lbl.SetText(strconv.Itoa(row.Errors))
			case 3:
				if row.Finished {
					lbl.SetText("done")
				} else {
					lbl.SetText("running")
				}
			}
			lbl.TextStyle = fyne.TextStyle{}
		},
	)
	a.domainTable.SetColumnWidth(0, 280)
	a.domainTable.SetColumnWidth(1, 70)
	a.domainTable.SetColumnWidth(2, 70)
	a.domainTable.SetColumnWidth(3, 90)

	browseBtn := widget.NewButton("Browse…", func() {
		d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			a.domainFile.SetText(r.URI().Path())
			r.Close()
		}, a.window)
		d.Show()
	})

	a.startBtn = widget.NewButton("Start", a.onStart)
	a.pauseBtn = widget.NewButton("Pause", a.onPause)
	a.resumeBtn = widget.NewButton("Resume", a.onResume)
	a.stopBtn = widget.NewButton("Stop", a.onStop)
	a.pauseBtn.Disable()
	a.resumeBtn.Disable()
	a.stopBtn.Disable()

	controls := container.NewGridWithColumns(4, a.startBtn, a.pauseBtn, a.resumeBtn, a.stopBtn)

	configCard := widget.NewCard("Configuration", "4-phase recon pipeline",
		container.NewVBox(
			container.NewBorder(nil, nil, widget.NewLabel("Domains file"), browseBtn, a.domainFile),
			container.NewBorder(nil, nil, widget.NewLabel("Output dir"), nil, a.outputDir),
			widget.NewSeparator(),
			a.labeledSlider("Depth", a.depth, func(v float64) string { return fmt.Sprintf("%.0f", v) }),
			a.labeledSlider("Pages / domain", a.pages, func(v float64) string { return fmt.Sprintf("%.0f", v) }),
			a.labeledSlider("Workers", a.workers, func(v float64) string { return fmt.Sprintf("%.0f", v) }),
			a.labeledSlider("Delay (ms)", a.delay, func(v float64) string { return fmt.Sprintf("%.0f", v) }),
			a.labeledSlider("Min param score", a.minScore, func(v float64) string { return fmt.Sprintf("%.0f", v) }),
			controls,
		),
	)

	phases := widget.NewCard("Workflow", "",
		container.NewVBox(
			widget.NewLabelWithData(a.phaseBind),
			widget.NewLabelWithData(a.elapsedBind),
			widget.NewLabel("① Crawl  →  ② Keywords  →  ③ SQLi score  →  ④ Dorks"),
		),
	)

	metrics := widget.NewCard("System", "",
		container.NewVBox(
			widget.NewLabelWithData(a.cpuBind),
			widget.NewLabelWithData(a.ramBind),
			widget.NewLabelWithData(a.throttleBind),
			widget.NewLabelWithData(a.statsBind),
		),
	)

	decisionEntry := widget.NewEntryWithData(a.decisionBind)
	decisionEntry.Disable()
	decisionEntry.SetMinRowsVisible(8)
	decisionEntry.Wrapping = fyne.TextWrapWord

	dorkEntry := widget.NewEntryWithData(a.dorkBind)
	dorkEntry.Disable()
	dorkEntry.SetMinRowsVisible(10)
	dorkEntry.Wrapping = fyne.TextWrapWord

	logEntry := widget.NewEntryWithData(a.logBind)
	logEntry.Disable()
	logEntry.SetMinRowsVisible(6)
	logEntry.Wrapping = fyne.TextWrapWord

	domainsCard := widget.NewCard("Domains", "Per-domain progress", a.domainTable)
	decisionsCard := widget.NewCard("Filter decisions", "Last param accept/reject", container.NewScroll(decisionEntry))
	dorksCard := widget.NewCard("Dork preview", "Phase 4 output", container.NewScroll(dorkEntry))
	logCard := widget.NewCard("Log", "", container.NewScroll(logEntry))

	left := container.NewVBox(configCard, phases, metrics)
	center := container.NewVBox(domainsCard, decisionsCard)
	right := container.NewVBox(dorksCard, logCard)

	return container.NewPadded(container.NewBorder(nil, nil, left, right, center))
}

func (a *App) labeledSlider(title string, s *widget.Slider, fmtVal func(float64) string) fyne.CanvasObject {
	val := widget.NewLabel(fmtVal(s.Value))
	s.OnChanged = func(v float64) { val.SetText(fmtVal(v)) }
	return container.NewBorder(nil, nil, widget.NewLabel(title), val, s)
}

func (a *App) onStart() {
	if a.running {
		return
	}
	cfg := config.Default()
	cfg.DomainFile = strings.TrimSpace(a.domainFile.Text)
	cfg.OutputDir = strings.TrimSpace(a.outputDir.Text)
	cfg.Depth = int(a.depth.Value)
	cfg.PageLimit = int(a.pages.Value)
	cfg.Workers = int(a.workers.Value)
	cfg.DelayMS = int(a.delay.Value)
	cfg.MinParamScore = int(a.minScore.Value)
	cfg.StateFile = cfg.OutputDir + "/crawl.state.json"

	if cfg.DomainFile == "" {
		dialog.ShowError(fmt.Errorf("select a domains .txt file"), a.window)
		return
	}
	if err := cfg.Validate(); err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	domains, err := crawler.LoadDomains(cfg.DomainFile)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	events := crawler.Events{
		OnSnapshot: a.applySnapshot,
		OnLog:      a.appendLog,
		OnDorksDone: func(dorksPath string) {
			a.promptSaveDorks(dorksPath, cfg.OutputDir)
		},
	}

	engine, err := crawler.NewWithEvents(cfg, events)
	if err != nil {
		dialog.ShowError(err, a.window)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.running = true
	a.engine = engine
	a.cancel = cancel
	a.mu.Unlock()

	a.setRunningUI(true)
	a.appendLog(fmt.Sprintf("Starting recon on %d domain(s)…", len(domains)))

	go func() {
		err := engine.Run(ctx, domains)
		a.mu.Lock()
		a.running = false
		a.engine = nil
		a.cancel = nil
		a.mu.Unlock()
		a.setRunningUI(false)
		if err != nil {
			dialog.ShowError(err, a.window)
		} else {
			a.appendLog("Recon complete — autres résultats dans " + cfg.OutputDir)
		}
	}()
}

func (a *App) promptSaveDorks(srcPath, outputDir string) {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(fmt.Errorf("save dialog: %w", err), a.window)
			return
		}
		if writer == nil {
			a.appendLog("Enregistrement des dorks annulé")
			dialog.ShowInformation(
				"Terminé",
				"Dorks temporaires dans :\n"+srcPath+"\n\nAutres fichiers dans :\n"+outputDir,
				a.window,
			)
			return
		}
		defer writer.Close()

		data, err := os.ReadFile(srcPath)
		if err != nil {
			dialog.ShowError(fmt.Errorf("read dorks: %w", err), a.window)
			return
		}
		if _, err := writer.Write(data); err != nil {
			dialog.ShowError(fmt.Errorf("write dorks: %w", err), a.window)
			return
		}

		saved := writer.URI().Path()
		if saved == "" {
			saved = writer.URI().String()
		}
		a.appendLog("Dorks enregistrés → " + saved)
		dialog.ShowInformation(
			"Dorks enregistrés",
			"Fichier sauvegardé :\n"+saved+"\n\nKeywords, params, URLs :\n"+outputDir,
			a.window,
		)
	}, a.window)
	d.SetFileName("dorks.txt")
	d.SetFilter(storage.NewExtensionFileFilter([]string{".txt"}))
	d.Show()
}

func (a *App) onPause() {
	a.mu.Lock()
	eng := a.engine
	a.mu.Unlock()
	if eng != nil {
		eng.Pause()
		a.appendLog("Paused")
	}
}

func (a *App) onResume() {
	a.mu.Lock()
	eng := a.engine
	a.mu.Unlock()
	if eng != nil {
		eng.Resume()
		a.appendLog("Resumed")
	}
}

func (a *App) onStop() {
	a.mu.Lock()
	eng := a.engine
	cancel := a.cancel
	a.mu.Unlock()
	if eng != nil {
		eng.Stop()
	}
	if cancel != nil {
		cancel()
	}
	a.appendLog("Stopping…")
}

func (a *App) setRunningUI(running bool) {
	a.startBtn.Disable()
	a.pauseBtn.Disable()
	a.resumeBtn.Disable()
	a.stopBtn.Disable()
	if running {
		a.pauseBtn.Enable()
		a.resumeBtn.Enable()
		a.stopBtn.Enable()
	} else {
		a.startBtn.Enable()
	}
}

func (a *App) applySnapshot(s monitor.UISnapshot) {
	a.mu.Lock()
	a.domainRows = append([]monitor.DomainStatus(nil), s.Domains...)
	a.mu.Unlock()

	_ = a.phaseBind.Set(s.PhaseLabel)
	_ = a.elapsedBind.Set("Elapsed: " + s.Elapsed.String())
	_ = a.cpuBind.Set(fmt.Sprintf("CPU: %.1f%%", s.CPU))
	_ = a.ramBind.Set(fmt.Sprintf("RAM: %.1f%%", s.RAM))
	_ = a.throttleBind.Set("Throttle: " + s.Throttle)
	_ = a.statsBind.Set(fmt.Sprintf("Keywords: %d | Params: %d | Filter: %d/%d | Workers: %d | Delay: %dms",
		s.Keywords, s.Params, s.Accepted, s.Rejected, s.Workers, s.DelayMS))

	var decLines []string
	start := 0
	if len(s.Decisions) > 12 {
		start = len(s.Decisions) - 12
	}
	for _, d := range s.Decisions[start:] {
		flag := "REJECT"
		if d.Accepted {
			flag = "ACCEPT"
		}
		decLines = append(decLines, fmt.Sprintf("[%s] %s score=%d %s — %s", flag, d.Param, d.Score, d.Tier, d.Reason))
	}
	_ = a.decisionBind.Set(strings.Join(decLines, "\n"))

	if s.DorkPreview != "" {
		_ = a.dorkBind.Set(s.DorkPreview)
	}

	a.domainTable.Refresh()
}

func (a *App) appendLog(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)
	cur, _ := a.logBind.Get()
	if cur == "" {
		_ = a.logBind.Set(line)
	} else {
		_ = a.logBind.Set(cur + "\n" + line)
	}
}

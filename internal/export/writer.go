package export

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/xadv404/letter/internal/dorks"
)

// Writer exports only dorks.txt — keywords/params stay in-memory for assembly.
type Writer struct {
	mu          sync.Mutex
	outputDir   string
	dorksPath   string
	dorksFile   *os.File
	dorksWriter *bufio.Writer
	lastFlush   time.Time
	flushEvery  time.Duration
}

func New(outputDir string, resume bool) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	dorksPath := filepath.Join(outputDir, "dorks.txt")
	flags := os.O_CREATE | os.O_WRONLY
	if resume {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	dorksFile, err := os.OpenFile(dorksPath, flags, 0o644)
	if err != nil {
		return nil, err
	}
	return &Writer{
		outputDir:   outputDir,
		dorksPath:   dorksPath,
		dorksFile:   dorksFile,
		dorksWriter: bufio.NewWriter(dorksFile),
		flushEvery:  2 * time.Second,
		lastFlush:   time.Now(),
	}, nil
}

func ts() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// WriteDorks writes the final scored dork list (replaces file).
func (w *Writer) WriteDorks(assembled []dorks.AssembledDork) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.dorksWriter != nil {
		_ = w.dorksWriter.Flush()
	}
	if w.dorksFile != nil {
		_ = w.dorksFile.Close()
	}

	f, err := os.OpenFile(w.dorksPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(f)
	fmt.Fprintf(bw, "# Letter Recon dorks — %s\n", ts())
	fmt.Fprintf(bw, "# score | tier | family | dork\n")
	for _, d := range assembled {
		fmt.Fprintf(bw, "%d | %s | %s | %s\n", d.Score, d.Tier, d.Family, d.Dork)
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	w.dorksFile = f
	w.dorksWriter = bufio.NewWriter(f)
	w.lastFlush = time.Now()
	return nil
}

func (w *Writer) ForceFlush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.dorksWriter != nil {
		return w.dorksWriter.Flush()
	}
	return nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var err error
	if w.dorksWriter != nil {
		if e := w.dorksWriter.Flush(); e != nil {
			err = e
		}
	}
	if w.dorksFile != nil {
		if e := w.dorksFile.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

func (w *Writer) DorksPath() string  { return w.dorksPath }
func (w *Writer) OutputDir() string  { return w.outputDir }

package export

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Writer struct {
	mu          sync.Mutex
	outputDir   string
	dorksPath   string
	dorks       *os.File
	dorkWriter  *bufio.Writer
	dorksHeader bool
	lastFlush   time.Time
	flushEvery  time.Duration
}

func New(outputDir string) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	dorkPath := filepath.Join(outputDir, "dorks.txt")

	dorkFile, err := os.OpenFile(dorkPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	return &Writer{
		outputDir:  outputDir,
		dorksPath:  dorkPath,
		dorks:      dorkFile,
		dorkWriter: bufio.NewWriter(dorkFile),
		flushEvery: 2 * time.Second,
		lastFlush:  time.Now(),
	}, nil
}

func (w *Writer) WriteDork(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.dorksHeader {
		ts := time.Now().UTC().Format(time.RFC3339)
		if _, err := fmt.Fprintf(w.dorkWriter, "# Letter Recon dorks — generated %s\n", ts); err != nil {
			return err
		}
		w.dorksHeader = true
	}
	if _, err := fmt.Fprintln(w.dorkWriter, line); err != nil {
		return err
	}
	return w.maybeFlush()
}

func (w *Writer) maybeFlush() error {
	if time.Since(w.lastFlush) < w.flushEvery {
		return nil
	}
	if err := w.dorkWriter.Flush(); err != nil {
		return err
	}
	w.lastFlush = time.Now()
	return nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var first error
	if err := w.dorkWriter.Flush(); err != nil {
		first = err
	}
	if err := w.dorks.Close(); err != nil && first == nil {
		first = err
	}
	return first
}

// DorksPath returns the path to the generated dorks file.
func (w *Writer) DorksPath() string {
	return w.dorksPath
}

// OutputDir returns the configured output directory.
func (w *Writer) OutputDir() string {
	return w.outputDir
}

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
	mu              sync.Mutex
	outputDir       string
	dorksPath       string
	exploitablePath string
	dorks           *os.File
	exploitable     *os.File
	dorkWriter      *bufio.Writer
	exploitWriter   *bufio.Writer
	dorksHeader     bool
	exploitHeader   bool
	lastFlush       time.Time
	flushEvery      time.Duration
}

func New(outputDir string) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	dorkPath := filepath.Join(outputDir, "dorks.txt")
	exploitPath := filepath.Join(outputDir, "dorks_exploitable.txt")

	dorkFile, err := os.OpenFile(dorkPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	exploitFile, err := os.OpenFile(exploitPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		dorkFile.Close()
		return nil, err
	}

	return &Writer{
		outputDir:       outputDir,
		dorksPath:       dorkPath,
		exploitablePath: exploitPath,
		dorks:           dorkFile,
		exploitable:     exploitFile,
		dorkWriter:      bufio.NewWriter(dorkFile),
		exploitWriter:   bufio.NewWriter(exploitFile),
		flushEvery:      2 * time.Second,
		lastFlush:       time.Now(),
	}, nil
}

func (w *Writer) WriteDork(line string) error {
	return w.writeLine(w.dorks, &w.dorkWriter, &w.dorksHeader, "dorks", line)
}

// WriteExploitableDork writes a high-confidence SQLi dork (run these first on Google).
func (w *Writer) WriteExploitableDork(line string) error {
	return w.writeLine(w.exploitable, &w.exploitWriter, &w.exploitHeader, "exploitable SQLi dorks", line)
}

func (w *Writer) writeLine(f *os.File, bw **bufio.Writer, header *bool, label, line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !*header {
		ts := time.Now().UTC().Format(time.RFC3339)
		if _, err := fmt.Fprintf(*bw, "# Letter Recon %s — generated %s\n# ~50 dorks × ~5-10k URLs = 200-500k URLs target\n", label, ts); err != nil {
			return err
		}
		*header = true
	}
	if _, err := fmt.Fprintln(*bw, line); err != nil {
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
	if err := w.exploitWriter.Flush(); err != nil {
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
	if err := w.exploitWriter.Flush(); err != nil && first == nil {
		first = err
	}
	if err := w.dorks.Close(); err != nil && first == nil {
		first = err
	}
	if err := w.exploitable.Close(); err != nil && first == nil {
		first = err
	}
	return first
}

// DorksPath returns the path to the generated dorks file.
func (w *Writer) DorksPath() string {
	return w.dorksPath
}

// ExploitableDorksPath returns high-confidence SQLi dorks (run first).
func (w *Writer) ExploitableDorksPath() string {
	return w.exploitablePath
}

// OutputDir returns the configured output directory.
func (w *Writer) OutputDir() string {
	return w.outputDir
}

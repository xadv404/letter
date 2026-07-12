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
	keywords    *os.File
	params      *os.File
	dorks       *os.File
	kwWriter    *bufio.Writer
	paramWriter *bufio.Writer
	dorkWriter  *bufio.Writer
	lastFlush   time.Time
	flushEvery  time.Duration
}

func New(outputDir string) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	kwPath := filepath.Join(outputDir, fmt.Sprintf("keywords_%s.txt", ts))
	paramPath := filepath.Join(outputDir, fmt.Sprintf("parameters_%s.txt", ts))
	dorkPath := filepath.Join(outputDir, "dorks.txt")

	kwFile, err := os.OpenFile(kwPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	paramFile, err := os.OpenFile(paramPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		kwFile.Close()
		return nil, err
	}
	dorkFile, err := os.OpenFile(dorkPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		kwFile.Close()
		paramFile.Close()
		return nil, err
	}

	return &Writer{
		outputDir:   outputDir,
		keywords:    kwFile,
		params:      paramFile,
		dorks:       dorkFile,
		kwWriter:    bufio.NewWriter(kwFile),
		paramWriter: bufio.NewWriter(paramFile),
		dorkWriter:  bufio.NewWriter(dorkFile),
		flushEvery:  2 * time.Second,
		lastFlush:   time.Now(),
	}, nil
}

func (w *Writer) WriteKeyword(domain, keyword string, weight int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := fmt.Fprintf(w.kwWriter, "%s\t%s\t%d\n", domain, keyword, weight); err != nil {
		return err
	}
	return w.maybeFlush()
}

func (w *Writer) WriteParameter(domain, url, param string, score int, tier string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := fmt.Fprintf(w.paramWriter, "%s\t%s\t%s\t%d\t%s\n", domain, url, param, score, tier); err != nil {
		return err
	}
	return w.maybeFlush()
}

func (w *Writer) WriteDork(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	ts := time.Now().UTC().Format(time.RFC3339)
	if _, err := fmt.Fprintf(w.dorkWriter, "[%s] %s\n", ts, line); err != nil {
		return err
	}
	return w.maybeFlush()
}

func (w *Writer) maybeFlush() error {
	if time.Since(w.lastFlush) < w.flushEvery {
		return nil
	}
	if err := w.kwWriter.Flush(); err != nil {
		return err
	}
	if err := w.paramWriter.Flush(); err != nil {
		return err
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
	if err := w.kwWriter.Flush(); err != nil && first == nil {
		first = err
	}
	if err := w.paramWriter.Flush(); err != nil && first == nil {
		first = err
	}
	if err := w.dorkWriter.Flush(); err != nil && first == nil {
		first = err
	}
	if err := w.keywords.Close(); err != nil && first == nil {
		first = err
	}
	if err := w.params.Close(); err != nil && first == nil {
		first = err
	}
	if err := w.dorks.Close(); err != nil && first == nil {
		first = err
	}
	return first
}

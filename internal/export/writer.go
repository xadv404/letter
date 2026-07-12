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
	mu           sync.Mutex
	outputDir    string
	keywords     *os.File
	params       *os.File
	urls         *os.File
	targets      *os.File
	dorks        *os.File
	kwWriter     *bufio.Writer
	paramWriter  *bufio.Writer
	urlWriter    *bufio.Writer
	targetWriter *bufio.Writer
	dorkWriter   *bufio.Writer
	dorksHeader  bool
	lastFlush    time.Time
	flushEvery   time.Duration
}

func New(outputDir string) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	ts := time.Now().UTC().Format("2006-01-02T15-04-05Z")
	kwPath := filepath.Join(outputDir, fmt.Sprintf("keywords_%s.txt", ts))
	paramPath := filepath.Join(outputDir, fmt.Sprintf("parameters_%s.txt", ts))
	urlPath := filepath.Join(outputDir, fmt.Sprintf("urls_%s.txt", ts))
	targetPath := filepath.Join(outputDir, fmt.Sprintf("targets_%s.txt", ts))
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
	urlFile, err := os.OpenFile(urlPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		kwFile.Close()
		paramFile.Close()
		return nil, err
	}
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		kwFile.Close()
		paramFile.Close()
		urlFile.Close()
		return nil, err
	}
	dorkFile, err := os.OpenFile(dorkPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		kwFile.Close()
		paramFile.Close()
		urlFile.Close()
		targetFile.Close()
		return nil, err
	}

	return &Writer{
		outputDir:    outputDir,
		keywords:     kwFile,
		params:       paramFile,
		urls:         urlFile,
		targets:      targetFile,
		dorks:        dorkFile,
		kwWriter:     bufio.NewWriter(kwFile),
		paramWriter:  bufio.NewWriter(paramFile),
		urlWriter:    bufio.NewWriter(urlFile),
		targetWriter: bufio.NewWriter(targetFile),
		dorkWriter:   bufio.NewWriter(dorkFile),
		flushEvery:   2 * time.Second,
		lastFlush:    time.Now(),
	}, nil
}

func (w *Writer) WriteURL(domain, rawURL string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := fmt.Fprintf(w.urlWriter, "%s\t%s\n", domain, rawURL); err != nil {
		return err
	}
	return w.maybeFlush()
}

func (w *Writer) WriteTarget(domain, rawURL string, highParams []string, maxScore int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := fmt.Fprintf(w.targetWriter, "%s\t%s\t%d\t%s\n", domain, rawURL, maxScore, join(highParams)); err != nil {
		return err
	}
	return w.maybeFlush()
}

func (w *Writer) WriteKeyword(domain, keyword string, weight int, source string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := fmt.Fprintf(w.kwWriter, "%s\t%s\t%d\t%s\n", domain, keyword, weight, source); err != nil {
		return err
	}
	return w.maybeFlush()
}

func (w *Writer) WriteParameter(domain, rawURL, param string, score int, tier, matched string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := fmt.Fprintf(w.paramWriter, "%s\t%s\t%s\t%d\t%s\t%s\n", domain, rawURL, param, score, tier, matched); err != nil {
		return err
	}
	return w.maybeFlush()
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

func join(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "," + parts[i]
	}
	return out
}

func (w *Writer) maybeFlush() error {
	if time.Since(w.lastFlush) < w.flushEvery {
		return nil
	}
	for _, bw := range []*bufio.Writer{w.kwWriter, w.paramWriter, w.urlWriter, w.targetWriter, w.dorkWriter} {
		if err := bw.Flush(); err != nil {
			return err
		}
	}
	w.lastFlush = time.Now()
	return nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var first error
	for _, bw := range []*bufio.Writer{w.kwWriter, w.paramWriter, w.urlWriter, w.targetWriter, w.dorkWriter} {
		if err := bw.Flush(); err != nil && first == nil {
			first = err
		}
	}
	for _, f := range []*os.File{w.keywords, w.params, w.urls, w.targets, w.dorks} {
		if err := f.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

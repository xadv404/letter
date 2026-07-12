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

type Writer struct {
	mu             sync.Mutex
	outputDir      string
	typesPath      string
	keywordsPath   string
	paramsPath     string
	dorksPath      string
	typesFile      *os.File
	keywordsFile   *os.File
	paramsFile     *os.File
	dorksFile      *os.File
	typesWriter    *bufio.Writer
	keywordsWriter *bufio.Writer
	paramsWriter   *bufio.Writer
	dorksWriter    *bufio.Writer
	typesHeader    bool
	kwHeader       bool
	pmHeader       bool
	dorksHeader    bool
	lastFlush      time.Time
	flushEvery     time.Duration
}

func New(outputDir string) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	typesPath := filepath.Join(outputDir, "dorktypes.txt")
	keywordsPath := filepath.Join(outputDir, "keywords.txt")
	paramsPath := filepath.Join(outputDir, "params.txt")
	dorksPath := filepath.Join(outputDir, "dorks.txt")

	open := func(p string) (*os.File, error) {
		return os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	}
	typesFile, err := open(typesPath)
	if err != nil {
		return nil, err
	}
	kwFile, err := open(keywordsPath)
	if err != nil {
		typesFile.Close()
		return nil, err
	}
	pmFile, err := open(paramsPath)
	if err != nil {
		typesFile.Close()
		kwFile.Close()
		return nil, err
	}
	dorksFile, err := open(dorksPath)
	if err != nil {
		typesFile.Close()
		kwFile.Close()
		pmFile.Close()
		return nil, err
	}

	return &Writer{
		outputDir:      outputDir,
		typesPath:      typesPath,
		keywordsPath:   keywordsPath,
		paramsPath:     paramsPath,
		dorksPath:      dorksPath,
		typesFile:      typesFile,
		keywordsFile:   kwFile,
		paramsFile:     pmFile,
		dorksFile:      dorksFile,
		typesWriter:    bufio.NewWriter(typesFile),
		keywordsWriter: bufio.NewWriter(kwFile),
		paramsWriter:   bufio.NewWriter(pmFile),
		dorksWriter:    bufio.NewWriter(dorksFile),
		flushEvery:     2 * time.Second,
		lastFlush:      time.Now(),
	}, nil
}

// WriteMaterials exports types, keywords, params and auto-assembled dorks.
func (w *Writer) WriteMaterials(m dorks.Materials) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.typesHeader {
		ts := time.Now().UTC().Format(time.RFC3339)
		fmt.Fprintf(w.typesWriter, "# Letter Recon dork types — %s\n", ts)
		fmt.Fprintf(w.typesWriter, "# family | volume | slots | pattern\n")
		w.typesHeader = true
	}
	for _, t := range m.Types {
		slots := joinSlots(t.Slots)
		if _, err := fmt.Fprintf(w.typesWriter, "%s | %s | %s | %s\n", t.Family, t.Volume, slots, t.Pattern); err != nil {
			return err
		}
	}

	if !w.kwHeader {
		ts := time.Now().UTC().Format(time.RFC3339)
		fmt.Fprintf(w.keywordsWriter, "# Letter Recon keywords — %s\n", ts)
		w.kwHeader = true
	}
	for _, kw := range m.Keywords {
		if _, err := fmt.Fprintln(w.keywordsWriter, kw); err != nil {
			return err
		}
	}
	for _, ph := range m.Phrases {
		if _, err := fmt.Fprintf(w.keywordsWriter, "\"%s\"\n", ph); err != nil {
			return err
		}
	}

	if !w.pmHeader {
		ts := time.Now().UTC().Format(time.RFC3339)
		fmt.Fprintf(w.paramsWriter, "# Letter Recon params — %s\n", ts)
		w.pmHeader = true
	}
	for _, pm := range m.Params {
		if _, err := fmt.Fprintln(w.paramsWriter, pm); err != nil {
			return err
		}
	}
	for _, path := range m.Paths {
		if _, err := fmt.Fprintf(w.paramsWriter, "# path:%s\n", path); err != nil {
			return err
		}
	}

	assembled := dorks.AssembleStrings(m)
	if !w.dorksHeader {
		ts := time.Now().UTC().Format(time.RFC3339)
		fmt.Fprintf(w.dorksWriter, "# Letter Recon dorks — %s\n", ts)
		fmt.Fprintf(w.dorksWriter, "# Auto-assembled: dorktypes × keywords × params\n")
		w.dorksHeader = true
	}
	for _, d := range assembled {
		if _, err := fmt.Fprintln(w.dorksWriter, d); err != nil {
			return err
		}
	}

	return w.flushLocked()
}

func joinSlots(slots []string) string {
	s := ""
	for i, x := range slots {
		if i > 0 {
			s += ","
		}
		s += x
	}
	return s
}

func (w *Writer) flushLocked() error {
	if time.Since(w.lastFlush) < w.flushEvery {
		return nil
	}
	for _, bw := range []*bufio.Writer{w.typesWriter, w.keywordsWriter, w.paramsWriter, w.dorksWriter} {
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
	for _, bw := range []*bufio.Writer{w.typesWriter, w.keywordsWriter, w.paramsWriter, w.dorksWriter} {
		if err := bw.Flush(); err != nil && first == nil {
			first = err
		}
	}
	for _, f := range []*os.File{w.typesFile, w.keywordsFile, w.paramsFile, w.dorksFile} {
		if err := f.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (w *Writer) DorksPath() string     { return w.dorksPath }
func (w *Writer) TypesPath() string     { return w.typesPath }
func (w *Writer) KeywordsPath() string  { return w.keywordsPath }
func (w *Writer) ParamsPath() string    { return w.paramsPath }
func (w *Writer) OutputDir() string     { return w.outputDir }

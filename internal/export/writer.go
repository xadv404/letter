package export

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/xadv404/letter/internal/config"
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
	kwExported     int
	kwSeen         map[string]struct{}
	pmExported     int
	pmSeen         map[string]struct{}
	lastFlush      time.Time
	flushEvery     time.Duration
}

func New(outputDir string, resume bool) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	typesPath := filepath.Join(outputDir, "dorktypes.txt")
	keywordsPath := filepath.Join(outputDir, "keywords.txt")
	paramsPath := filepath.Join(outputDir, "params.txt")
	dorksPath := filepath.Join(outputDir, "dorks.txt")

	open := func(p string) (*os.File, error) {
		flags := os.O_CREATE | os.O_WRONLY
		if resume {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
		return os.OpenFile(p, flags, 0o644)
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

	w := &Writer{
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
		kwSeen:         map[string]struct{}{},
		pmSeen:         map[string]struct{}{},
		flushEvery:     2 * time.Second,
		lastFlush:      time.Now(),
	}

	if resume {
		w.kwHeader = fileHasContent(keywordsPath)
		w.pmHeader = fileHasContent(paramsPath)
		w.typesHeader = fileHasContent(typesPath)
		w.dorksHeader = fileHasContent(dorksPath)
	}
	return w, nil
}

func fileHasContent(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

func ts() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// WriteKeywordIncremental appends a high-weight keyword during crawl (capped).
func (w *Writer) WriteKeywordIncremental(keyword string, weight int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if weight < config.MinKeywordExportWeight || w.kwExported >= config.MaxExportKeywords {
		return nil
	}
	keyword = trimKey(keyword)
	if keyword == "" {
		return nil
	}
	if _, ok := w.kwSeen[keyword]; ok {
		return nil
	}
	w.kwSeen[keyword] = struct{}{}
	w.kwExported++

	if !w.kwHeader {
		if _, err := fmt.Fprintf(w.keywordsWriter, "# Letter Recon keywords — %s\n", ts()); err != nil {
			return err
		}
		fmt.Fprintf(w.keywordsWriter, "# weight | keyword (live, max %d)\n", config.MaxExportKeywords)
		w.kwHeader = true
	}
	_, err := fmt.Fprintf(w.keywordsWriter, "%d\t%s\n", weight, keyword)
	return err
}

// WriteParamIncremental appends a scored parameter during crawl (deduped, capped).
func (w *Writer) WriteParamIncremental(name string, score int, tier string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.pmExported >= config.MaxExportParams {
		return nil
	}
	name = trimKey(name)
	if name == "" {
		return nil
	}
	if _, ok := w.pmSeen[name]; ok {
		return nil
	}
	w.pmSeen[name] = struct{}{}
	w.pmExported++

	if !w.pmHeader {
		if _, err := fmt.Fprintf(w.paramsWriter, "# Letter Recon params — %s\n", ts()); err != nil {
			return err
		}
		w.pmHeader = true
	}
	_, err := fmt.Fprintf(w.paramsWriter, "%s\t%s\t%d\t%s\n", ts(), name, score, tier)
	return err
}

// WriteFinalExport rewrites keywords, params and dorks with curated final output.
func (w *Writer) WriteFinalExport(m dorks.Materials, assembled []dorks.AssembledDork, rankedKeywords []KeywordExport) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.rewriteKeywords(rankedKeywords, m.Phrases); err != nil {
		return err
	}
	if err := w.rewriteParams(m.Params, m.Paths); err != nil {
		return err
	}
	if err := w.rewriteDorks(assembled); err != nil {
		return err
	}
	if !w.typesHeader {
		fmt.Fprintf(w.typesWriter, "# Letter Recon dork types — %s\n", ts())
		fmt.Fprintf(w.typesWriter, "# family | volume | slots | pattern\n")
		w.typesHeader = true
	}
	for _, t := range m.Types {
		slots := joinSlots(t.Slots)
		if _, err := fmt.Fprintf(w.typesWriter, "%s | %s | %s | %s\n", t.Family, t.Volume, slots, t.Pattern); err != nil {
			return err
		}
	}
	return w.flushLocked()
}

// KeywordExport is a ranked keyword for final export.
type KeywordExport struct {
	Keyword string
	Weight  int
}

func (w *Writer) rewriteKeywords(ranked []KeywordExport, phrases []string) error {
	if w.keywordsWriter != nil {
		_ = w.keywordsWriter.Flush()
	}
	if w.keywordsFile != nil {
		_ = w.keywordsFile.Close()
	}
	f, err := os.OpenFile(w.keywordsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(f)
	fmt.Fprintf(bw, "# Letter Recon keywords — %s\n", ts())
	fmt.Fprintf(bw, "# weight | keyword (top %d ranked)\n", config.MaxExportKeywords)
	for _, r := range ranked {
		fmt.Fprintf(bw, "%d\t%s\n", r.Weight, r.Keyword)
	}
	for _, ph := range phrases {
		fmt.Fprintf(bw, "%d\t\"%s\"\n", 0, ph)
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	w.keywordsFile = f
	w.keywordsWriter = bufio.NewWriter(f)
	w.kwHeader = true
	return nil
}

func (w *Writer) rewriteParams(params, paths []string) error {
	if w.paramsWriter != nil {
		_ = w.paramsWriter.Flush()
	}
	if w.paramsFile != nil {
		_ = w.paramsFile.Close()
	}
	f, err := os.OpenFile(w.paramsPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(f)
	fmt.Fprintf(bw, "# Letter Recon params — %s\n", ts())
	fmt.Fprintf(bw, "# top %d injectable params\n", config.MaxExportParams)
	for _, pm := range params {
		fmt.Fprintln(bw, pm)
	}
	for _, path := range paths {
		fmt.Fprintf(bw, "#path:%s\n", path)
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	w.paramsFile = f
	w.paramsWriter = bufio.NewWriter(f)
	w.pmHeader = true
	return nil
}

func (w *Writer) rewriteDorks(assembled []dorks.AssembledDork) error {
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
	fmt.Fprintf(bw, "# score | tier | family | dork (%d curated)\n", len(assembled))
	for _, d := range assembled {
		fmt.Fprintf(bw, "%d | %s | %s | %s\n", d.Score, d.Tier, d.Family, d.Dork)
	}
	if err := bw.Flush(); err != nil {
		f.Close()
		return err
	}
	w.dorksFile = f
	w.dorksWriter = bufio.NewWriter(f)
	w.dorksHeader = true
	return nil
}

// WriteMaterials is deprecated — use WriteFinalExport.
func (w *Writer) WriteMaterials(m dorks.Materials) error {
	assembled := dorks.RankAssembled(m)
	kw := make([]KeywordExport, len(m.Keywords))
	for i, k := range m.Keywords {
		kw[i] = KeywordExport{Keyword: k, Weight: m.KeywordScores[k]}
	}
	return w.WriteFinalExport(m, assembled, kw)
}

func trimKey(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 100 {
		return s[:100]
	}
	return s
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

func (w *Writer) ForceFlush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flushAll()
}

func (w *Writer) flushLocked() error {
	if time.Since(w.lastFlush) < w.flushEvery {
		return nil
	}
	return w.flushAll()
}

func (w *Writer) flushAll() error {
	for _, bw := range []*bufio.Writer{w.typesWriter, w.keywordsWriter, w.paramsWriter, w.dorksWriter} {
		if bw != nil {
			if err := bw.Flush(); err != nil {
				return err
			}
		}
	}
	w.lastFlush = time.Now()
	return nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var first error
	if err := w.flushAll(); err != nil && first == nil {
		first = err
	}
	for _, f := range []*os.File{w.typesFile, w.keywordsFile, w.paramsFile, w.dorksFile} {
		if f != nil {
			if err := f.Close(); err != nil && first == nil {
				first = err
			}
		}
	}
	return first
}

func (w *Writer) DorksPath() string    { return w.dorksPath }
func (w *Writer) TypesPath() string    { return w.typesPath }
func (w *Writer) KeywordsPath() string { return w.keywordsPath }
func (w *Writer) ParamsPath() string   { return w.paramsPath }
func (w *Writer) OutputDir() string    { return w.outputDir }

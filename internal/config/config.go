package config

import (
	"fmt"
	"time"
)

const (
	MinDepth     = 1
	MaxDepth     = 10
	MinPageLimit = 10
	MaxPageLimit = 500
	// MaxKeywords — in-memory extraction ceiling per session.
	MaxKeywords = 15_000
	MaxParams   = 10_000
	// MaxExportKeywords — lines written to keywords.txt (ranked, recon-ready).
	MaxExportKeywords = 500
	// MaxExportParams — unique params written to params.txt.
	MaxExportParams = 500
	// MinKeywordExportWeight — ignore low-weight noise during live export.
	MinKeywordExportWeight = 8
)

type CrawlConfig struct {
	Domains       []string
	DomainFile    string
	Depth         int
	PageLimit     int
	Workers       int
	DelayMS       int
	TimeoutPerDom time.Duration
	OutputDir     string
	MinParamScore int
	UserAgent     string
	StateFile     string
}

func Default() CrawlConfig {
	return CrawlConfig{
		Depth:         3,
		PageLimit:     100,
		Workers:       8,
		DelayMS:       250,
		TimeoutPerDom: 5 * time.Minute,
		OutputDir:     "./output",
		MinParamScore: 65,
		UserAgent:     "LetterRecon/1.0 (+https://github.com/xadv404/letter)",
		StateFile:     "./output/crawl.state.json",
	}
}

func (c *CrawlConfig) Validate() error {
	if c.Depth < MinDepth || c.Depth > MaxDepth {
		return fmt.Errorf("depth must be between %d and %d", MinDepth, MaxDepth)
	}
	if c.PageLimit < MinPageLimit || c.PageLimit > MaxPageLimit {
		return fmt.Errorf("page limit must be between %d and %d", MinPageLimit, MaxPageLimit)
	}
	if c.Workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}
	if len(c.Domains) == 0 && c.DomainFile == "" {
		return fmt.Errorf("provide domains via -domains or -domains-file")
	}
	return nil
}

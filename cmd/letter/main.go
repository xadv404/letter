package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/xadv404/letter/internal/config"
	"github.com/xadv404/letter/internal/crawler"
	"github.com/xadv404/letter/internal/license"
)

func main() {
	cfg := config.Default()

	domainsFlag := flag.String("domains", "", "Comma-separated list of domains")
	domainFile := flag.String("domains-file", "", "Path to .txt file with one domain per line")
	depth := flag.Int("depth", cfg.Depth, "Crawl depth (1-10)")
	pageLimit := flag.Int("pages", cfg.PageLimit, "Max pages per domain (10-500)")
	workers := flag.Int("workers", cfg.Workers, "Concurrent domain workers")
	delay := flag.Int("delay", cfg.DelayMS, "Base delay between requests (ms)")
	output := flag.String("output", cfg.OutputDir, "Output directory")
	minScore := flag.Int("min-param-score", cfg.MinParamScore, "Minimum parameter score (50-100)")
	previewOnly := flag.Bool("preview-dorks", false, "Preview dork generation without crawling")
	skipLicense := flag.Bool("skip-license", true, "Skip license validation (dev mode)")

	flag.Parse()

	cfg.DomainFile = *domainFile
	cfg.Depth = *depth
	cfg.PageLimit = *pageLimit
	cfg.Workers = *workers
	cfg.DelayMS = *delay
	cfg.OutputDir = *output
	cfg.MinParamScore = *minScore
	cfg.StateFile = *output + "/crawl.state.json"

	if *domainsFlag != "" {
		for _, d := range strings.Split(*domainsFlag, ",") {
			d = strings.TrimSpace(d)
			if d != "" {
				cfg.Domains = append(cfg.Domains, d)
			}
		}
	}
	if cfg.DomainFile != "" {
		fromFile, err := crawler.LoadDomains(cfg.DomainFile)
		if err != nil {
			exitErr("load domains file: %v", err)
		}
		cfg.Domains = append(cfg.Domains, fromFile...)
	}

	if err := cfg.Validate(); err != nil {
		exitErr("%v", err)
	}

	if !*skipLicense {
		v := license.New("dev-secret-change-me")
		status, err := v.Cached()
		if err != nil || status != license.Active {
			exitErr("license invalid: %v (HWID=%s)", err, license.HWID())
		}
	}

	if *previewOnly {
		fmt.Println("Dork preview mode — provide domains and use full crawl for live extraction.")
		return
	}

	engine, err := crawler.New(cfg)
	if err != nil {
		exitErr("init engine: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nGraceful shutdown (30s)...")
		engine.Stop()
		cancel()
	}()

	fmt.Printf("Letter Recon — crawling %d domain(s)\n", len(cfg.Domains))
	fmt.Printf("Config: depth=%d pages=%d workers=%d delay=%dms output=%s\n\n",
		cfg.Depth, cfg.PageLimit, cfg.Workers, cfg.DelayMS, cfg.OutputDir)

	if err := engine.Run(ctx, cfg.Domains); err != nil {
		exitErr("crawl failed: %v", err)
	}
	fmt.Println("\nCrawl complete. Results in", cfg.OutputDir)
}

func exitErr(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

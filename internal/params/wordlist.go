package params

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const cacheFileName = "seclists_burp_params.cache"
const cacheTTL = 7 * 24 * time.Hour

// LoadWordlist loads the Burp parameter wordlist from cache, embedded data, or a refresh.
func LoadWordlist(cacheDir string) map[string]struct{} {
	if cacheDir != "" {
		if wl, ok := loadFromCache(filepath.Join(cacheDir, cacheFileName)); ok {
			return wl
		}
	}

	wl := parseWordlist(burpParamsRaw)
	if len(wl) == 0 {
		wl = buildSecListsFallback()
	}

	if cacheDir != "" {
		_ = writeCache(filepath.Join(cacheDir, cacheFileName), burpParamsRaw)
	}
	return wl
}

func parseWordlist(raw string) map[string]struct{} {
	set := make(map[string]struct{})
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		name := strings.ToLower(strings.TrimSpace(sc.Text()))
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		set[name] = struct{}{}
	}
	return set
}

func loadFromCache(path string) (map[string]struct{}, bool) {
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > cacheTTL {
		return nil, false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	wl := parseWordlist(string(raw))
	if len(wl) == 0 {
		return nil, false
	}
	return wl, true
}

func writeCache(path, raw string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0o644)
}

func WordlistSize(wl map[string]struct{}) int {
	return len(wl)
}

func buildSecListsFallback() map[string]struct{} {
	names := []string{
		"id", "user", "username", "password", "email", "search", "query", "q", "page", "cat",
		"category", "product", "item", "order", "sort", "filter", "file", "path", "dir", "doc",
		"document", "view", "action", "cmd", "command", "exec", "module", "name", "type", "table",
		"column", "field", "select", "where", "report", "download", "export", "import", "debug",
		"test", "admin", "role", "group", "account", "invoice", "payment", "amount", "price",
		"year", "month", "day", "date", "from", "to", "limit", "offset", "start", "end",
	}
	m := make(map[string]struct{}, len(names))
	for _, n := range names {
		m[n] = struct{}{}
	}
	return m
}

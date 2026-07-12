package keywords

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var suggestClient = &http.Client{Timeout: 6 * time.Second}

// ExpandAutocomplete enriches terms using Google and Bing suggest APIs.
func ExpandAutocomplete(terms []string, limitPerTerm int) []string {
	if limitPerTerm <= 0 {
		limitPerTerm = 5
	}
	seen := map[string]struct{}{}
	var out []string
	for i, term := range terms {
		if i >= 30 {
			break
		}
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		for _, s := range append(suggestGoogle(term), suggestBing(term)...) {
			s = strings.ToLower(strings.TrimSpace(s))
			if s == "" || len(s) < 3 {
				continue
			}
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
			if len(out) >= limitPerTerm*(i+1) {
				break
			}
		}
	}
	return out
}

func suggestGoogle(q string) []string {
	u := "https://suggestqueries.google.com/complete/search?client=firefox&q=" + url.QueryEscape(q)
	body, err := fetchSuggest(u)
	if err != nil {
		return nil
	}
	var parsed []any
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed) < 2 {
		return nil
	}
	arr, _ := parsed[1].([]any)
	var out []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func suggestBing(q string) []string {
	u := "https://api.bing.com/osjson.aspx?query=" + url.QueryEscape(q)
	body, err := fetchSuggest(u)
	if err != nil {
		return nil
	}
	var parsed []any
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed) < 2 {
		return nil
	}
	arr, _ := parsed[1].([]any)
	var out []string
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func fetchSuggest(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LetterRecon/1.2 (+https://github.com/xadv404/letter)")
	resp, err := suggestClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 64<<10))
}

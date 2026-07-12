package keywords

import (
	"bufio"
	_ "embed"
	"strings"
)

//go:embed data/stopwords_en.txt
var stopwordsRaw string

//go:embed data/technical_en.txt
var technicalRaw string

func loadWordSet(raw string) map[string]struct{} {
	set := make(map[string]struct{})
	sc := bufio.NewScanner(strings.NewReader(raw))
	for sc.Scan() {
		w := strings.TrimSpace(strings.ToLower(sc.Text()))
		if w == "" || strings.HasPrefix(w, "#") {
			continue
		}
		set[w] = struct{}{}
	}
	return set
}

func StopwordCount() int {
	return len(loadWordSet(stopwordsRaw))
}

func TechnicalCount() int {
	return len(loadWordSet(technicalRaw))
}

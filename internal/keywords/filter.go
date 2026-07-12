package keywords

import (
	"regexp"
	"strings"
	"unicode"
)

const (
	MinKeywordLen = 2
	MaxKeywordLen = 100
)

var (
	pureNumber   = regexp.MustCompile(`^\d+$`)
	noLetter     = regexp.MustCompile(`^[^a-z]+$`)
	hexLike      = regexp.MustCompile(`^[0-9a-f]{8,}$`)
	uuidLike     = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	cssClassLike = regexp.MustCompile(`^(col|row|btn|nav|container|wrapper|flex|grid|md-|lg-|sm-|xs-|xl-|pt-|pb-|pl-|pr-|mt-|mb-|ml-|mr-|px-|py-|mx-|my-|w-|h-|text-|bg-|border-|rounded-|shadow-|font-|leading-|tracking-|opacity-|z-|top-|bottom-|left-|right-|absolute|relative|fixed|sticky|block|inline|hidden|visible|overflow-|justify-|items-|self-|gap-|space-)`)
)

type Filter struct {
	stopwords   map[string]struct{}
	technical   map[string]struct{}
	boilerplate map[string]struct{}
}

func NewFilter() *Filter {
	return &Filter{
		stopwords:   loadWordSet(stopwordsRaw),
		technical:   loadWordSet(technicalRaw),
		boilerplate: buildBoilerplate(),
	}
}

func (f *Filter) IsStopword(word string) bool {
	_, ok := f.stopwords[strings.ToLower(word)]
	return ok
}

func (f *Filter) IsTechnical(word string) bool {
	_, ok := f.technical[strings.ToLower(word)]
	return ok
}

func (f *Filter) AcceptToken(token string) bool {
	token = strings.ToLower(strings.TrimSpace(token))
	if len(token) < MinKeywordLen || len(token) > MaxKeywordLen {
		return false
	}
	if f.IsStopword(token) || f.IsTechnical(token) {
		return false
	}
	if _, ok := f.boilerplate[token]; ok {
		return false
	}
	if pureNumber.MatchString(token) || noLetter.MatchString(token) {
		return false
	}
	if hasRepeatedChars(token, 4) || hexLike.MatchString(token) || uuidLike.MatchString(token) {
		return false
	}
	if cssClassLike.MatchString(token) {
		return false
	}
	if !hasLetter(token) {
		return false
	}
	return true
}

func (f *Filter) AcceptPhrase(phrase string) bool {
	phrase = normalizePhrase(phrase)
	if len(phrase) < MinKeywordLen || len(phrase) > MaxKeywordLen {
		return false
	}
	tokens := strings.Fields(phrase)
	if len(tokens) == 0 {
		return false
	}

	substantive := 0
	for _, t := range tokens {
		if f.AcceptToken(t) {
			substantive++
		}
	}
	if substantive == 0 {
		return false
	}
	// reject phrases that are only stopwords/technical
	allNoise := true
	for _, t := range tokens {
		lower := strings.ToLower(t)
		if !f.IsStopword(lower) && !f.IsTechnical(lower) {
			allNoise = false
			break
		}
	}
	return !allNoise
}

func normalizePhrase(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ' ' {
			return r
		}
		return ' '
	}, s)
}

func hasRepeatedChars(s string, n int) bool {
	if n <= 1 || len(s) < n {
		return false
	}
	count := 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			count++
			if count >= n {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func buildBoilerplate() map[string]struct{} {
	words := []string{
		"click", "here", "read", "more", "learn", "submit", "subscribe", "unsubscribe",
		"share", "tweet", "follow", "like", "comment", "reply", "view", "show", "hide",
		"toggle", "expand", "collapse", "close", "open", "back", "next", "previous", "prev",
		"continue", "skip", "cancel", "confirm", "accept", "decline", "agree", "disagree",
		"yes", "no", "ok", "okay", "thanks", "thank", "hello", "hi", "hey", "bye", "goodbye",
		"welcome", "home", "menu", "navigation", "nav", "sidebar", "footer", "header", "top",
		"bottom", "left", "right", "center", "middle", "loading", "load", "please", "wait",
		"error", "success", "warning", "info", "notice", "alert", "message", "notification",
		"required", "optional", "field", "input", "output", "form", "button", "link", "image",
		"video", "audio", "photo", "picture", "icon", "logo", "banner", "advertisement", "ad",
		"sponsored", "promoted", "featured", "popular", "trending", "latest", "recent", "updated",
		"posted", "published", "author", "editor", "admin", "administrator", "moderator", "user",
		"member", "guest", "visitor", "account", "profile", "settings", "preferences", "options",
		"help", "support", "faq", "faqs", "guide", "tutorial", "documentation", "docs", "manual",
		"copyright", "rights", "reserved", "trademark", "patent", "disclaimer", "liability",
		"privacy", "policy", "policies", "terms", "conditions", "legal", "compliance", "gdpr",
		"ccpa", "cookies", "consent", "preferences", "contact", "email", "phone", "address",
		"location", "hours", "schedule", "calendar", "date", "time", "year", "month", "day",
		"week", "hour", "minute", "second", "am", "pm", "today", "tomorrow", "yesterday",
		"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday",
		"january", "february", "march", "april", "may", "june", "july", "august",
		"september", "october", "november", "december",
	}
	m := make(map[string]struct{}, len(words))
	for _, w := range words {
		m[w] = struct{}{}
	}
	return m
}

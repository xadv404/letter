package keywords

import (
	"math"
	"strings"
	"unicode"
)

// reconSignals boost terms that correlate with injectable app surfaces.
var reconSignals = map[string]struct{}{
	"invoice": {}, "invoices": {}, "billing": {}, "payment": {}, "payments": {},
	"order": {}, "orders": {}, "checkout": {}, "cart": {}, "basket": {},
	"catalog": {}, "catalogue": {}, "product": {}, "products": {}, "inventory": {},
	"patient": {}, "patients": {}, "prescription": {}, "diagnosis": {}, "medical": {},
	"student": {}, "students": {}, "enrollment": {}, "registration": {}, "register": {},
	"booking": {}, "reservation": {}, "ticket": {}, "tickets": {}, "flight": {},
	"shipment": {}, "tracking": {}, "delivery": {}, "warehouse": {},
	"report": {}, "reports": {}, "query": {}, "search": {}, "filter": {}, "filters": {},
	"category": {}, "categories": {}, "department": {}, "departments": {},
	"customer": {}, "customers": {}, "client": {}, "clients": {}, "vendor": {}, "supplier": {},
	"transaction": {}, "transactions": {}, "statement": {}, "statements": {},
	"document": {}, "documents": {}, "record": {}, "records": {}, "archive": {},
	"download": {}, "export": {}, "import": {}, "upload": {},
	"subscription": {}, "license": {}, "serial": {}, "reference": {},
	"confirmation": {}, "receipt": {}, "quote": {}, "estimate": {},
	"database": {}, "backup": {}, "restore": {}, "vulnerability": {},
	"portal": {}, "dashboard": {}, "panel": {}, "management": {},
}

func expandCompoundToken(tok string) []string {
	tok = strings.TrimSpace(tok)
	if tok == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var parts []string

	add := func(s string) {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return
		}
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			parts = append(parts, s)
		}
	}

	for _, chunk := range strings.FieldsFunc(tok, func(r rune) bool {
		return r == '_' || r == '-' || r == '.'
	}) {
		add(chunk)
		for _, sub := range splitCamelCase(chunk) {
			add(sub)
		}
	}
	if len(parts) == 0 {
		add(tok)
	}
	return parts
}

func splitCamelCase(s string) []string {
	if s == "" {
		return nil
	}
	var parts []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			parts = append(parts, b.String())
			b.Reset()
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 {
			prev := runes[i-1]
			if unicode.IsLower(prev) || (i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
				flush()
			}
		}
		b.WriteRune(unicode.ToLower(r))
	}
	flush()
	if len(parts) <= 1 {
		return nil
	}
	return parts
}

func reconWeightMultiplier(phrase string) float64 {
	tokens := strings.Fields(strings.ToLower(phrase))
	if len(tokens) == 0 {
		return 1
	}
	hits := 0
	for _, t := range tokens {
		if _, ok := reconSignals[t]; ok {
			hits++
		}
	}
	if hits == 0 {
		return 1
	}
	return 1 + float64(hits)*0.35
}

func phraseWeightMultiplier(phrase string) float64 {
	n := len(strings.Fields(phrase))
	switch {
	case n >= 3:
		return 2.4
	case n == 2:
		return 1.85
	default:
		return 1
	}
}

func intelligentRank(baseScore, pageHits int, keyword string) float64 {
	if baseScore <= 0 {
		return 0
	}
	pageBoost := 1 + math.Min(float64(pageHits), 8)*0.12
	return float64(baseScore) * phraseWeightMultiplier(keyword) * reconWeightMultiplier(keyword) * pageBoost
}

func isSubsumedBy(shorter, longer string) bool {
	if shorter == longer || !strings.Contains(longer, " ") {
		return false
	}
	return strings.Contains(longer, shorter)
}

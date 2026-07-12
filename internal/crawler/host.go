package crawler

import "strings"

// NormalizeHost canonicalizes a hostname for state and keyword indexing.
func NormalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if strings.HasPrefix(host, "www.") {
		return host[4:]
	}
	return host
}

package nginx

import "strings"

func isTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

func normalizeRewrite(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "/"
	}
	if strings.HasPrefix(v, "/$") {
		return "/"
	}
	return v
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func strPtr(s string) *string { return &s }

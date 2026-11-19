// Package privacy enforces the personal deployment's no-content-retention policy.
// The server enables it before initialization; there is deliberately no runtime,
// environment, database or API switch that can turn it off again.
package privacy

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
)

var enforced atomic.Bool

var ErrDisabled = errors.New("feature disabled by personal privacy policy")

var accountHTTPStatus = regexp.MustCompile(`(?i)^(?:(?:api returned|http|upstream|access forbidden|status)(?:\s+status)?[\s:(]*)?([45][0-9]{2})\b`)

var accountReasons = []struct {
	marker string
	reason string
}{
	{"deactivated_workspace", "workspace deactivated"},
	{"workspace deactivated", "workspace deactivated"},
	{"invalid_grant", "invalid OAuth grant"},
	{"invalid oauth grant", "invalid OAuth grant"},
	{"invalid_client", "invalid OAuth client"},
	{"invalid oauth client", "invalid OAuth client"},
	{"token expired", "token expired"},
	{"expired_token", "token expired"},
	{"invalid_api_key", "invalid API key"},
	{"invalid api key", "invalid API key"},
	{"insufficient_quota", "quota exhausted"},
	{"quota exceeded", "quota exhausted"},
	{"quota exhausted", "quota exhausted"},
	{"rate limit", "rate limited"},
	{"rate_limit_exceeded", "rate limited"},
	{"permission_denied", "permission denied"},
	{"permission denied", "permission denied"},
	{"account suspended", "account suspended"},
	{"account deactivated", "account deactivated"},
	{"invalid credentials", "invalid credentials"},
	{"token revoked", "token revoked"},
	{"context deadline exceeded", "request timed out"},
	{"request timed out", "request timed out"},
	{"timed out", "request timed out"},
	{"connection refused", "connection refused"},
	{"no such host", "DNS lookup failed"},
	{"dns lookup failed", "DNS lookup failed"},
	{"proxy authentication required", "proxy authentication required"},
	{"network unreachable", "network unreachable"},
	{"repeated upstream 500 errors", "repeated upstream 500 errors"},
	{"unauthorized", "unauthorized"},
	{"forbidden", "access forbidden"},
}

func Enforce()      { enforced.Store(true) }
func Enabled() bool { return enforced.Load() }

// Diagnostic retains only a coarse classification. Never persist provider-supplied
// text: even an error can echo a complete user prompt.
func Diagnostic(value string) string {
	if !Enabled() || value == "" {
		return value
	}
	for _, prefix := range []string{"token refresh retry exhausted:", "401", "403", "429"} {
		if strings.HasPrefix(value, prefix) {
			return prefix + " [details omitted]"
		}
	}
	return "[details omitted by privacy policy]"
}

// AccountDiagnostic retains an actionable, fixed-vocabulary reason for the
// account status tooltip. Upstream error bodies can echo prompts or secrets,
// so no provider-supplied free text is copied into persistent account state.
func AccountDiagnostic(value string) string {
	if !Enabled() || value == "" {
		return value
	}
	if value == "Account error (cause could not be classified)" {
		return value
	}

	lower := strings.ToLower(value)
	status := ""
	if match := accountHTTPStatus.FindStringSubmatch(strings.TrimSpace(value)); match != nil {
		status = "HTTP " + match[1]
	}
	reason := ""
	switch {
	case strings.HasPrefix(lower, "privacy not set"):
		reason = "privacy not set for required group"
	case strings.HasPrefix(lower, "internal 500 consecutive failures"):
		reason = "repeated upstream 500 errors"
		status = "HTTP 500"
	default:
		for _, candidate := range accountReasons {
			if strings.Contains(lower, candidate.marker) {
				reason = candidate.reason
				break
			}
		}
	}
	if strings.HasPrefix(lower, "token refresh failed (non-retryable)") {
		if reason != "" {
			reason = "token refresh failed (non-retryable): " + reason
		} else {
			reason = "token refresh failed (non-retryable)"
		}
	}
	if status != "" && reason != "" {
		return fmt.Sprintf("%s: %s", status, reason)
	}
	if status != "" {
		code, _ := strconv.Atoi(status[len("HTTP "):])
		if code == http.StatusForbidden {
			return status + ": access forbidden"
		}
		if code == http.StatusUnauthorized {
			return status + ": unauthorized"
		}
		return status + ": " + http.StatusText(code)
	}
	if reason != "" {
		return reason
	}
	return "Account error (cause could not be classified)"
}

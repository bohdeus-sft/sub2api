// Package privacy enforces the personal deployment's no-content-retention policy.
// The server enables it before initialization; there is deliberately no runtime,
// environment, database or API switch that can turn it off again.
package privacy

import (
	"errors"
	"strings"
	"sync/atomic"
)

var enforced atomic.Bool

var ErrDisabled = errors.New("feature disabled by personal privacy policy")

func Enforce()      { enforced.Store(true) }
func Enabled() bool { return enforced.Load() }

// Diagnostic preserves only a classification needed by account recovery. Never
// persist provider-supplied text: even an error can echo a complete user prompt.
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

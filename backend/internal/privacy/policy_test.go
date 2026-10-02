package privacy_test

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/privacy"
	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/stretchr/testify/require"
)

func TestAccountDiagnostic(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		for _, tc := range []struct {
			input string
			want  string
		}{
			{"API returned 403: private prompt in response", "HTTP 403: access forbidden"},
			{"Access forbidden (403): account suspended; access_token=secret", "HTTP 403: account suspended"},
			{"HTTP 402: {\"detail\":{\"code\":\"deactivated_workspace\",\"message\":\"private prompt\"}}", "HTTP 402: workspace deactivated"},
			{"Token refresh failed (non-retryable): invalid_grant refresh_token=secret", "token refresh failed (non-retryable): invalid OAuth grant"},
			{"Request failed: dial tcp: connection refused; access_token=secret", "connection refused"},
			{"Token refresh failed (non-retryable): context deadline exceeded", "token refresh failed (non-retryable): request timed out"},
			{"Privacy not set, required by group [private group]", "privacy not set for required group"},
			{"INTERNAL 500 consecutive failures: 3 rounds", "HTTP 500: repeated upstream 500 errors"},
			{"canary-private-prompt", "Account error (cause could not be classified)"},
			{"Account error (cause could not be classified)", "Account error (cause could not be classified)"},
		} {
			got := privacy.AccountDiagnostic(tc.input)
			require.Equal(t, tc.want, got)
			require.Equal(t, got, privacy.AccountDiagnostic(got), "cache reclassification must be stable")
			require.NotContains(t, strings.ToLower(got), "private")
			require.NotContains(t, got, "secret")
		}
		require.Empty(t, privacy.AccountDiagnostic(""))
		require.Equal(t, "[details omitted by privacy policy]", privacy.Diagnostic("private prompt"), "other retained diagnostics stay strict")
	})
}

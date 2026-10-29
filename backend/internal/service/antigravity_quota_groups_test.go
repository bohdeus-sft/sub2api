//go:build unit

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/stretchr/testify/require"
)

func TestAntigravitySummaryGroupsPreservePoolsAndUnknownValues(t *testing.T) {
	var summary antigravity.QuotaSummary
	require.NoError(t, json.Unmarshal([]byte(`{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","remainingFraction":0,"resetTime":"2026-10-29T23:00:00+03:00"},{"bucketId":"gemini-weekly","remainingFraction":1}]}],"buckets":[{"bucketId":"gemini-weekly","remainingFraction":0.1},{"bucketId":"3p-5h","remainingFraction":0.4},{"bucketId":"3p-weekly","remainingFraction":null,"resetTime":"invalid"},{"bucketId":"gemini-argon-wtu-weekly","remainingFraction":0.5},{"bucketId":"future-pool","window":"monthly","remainingFraction":2,"disabled":true},{"remainingFraction":1}]}`), &summary))
	groups := antigravitySummaryGroups(&summary)
	require.Len(t, groups, 4)
	require.Equal(t, "Gemini Models", groups[0].DisplayName)
	require.Equal(t, "5h", groups[0].Buckets[0].Window)
	require.Equal(t, 100.0, *groups[0].Buckets[0].Utilization)
	require.Equal(t, "2026-10-29T20:00:00Z", groups[0].Buckets[0].ResetTime)
	require.Zero(t, *groups[0].Buckets[1].Utilization)
	require.Equal(t, "3p", groups[1].DisplayName)
	require.Len(t, groups[1].Buckets, 2)
	require.InDelta(t, 60.0, *groups[1].Buckets[0].Utilization, 0.001)
	require.Nil(t, groups[1].Buckets[1].Utilization)
	require.Empty(t, groups[1].Buckets[1].ResetTime)
	require.Equal(t, "gemini-argon-wtu", groups[2].DisplayName)
	require.Equal(t, "weekly", groups[2].Buckets[0].Window)
	require.Equal(t, "monthly", groups[3].Buckets[0].Window)
	require.True(t, groups[3].Buckets[0].Disabled)
	require.Nil(t, groups[3].Buckets[0].Utilization)
}

func TestAntigravityModelGroupsDoNotInventWindowsOrMissingQuota(t *testing.T) {
	var models antigravity.FetchAvailableModelsResponse
	require.NoError(t, json.Unmarshal([]byte(`{"models":{"claude-opus-5-5-high":{"displayName":"Claude Opus 5.5","quotaInfo":{"remainingFraction":0.6},"weeklyQuotaInfo":{"remainingFraction":0.2}},"gemini-4-argon-high":{"quotaInfo":{"resetTime":"2026-11-01T00:00:00Z"}},"custom-model":{"weeklyQuotaInfo":{"remainingFraction":0}}}}`), &models))
	info := (&AntigravityQuotaFetcher{}).buildUsageInfo(&models, "", "", nil)
	require.Len(t, info.AntigravityQuotaGroups, 3)
	opus := info.AntigravityQuotaGroups[0]
	require.Equal(t, "Claude Opus 5.5", opus.DisplayName)
	require.Empty(t, opus.Buckets[0].Window, "legacy quota has no confirmed window duration")
	require.Equal(t, "weekly", opus.Buckets[1].Window)
	require.InDelta(t, 80, *opus.Buckets[1].Utilization, 0.001)
	require.Equal(t, 100.0, *info.AntigravityQuotaGroups[1].Buckets[0].Utilization)
	require.Nil(t, info.AntigravityQuotaGroups[2].Buckets[0].Utilization)
	require.NotContains(t, info.AntigravityQuota, "gemini-4-argon-high")
}

func TestFetchAntigravityQuotaSummaryIndependentOfModelEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		summaryOK, modelsOK, forbidden bool
	}{
		{"both endpoints", true, true, false}, {"model endpoint unavailable", true, false, false}, {"summary unavailable", false, true, false},
		{"summary must not hide account verification", true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v1internal:retrieveUserQuotaSummary":
					if !tc.summaryOK {
						w.WriteHeader(404)
						return
					}
					_, _ = fmt.Fprint(w, `{"groups":[{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-5h","window":"5h","remainingFraction":0.4},{"bucketId":"3p-weekly","window":"weekly","remainingFraction":0.1}]}]}`)
				case "/v1internal:fetchAvailableModels":
					if !tc.modelsOK {
						if tc.forbidden {
							w.WriteHeader(http.StatusForbidden)
							_, _ = fmt.Fprint(w, `{"error":{"message":"VALIDATION_REQUIRED"}}`)
							return
						}
						w.WriteHeader(503)
						return
					}
					_, _ = fmt.Fprint(w, `{"models":{"claude-opus-5-5-high":{"quotaInfo":{"remainingFraction":0.4}}}}`)
				case "/v1internal:loadCodeAssist":
					_, _ = fmt.Fprint(w, `{"currentTier":{"id":"g1-pro-tier"}}`)
				default:
					t.Errorf("unexpected endpoint: %s", r.URL.Path)
					w.WriteHeader(404)
				}
			}))
			defer server.Close()
			original := antigravity.BaseURLs
			originalAvailability := antigravity.DefaultURLAvailability
			antigravity.BaseURLs = []string{server.URL}
			antigravity.DefaultURLAvailability = antigravity.NewURLAvailability(time.Hour)
			t.Cleanup(func() { antigravity.BaseURLs = original; antigravity.DefaultURLAvailability = originalAvailability })
			result, err := (&AntigravityQuotaFetcher{}).FetchQuota(context.Background(), &Account{Platform: PlatformAntigravity, Credentials: map[string]any{"access_token": "token", "project_id": "project"}}, "")
			require.NoError(t, err)
			if tc.forbidden {
				require.True(t, result.UsageInfo.IsForbidden)
				require.True(t, result.UsageInfo.NeedsVerify)
				return
			}
			require.Equal(t, "PRO", result.UsageInfo.SubscriptionTier)
			if tc.summaryOK {
				require.Equal(t, "available", result.UsageInfo.AntigravityQuotaSummaryStatus)
				require.Len(t, result.UsageInfo.AntigravityQuotaGroups[0].Buckets, 2)
				require.Equal(t, "weekly", result.UsageInfo.AntigravityQuotaGroups[0].Buckets[1].Window)
			} else {
				require.Equal(t, "unavailable", result.UsageInfo.AntigravityQuotaSummaryStatus)
			}
			if tc.modelsOK {
				require.Contains(t, result.UsageInfo.AntigravityQuota, "claude-opus-5-5-high")
			}
		})
	}
}

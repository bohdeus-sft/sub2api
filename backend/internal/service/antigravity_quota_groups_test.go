//go:build unit

package service

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/stretchr/testify/require"
)

func TestAntigravitySummaryGroupsFourWindows(t *testing.T) {
	var summary antigravity.QuotaSummary
	require.NoError(t, json.Unmarshal([]byte(`{"groups":[{"buckets":[{"bucketId":"3p-weekly","remainingFraction":0.75},{"bucketId":"gemini-weekly","remainingFraction":1},{"bucketId":"gemini-5h","remainingFraction":0,"resetTime":"2026-10-29T23:00:00+03:00"}]}],"buckets":[{"bucketId":"gemini-weekly","remainingFraction":0.1},{"bucketId":"3p-5h","remainingFraction":0.4},{"bucketId":"extra-weekly","remainingFraction":0.5}]}`), &summary))
	groups := antigravitySummaryGroups(&summary)
	require.Len(t, groups, 2)
	require.Equal(t, "gemini", groups[0].ID)
	require.Equal(t, "3p", groups[1].ID)
	for _, group := range groups {
		require.Len(t, group.Buckets, 2)
		require.Equal(t, "5h", group.Buckets[0].Window)
		require.Equal(t, "weekly", group.Buckets[1].Window)
	}
	require.Equal(t, 100.0, *groups[0].Buckets[0].Utilization)
	require.Equal(t, "2026-10-29T20:00:00Z", groups[0].Buckets[0].ResetTime)
	require.Zero(t, *groups[0].Buckets[1].Utilization)
	require.InDelta(t, 60.0, *groups[1].Buckets[0].Utilization, 0.001)
	require.InDelta(t, 25.0, *groups[1].Buckets[1].Utilization, 0.001)
}

func TestAntigravitySummaryGroupsUnknownAndDisabled(t *testing.T) {
	var summary antigravity.QuotaSummary
	require.NoError(t, json.Unmarshal([]byte(`{"buckets":[{"bucketId":"gemini-5h","remainingFraction":null,"resetTime":"invalid"},{"bucketId":"gemini-weekly","remainingFraction":2},{"bucketId":"3p-5h","remainingFraction":0,"disabled":true}]}`), &summary))
	groups := antigravitySummaryGroups(&summary)
	require.Len(t, groups, 2)
	require.Nil(t, groups[0].Buckets[0].Utilization)
	require.Empty(t, groups[0].Buckets[0].ResetTime)
	require.Nil(t, groups[0].Buckets[1].Utilization)
	require.True(t, groups[1].Buckets[0].Disabled)
	require.Nil(t, groups[1].Buckets[1].Utilization)
	require.Nil(t, antigravitySummaryGroups(nil))
	require.Empty(t, antigravitySummaryGroups(&antigravity.QuotaSummary{}))
}

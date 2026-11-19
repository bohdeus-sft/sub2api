package service

import (
	"math"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
)

func antigravitySummaryGroups(summary *antigravity.QuotaSummary) []AntigravityQuotaGroup {
	if summary == nil {
		return nil
	}
	buckets := make(map[string]antigravity.QuotaBucket, 4)
	collect := func(items []antigravity.QuotaBucket) {
		for _, bucket := range items {
			id := strings.TrimSpace(bucket.BucketID)
			switch id {
			case "gemini-5h", "gemini-weekly", "3p-5h", "3p-weekly":
				if _, exists := buckets[id]; !exists {
					buckets[id] = bucket
				}
			}
		}
	}
	for _, group := range summary.Groups {
		collect(group.Buckets)
	}
	collect(summary.Buckets)
	if len(buckets) == 0 {
		return nil
	}

	groups := make([]AntigravityQuotaGroup, 0, 2)
	for _, pool := range []struct{ id, name string }{{"gemini", "Gemini Models"}, {"3p", "Claude and GPT models"}} {
		group := AntigravityQuotaGroup{ID: pool.id, DisplayName: pool.name}
		for _, window := range []string{"5h", "weekly"} {
			id := pool.id + "-" + window
			raw := buckets[id]
			bucket := AntigravityQuotaBucket{ID: id, Window: window, Disabled: raw.Disabled}
			remaining := raw.RemainingFraction
			if remaining != nil && !math.IsNaN(*remaining) && !math.IsInf(*remaining, 0) && *remaining >= 0 && *remaining <= 1 {
				used := (1 - *remaining) * 100
				bucket.Utilization = &used
			}
			if reset, err := time.Parse(time.RFC3339Nano, raw.ResetTime); err == nil {
				bucket.ResetTime = reset.UTC().Format(time.RFC3339Nano)
			}
			group.Buckets = append(group.Buckets, bucket)
		}
		groups = append(groups, group)
	}
	return groups
}

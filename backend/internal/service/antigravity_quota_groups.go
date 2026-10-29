package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
)

func antigravityQuotaBucket(id, name, window, reset string, remaining *float64, disabled bool) AntigravityQuotaBucket {
	bucket := AntigravityQuotaBucket{
		ID: id, DisplayName: name, Window: window, Disabled: disabled,
	}
	if remaining != nil && !math.IsNaN(*remaining) && !math.IsInf(*remaining, 0) && *remaining >= 0 && *remaining <= 1 {
		used := (1 - *remaining) * 100
		bucket.Utilization = &used
	}
	if parsed, err := time.Parse(time.RFC3339Nano, reset); err == nil {
		bucket.ResetTime = parsed.UTC().Format(time.RFC3339Nano)
	}
	return bucket
}

func antigravitySummaryGroups(summary *antigravity.QuotaSummary) []AntigravityQuotaGroup {
	if summary == nil {
		return nil
	}
	groups := append([]antigravity.QuotaGroup(nil), summary.Groups...)
	if len(summary.Buckets) > 0 {
		groups = append(groups, antigravity.QuotaGroup{Buckets: summary.Buckets})
	}
	var result []AntigravityQuotaGroup
	seen := make(map[string]bool)
	for groupIndex, group := range groups {
		// Unnamed/flat responses must be split by pool, so e.g. Argon
		// never appears under the general Gemini or Claude allowance.
		indices := make(map[string]int)
		for _, bucket := range group.Buckets {
			id := strings.TrimSpace(bucket.BucketID)
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			window := strings.TrimSpace(bucket.Window)
			if window == "" {
				if strings.HasSuffix(id, "-weekly") {
					window = "weekly"
				} else if strings.HasSuffix(id, "-5h") {
					window = "5h"
				}
			}
			name := strings.TrimSpace(group.DisplayName)
			if name == "" {
				name = strings.TrimSuffix(strings.TrimSuffix(id, "-weekly"), "-5h")
			}
			index, ok := indices[name]
			if !ok {
				index = len(result)
				indices[name] = index
				result = append(result, AntigravityQuotaGroup{
					ID: fmt.Sprintf("summary-%d-%d", groupIndex, index), DisplayName: name, Source: "summary",
				})
			}
			result[index].Buckets = append(result[index].Buckets, antigravityQuotaBucket(id, bucket.DisplayName, window, bucket.ResetTime, bucket.RemainingFraction, bucket.Disabled))
		}
	}
	return result
}

func antigravityModelGroups(models *antigravity.FetchAvailableModelsResponse) []AntigravityQuotaGroup {
	ids := make([]string, 0, len(models.Models))
	for id := range models.Models {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var groups []AntigravityQuotaGroup
	for _, id := range ids {
		model := models.Models[id]
		group := AntigravityQuotaGroup{ID: id, DisplayName: model.DisplayName, Source: "model"}
		if group.DisplayName == "" {
			group.DisplayName = id
		}
		for _, entry := range []struct {
			window string
			quota  *antigravity.ModelQuotaInfo
		}{{"", model.QuotaInfo}, {"weekly", model.WeeklyQuotaInfo}} {
			if entry.quota == nil {
				continue
			}
			var remaining *float64
			if entry.quota.HasRemainingFraction() {
				remaining = &entry.quota.RemainingFraction
			}
			group.Buckets = append(group.Buckets, antigravityQuotaBucket(id+"/"+entry.window, "", entry.window, entry.quota.ResetTime, remaining, false))
		}
		if len(group.Buckets) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

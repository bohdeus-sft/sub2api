package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/servertiming"
)

// QuotaSummary contains the actual quota pools, including weekly limits that
// fetchAvailableModels does not consistently expose.
type QuotaSummary struct {
	Groups  []QuotaGroup  `json:"groups"`
	Buckets []QuotaBucket `json:"buckets"`
}

type QuotaGroup struct {
	DisplayName string        `json:"displayName"`
	Buckets     []QuotaBucket `json:"buckets"`
}

type QuotaBucket struct {
	BucketID          string   `json:"bucketId"`
	DisplayName       string   `json:"displayName"`
	Window            string   `json:"window"`
	RemainingFraction *float64 `json:"remainingFraction"`
	ResetTime         string   `json:"resetTime"`
	Disabled          bool     `json:"disabled"`
}

// RetrieveUserQuotaSummary uses the accounting endpoint (daily first) and the
// ACP client identity. Older hub identities can receive only the third-party
// pool, silently omitting Gemini and Argon. Never follow credentialed redirects.
func (c *Client) RetrieveUserQuotaSummary(ctx context.Context, accessToken, projectID string, bodyLimit int64) (*QuotaSummary, error) {
	if c == nil || c.httpClient == nil || bodyLimit <= 0 {
		return nil, errors.New("quota summary client and positive body limit are required")
	}
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("quota summary requires a project ID")
	}
	body, err := json.Marshal(FetchAvailableModelsRequest{Project: projectID})
	if err != nil {
		return nil, err
	}
	httpClient := *c.httpClient
	httpClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	var lastErr error
	for _, baseURL := range ForwardBaseURLs() {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1internal:retrieveUserQuotaSummary", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "antigravity/acp/1.3.0")
		resp, err := servertiming.Do(&httpClient, req)
		if err != nil {
			lastErr = err
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			continue
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, bodyLimit+1))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if int64(len(data)) > bodyLimit {
			return nil, fmt.Errorf("quota summary exceeds %d bytes", bodyLimit)
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("quota summary HTTP %d", resp.StatusCode)
			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden || (resp.StatusCode >= 300 && resp.StatusCode < 400) {
				return nil, lastErr
			}
			continue
		}
		var summary QuotaSummary
		if err := json.Unmarshal(data, &summary); err != nil {
			return nil, fmt.Errorf("decode quota summary: %w", err)
		}
		if len(summary.Groups) == 0 && len(summary.Buckets) == 0 {
			lastErr = errors.New("quota summary contains no quota groups")
			continue
		}
		return &summary, nil
	}
	if lastErr == nil {
		lastErr = errors.New("quota summary endpoint unavailable")
	}
	return nil, lastErr
}

// Missing fractions must remain unknown rather than becoming an exhausted quota.
func (q *ModelQuotaInfo) UnmarshalJSON(data []byte) error {
	var wire struct {
		RemainingFraction *float64 `json:"remainingFraction"`
		ResetTime         string   `json:"resetTime"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	q.ResetTime = wire.ResetTime
	q.remainingFractionMissing = wire.RemainingFraction == nil
	q.RemainingFraction = 0
	if wire.RemainingFraction != nil {
		q.RemainingFraction = *wire.RemainingFraction
	}
	return nil
}

func (q *ModelQuotaInfo) HasRemainingFraction() bool {
	return q != nil && !q.remainingFractionMissing && !math.IsNaN(q.RemainingFraction) && !math.IsInf(q.RemainingFraction, 0) && q.RemainingFraction >= 0 && q.RemainingFraction <= 1
}

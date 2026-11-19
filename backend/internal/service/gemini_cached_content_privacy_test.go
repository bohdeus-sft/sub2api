//go:build unit

package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersonalPrivacyGeminiCacheKeepsBindingWithoutLabel(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		repo := newGeminiCachedContentRepoStub()
		svc := NewGeminiCachedContentService(repo, nil, nil)
		record := &GeminiCachedContent{PublicID: "cache-id", UpstreamName: "cachedContents/upstream-id", DisplayName: "private document title"}
		require.NoError(t, svc.Save(context.Background(), record))
		require.Empty(t, repo.records[record.PublicID].DisplayName)
		require.Equal(t, "cachedContents/upstream-id", repo.records[record.PublicID].UpstreamName)
		require.NoError(t, svc.Forget(context.Background(), record))
		require.Empty(t, repo.records)
	})
}

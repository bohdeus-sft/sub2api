package securityaudit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/stretchr/testify/require"
)

func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		ctx := context.Background()
		_, err := NewPostgreSQLRepository(nil).CreateStagingWithCapacity(ctx, PromptSnapshot{}, 1, 1, 1)
		require.Error(t, err)
		_, err = NewPostgreSQLRepository(nil).Complete(ctx, nil, nil, true)
		require.Error(t, err)
		// Nil dependencies would panic if workers, storage or configuration ran.
		svc := NewPromptService(nil, nil, nil, nil, nil)
		require.NoError(t, svc.Start(ctx))
		require.Equal(t, ModeOff, svc.EffectiveMode())
		require.NoError(t, svc.Enqueue(ctx, Request{}))
		decision, err := svc.Evaluate(ctx, Request{})
		require.NoError(t, err)
		require.True(t, decision.AllowNextStage)
		_, err = svc.SaveConfig(ctx, UpdateConfigRequest{}, 1)
		require.Error(t, err)
		require.Equal(t, "disabled", svc.Probe(ctx, ProbeRequest{}).Status)
		require.Error(t, NewRedisPayloadStore(nil).Set(ctx, 1, "private prompt", time.Minute))
		var calls atomic.Int64
		endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
		defer endpoint.Close()
		_, err = NewOpenAICompatibleScanner().Scan(ctx, ActiveEndpoint{BaseURL: endpoint.URL}, "private prompt", nil)
		require.Error(t, err)
		require.Zero(t, calls.Load())
		require.True(t, NewCoordinator(nil, nil).Check(ctx, Request{}).AllowNextStage)
	})
}

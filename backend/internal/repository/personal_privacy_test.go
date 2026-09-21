package repository

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		ctx := context.Background()
		require.NoError(t, NewContentModerationRepository(nil).CreateLog(ctx, &service.ContentModerationLog{}))
		_, uploadErr := (&S3ImageStorage{}).Save(ctx, "private", "image/png", []byte("private content"))
		require.Error(t, uploadErr)
		mr := miniredis.RunT(t)
		rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
		defer rdb.Close()
		cache := NewGatewayCache(rdb)
		require.NoError(t, cache.SetReasoningContent(ctx, "item_private", "private reasoning", time.Minute))
		require.Empty(t, mr.Keys())
		// Existing cache data is never replayed by the personal deployment.
		mr.Set(reasoningContentPrefix+"old", "old private reasoning")
		_, err := cache.GetReasoningContent(ctx, "old")
		require.ErrorIs(t, err, service.ErrReasoningContentNotFound)
		require.Error(t, NewImageTaskStore(rdb).Save(ctx, &service.ImageTaskRecord{ID: "private"}, time.Minute))
		require.False(t, mr.Exists(imageTaskKey("private")))
		// These repositories must not access a database, even with content input.
		ops := NewOpsRepository(nil)
		_, err = ops.InsertErrorLog(ctx, &service.OpsInsertErrorLogInput{ErrorBody: "private response"})
		require.NoError(t, err)
		_, err = ops.BatchInsertSystemLogs(ctx, []*service.OpsInsertSystemLogInput{{}})
		require.NoError(t, err)
		_, err = (&auditLogRepository{}).BatchInsert(ctx, []*service.AuditLog{{RequestBody: "private request"}})
		require.NoError(t, err)
		// Errors can echo the original prompt, including in scheduling caches.
		state := &service.TempUnschedState{UntilUnix: time.Now().Add(time.Hour).Unix(), ErrorMessage: "canary-private-prompt", StatusCode: 429}
		require.NoError(t, NewTempUnschedCache(rdb).SetTempUnsched(ctx, 9, state))
		raw, err := mr.Get(tempUnschedPrefix + "9")
		require.NoError(t, err)
		require.NotContains(t, raw, "canary-private-prompt")
		require.Equal(t, "canary-private-prompt", state.ErrorMessage, "do not mutate caller state")
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()
		now := time.Now()
		mock.ExpectQuery("INSERT INTO scheduled_test_results").
			WithArgs(int64(1), "error", "", "[details omitted by privacy policy]", int64(12), now, now).
			WillReturnRows(sqlmock.NewRows([]string{"id", "plan_id", "status", "response_text", "error_message", "latency_ms", "started_at", "finished_at", "created_at"}).AddRow(2, 1, "error", "", "[details omitted by privacy policy]", 12, now, now, now))
		_, err = NewScheduledTestResultRepository(db).Create(ctx, &service.ScheduledTestResult{PlanID: 1, Status: "error", ResponseText: "canary-private-response", ErrorMessage: "canary-private-prompt", LatencyMs: 12, StartedAt: now, FinishedAt: now})
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
		full, meta, err := marshalSchedulerCacheAccount(service.Account{ID: 1, ErrorMessage: "canary-private-prompt", TempUnschedulableReason: "canary-private-response"})
		require.NoError(t, err)
		for _, payload := range [][]byte{full, meta} {
			require.NotContains(t, string(payload), "canary-private")
		}
		// Scheduling metadata still works without keeping conversation text.
		require.NoError(t, cache.SetSessionAccountID(ctx, 1, "hash", 7, time.Minute))
		id, err := cache.GetSessionAccountID(ctx, 1, "hash")
		require.NoError(t, err)
		require.EqualValues(t, 7, id)
	})
}

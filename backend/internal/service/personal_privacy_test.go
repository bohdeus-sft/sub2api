package service

import (
	"bytes"
	"context"
	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		ctx := context.Background()
		stage := newDefaultOpenAIFirstOutputStage()
		defer stage.Close()
		stage.createTemp = func() (*os.File, error) { t.Fatal("response must never spill to a file"); return nil, nil }
		payload := strings.Repeat("private response", 16384)
		n, err := stage.WriteString(payload)
		require.NoError(t, err)
		require.Equal(t, len(payload), n)
		var delivered bytes.Buffer
		require.NoError(t, stage.CommitTo(&delivered))
		require.Equal(t, payload, delivered.String())
		require.Nil(t, stage.tempFile)
		bounded := newOpenAIFirstOutputStage(3)
		_, err = bounded.WriteString("four")
		require.ErrorIs(t, err, errOpenAIFirstOutputStageLimit)
		mod := &ContentModerationService{}
		decision, err := mod.Check(ctx, ContentModerationCheckInput{})
		require.NoError(t, err)
		require.True(t, decision.Allowed)
		_, err = mod.TestAPIKeys(ctx, TestContentModerationAPIKeysInput{})
		require.Error(t, err)
		_, err = mod.UpdateConfig(ctx, UpdateContentModerationConfigInput{})
		require.Error(t, err)
		mod.RecordCyberPolicyEvent(ctx, CyberPolicyRecordInput{})
		require.NoError(t, (&PluginManager{}).Start(ctx))
		_, err = (&PluginManager{}).Enable(ctx, 1, true, 100)
		require.Error(t, err)
		uploader, enabled := (&ImageStorageSettingService{}).resolve()
		require.Nil(t, uploader)
		require.False(t, enabled)
		update := &UpdateService{}
		require.Error(t, update.PerformUpdate(ctx))
		require.Error(t, update.Rollback())
		require.Error(t, update.RollbackToVersion(ctx, "v0.2.7"))
	})
}

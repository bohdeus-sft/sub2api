package config

import (
	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		resetViperWithJWTSecret(t)
		viper.Set("gateway.log_upstream_error_body", true)
		viper.Set("gateway.openai_ws.payload_log_sample_rate", 1.0)
		viper.Set("log.output.to_file", true)
		viper.Set("log.output.to_stdout", true)
		cfg, err := Load()
		require.NoError(t, err)
		require.False(t, cfg.Gateway.LogUpstreamErrorBody)
		require.Zero(t, cfg.Gateway.OpenAIWS.PayloadLogSampleRate)
		require.False(t, cfg.Log.Output.ToFile)
		require.False(t, cfg.Log.Output.ToStdout)
		require.False(t, cfg.BatchImage.Enabled)
		require.False(t, cfg.BatchImage.QueueEnabled)
		require.False(t, cfg.ImageStorage.Enabled)
	})
}

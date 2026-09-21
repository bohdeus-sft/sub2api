package logger

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type privateTestSink struct{ calls int }

func (s *privateTestSink) WriteLogEvent(_ *LogEvent) { s.calls++ }

func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		// Capture the actual output, including stdlib bridges and reconfigure.
		r, w, err := os.Pipe()
		require.NoError(t, err)
		original := os.Stdout
		os.Stdout = w
		defer func() { os.Stdout = original; r.Close(); w.Close() }()
		filename := filepath.Join(t.TempDir(), "private.log")
		require.NoError(t, Init(InitOptions{Level: "debug", Output: OutputOptions{ToFile: true, ToStdout: true, FilePath: filename}}))
		sink := &privateTestSink{}
		SetSink(sink)
		secret := "SENSITIVE_PROMPT_TOKEN_123"
		L().Named(secret).With(zap.String("authorization", secret), zap.Int("status_code", 502)).Error(
			"upstream failed "+secret,
			zap.Error(errors.New("connection refused "+secret)),
			zap.String("request", secret), zap.Any("response", map[string]string{"text": secret}),
			zap.String("status", secret), zap.String("method", secret),
			zap.Int64("latency_ms", 123))
		log.Print("Auto setup failed: timeout " + secret)
		slog.With("token", secret).Error("private response " + secret)
		S().Errorf("failed response: %s", secret)
		L().WithOptions(zap.AddStacktrace(zap.DebugLevel)).Error("http request contains gin errors", zap.String("errors", secret))
		WriteSinkEvent("error", "gateway", secret, nil)
		require.NoError(t, SetLevel("debug"))
		require.NoError(t, Reconfigure(func(o *InitOptions) error { o.Output.ToFile = true; o.Output.ToStdout = false; return nil }))
		L().Debug(secret)
		L().Info("http request completed", zap.String("method", "POST"), zap.Int("status_code", 200))
		Sync()
		require.NoError(t, w.Close())
		output, err := io.ReadAll(r)
		require.NoError(t, err)
		require.NotContains(t, string(output), secret)
		require.NotContains(t, string(output), "stacktrace")
		require.Contains(t, string(output), "connection_refused")
		require.Contains(t, string(output), "timeout")
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		require.Len(t, lines, 7)
		var first map[string]any
		require.NoError(t, json.Unmarshal([]byte(lines[0]), &first))
		require.Equal(t, "ERROR", first["level"])
		require.Equal(t, float64(502), first["status_code"])
		require.Equal(t, float64(123), first["latency_ms"])
		require.Contains(t, first["caller"], "personal_privacy_test.go")
		var slogEvent map[string]any
		require.NoError(t, json.Unmarshal([]byte(lines[2]), &slogEvent))
		require.Contains(t, slogEvent["caller"], "personal_privacy_test.go")
		require.Contains(t, lines[6], "http request completed")
		require.Contains(t, lines[6], "POST")
		_, err = os.Stat(filename)
		require.True(t, os.IsNotExist(err))
		require.Zero(t, sink.calls)
	})
}

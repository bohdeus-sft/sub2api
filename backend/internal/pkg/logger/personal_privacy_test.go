package logger

import (
	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

type privateTestSink struct{ calls int }

func (s *privateTestSink) WriteLogEvent(_ *LogEvent) { s.calls++ }
func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		filename := filepath.Join(t.TempDir(), "private.log")
		require.NoError(t, Init(InitOptions{Level: "debug", Output: OutputOptions{ToFile: true, ToStdout: true, FilePath: filename}}))
		sink := &privateTestSink{}
		SetSink(sink)
		L().Error("private response", zap.String("request", "private prompt"))
		log.Print("private prompt")
		slog.Error("private response")
		WriteSinkEvent("error", "gateway", "private response", nil)
		require.NoError(t, SetLevel("debug"))
		require.NoError(t, Reconfigure(func(o *InitOptions) error { o.Output.ToFile = true; return nil }))
		L().Debug("private prompt")
		Sync()
		_, err := os.Stat(filename)
		require.True(t, os.IsNotExist(err))
		require.Zero(t, sink.calls)
	})
}

package main

import (
	"github.com/Wei-Shaw/sub2api/internal/privacy"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestPersonalPrivacy(t *testing.T) {
	require.True(t, privacy.Enabled(), "server startup must always enforce privacy")
	require.Equal(t, io.Discard, gin.DefaultWriter)
	require.Equal(t, io.Discard, gin.DefaultErrorWriter)
}

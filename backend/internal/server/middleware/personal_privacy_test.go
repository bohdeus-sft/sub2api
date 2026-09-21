package middleware

import (
	"bytes"
	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPersonalPrivacy(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		var logs bytes.Buffer
		original := gin.DefaultErrorWriter
		gin.DefaultErrorWriter = &logs
		defer func() { gin.DefaultErrorWriter = original }()
		r := gin.New()
		r.Use(PersonalPrivacy(), Recovery())
		calls := 0
		r.NoRoute(func(c *gin.Context) { calls++; c.Status(200) })
		for _, p := range []string{
			"/api/v1/admin/prompt-audit/endpoints/probe", "/api/v1/admin/risk-control/config",
			"/api/v1/admin/plugins/1/enable", "/api/v1/plugin-ui/1/index.html",
			"/api/v1/admin/backups/image-storage/test", "/api/v1/admin/system/update",
			"/api/v1/admin/system/rollback", "/v1/images/generations/async",
			"/images/edits/async", "/api/v1/images/batches/1", "/v1/images/tasks/1", "/v1/web_search",
		} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("POST", p, nil))
			require.Equal(t, 403, w.Code, p)
		}
		require.Zero(t, calls)
		for _, p := range []string{"/v1/responses", "/v1/chat/completions", "/v1/messages", "/health"} {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest("POST", p, nil))
			require.Equal(t, 200, w.Code, p)
		}
		r.POST("/panic", func(c *gin.Context) { panic("private response") })
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/panic?private=prompt", nil)
		req.Header.Set("Authorization", "Bearer private-secret")
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Empty(t, logs.String())
		require.NotContains(t, w.Body.String(), "private")
	})
}

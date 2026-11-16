package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/privacy/testutil"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPersonalPrivacyMCPManagementRouter(t *testing.T) {
	testutil.Run(t, func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := newMCPManagementRouter(mcpRouteHandlers(), middleware.AuditLogMiddleware(mcpRouteNext),
			middleware.StepUpAuthMiddleware(mcpRouteNext), nil, nil)
		for _, path := range []string{"/api/v1/admin/system/update", "/api/v1/admin/plugins/1/enable", "/api/v1/admin/prompt-audit/endpoints/probe"} {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
			require.Equal(t, http.StatusForbidden, response.Code, path)
			require.Contains(t, response.Body.String(), "privacy_policy")
		}
		router.GET("/privacy-panic", func(*gin.Context) { panic("private MCP payload") })
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/privacy-panic", nil))
		require.Equal(t, http.StatusInternalServerError, response.Code)
		require.NotContains(t, response.Body.String(), "private MCP payload")
	})
}

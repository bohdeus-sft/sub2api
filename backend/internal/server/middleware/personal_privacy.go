package middleware

import (
	"net/http"
	"path"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/privacy"
	"github.com/gin-gonic/gin"
)

// PersonalPrivacy rejects features whose contract requires persisted content or
// additional recipients. Runs before request/audit loggers and endpoint handlers.
func PersonalPrivacy() gin.HandlerFunc {
	return func(c *gin.Context) {
		if privacy.Enabled() && privateContentRoute(c.Request.URL.Path) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{
				"type": "privacy_policy", "message": "This feature is unavailable in this personal, no-content-retention deployment.",
			}})
			return
		}
		c.Next()
	}
}

func privateContentRoute(raw string) bool {
	p := path.Clean(raw)
	for _, prefix := range []string{
		"/api/v1/admin/prompt-audit", "/api/v1/admin/risk-control",
		"/api/v1/admin/plugins", "/api/v1/plugin-ui",
		"/api/v1/admin/backups/image-storage",
		"/api/v1/admin/system/update", "/api/v1/admin/system/rollback",
	} {
		if p == prefix || strings.HasPrefix(p, prefix+"/") {
			return true
		}
	}
	for _, prefix := range []string{"", "/v1", "/api/v1"} {
		for _, suffix := range []string{"/images/batches", "/images/tasks"} {
			base := prefix + suffix
			if p == base || strings.HasPrefix(p, base+"/") {
				return true
			}
		}
		if p == prefix+"/images/generations/async" || p == prefix+"/images/edits/async" || p == prefix+"/web_search" {
			return true
		}
	}
	return false
}

package logger

import (
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// privacyCore is a fail-closed boundary: never serialize arbitrary messages,
// strings, objects, errors, logger names or stack traces. A denylist of secret
// field names alone cannot protect prompts embedded in provider error text.
type privacyCore struct{ core zapcore.Core }

func (c *privacyCore) Enabled(level zapcore.Level) bool { return c.core.Enabled(level) }
func (c *privacyCore) Sync() error                      { return c.core.Sync() }
func (c *privacyCore) With(fields []zapcore.Field) zapcore.Core {
	return &privacyCore{core: c.core.With(privateLogFields(fields))}
}
func (c *privacyCore) Check(e zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// Do not register the underlying core: that would bypass filtering in Write.
	if c.Enabled(e.Level) {
		return ce.AddCore(e, c)
	}
	return ce
}
func (c *privacyCore) Write(e zapcore.Entry, fields []zapcore.Field) error {
	category := privateErrorCategory(e.Message)
	e.Message = privateLogMessage(e.Message)
	e.LoggerName = ""
	e.Stack = ""
	safe := privateLogFields(fields)
	if category != "" {
		safe = append(safe, zap.String("message_category", category))
	}
	return c.core.Write(e, safe)
}

func privateLogMessage(message string) string {
	// Exact static messages are safe. Never retain a suffix, even for a known
	// event: printf-style suffixes commonly contain URLs, credentials and bodies.
	switch message {
	case "http request completed", "http request contains gin errors",
		"http handler panic", "audit log flush failed", "audit log retention cleanup failed",
		"Auto setup mode enabled...", "First run detected, starting setup wizard...",
		"Complete the setup wizard to configure Sub2API", "Shutting down server...", "Server exited",
		"Auto setup enabled, configuring from environment variables...",
		"Testing database connection...", "Database connection successful",
		"Testing Redis connection...", "Redis connection successful",
		"Initializing database...", "Database initialized successfully",
		"Creating admin user...", "Admin user already exists, skipping admin bootstrap",
		"Database already has user data; skipping auto admin bootstrap to avoid password overwrite",
		"Admin bootstrap skipped", "Writing configuration file...", "Configuration file created",
		"Installation lock created", "Auto setup completed successfully!":
		return message
	}
	for _, prefix := range []string{
		"Auto setup failed:", "Setup failed:", "Failed to load config:",
		"Failed to initialize logger:", "Failed to initialize application:",
		"Failed to start setup server:", "Failed to start server:", "Server forced to shutdown:",
		"Admin user created:", "Server started on",
		"Server listening on", "Starting server on", "Setup wizard available at",
	} {
		if strings.HasPrefix(message, prefix) {
			return strings.TrimSuffix(prefix, ":")
		}
	}
	return "application event (free text omitted)"
}

func privateErrorCategory(message string) string {
	lower := strings.ToLower(message)
	for _, item := range []struct{ match, category string }{
		{"connection refused", "connection_refused"},
		{"no such host", "dns_failure"},
		{"deadline exceeded", "timeout"}, {"timeout", "timeout"},
		{"context canceled", "canceled"},
		{"certificate", "tls_certificate_error"},
		{"password authentication failed", "database_authentication_failed"},
		{"permission denied", "permission_denied"},
		{"no space left", "disk_full"},
		{"too many connections", "connection_limit"},
		{"rate limit", "rate_limited"},
		{"unauthorized", "unauthorized"}, {"forbidden", "forbidden"},
		{"connection reset", "connection_reset"}, {"broken pipe", "broken_pipe"},
		{"eof", "unexpected_eof"},
	} {
		if strings.Contains(lower, item.match) {
			return item.category
		}
	}
	return ""
}

func privateLogFields(fields []zapcore.Field) []zapcore.Field {
	safe := make([]zapcore.Field, 0, len(fields))
	for _, f := range fields {
		switch f.Key {
		case "status", "status_code", "upstream_status", "http_status", "latency_ms",
			"duration_ms", "elapsed_ms", "attempt", "retry_count", "batch", "count", "dropped_count":
			switch f.Type {
			case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type,
				zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type,
				zapcore.Float64Type, zapcore.Float32Type, zapcore.DurationType:
				safe = append(safe, f)
			}
		case "method":
			if f.Type == zapcore.StringType {
				switch f.String {
				case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
					safe = append(safe, f)
				}
			}
		case "error", "err":
			message := f.String
			if f.Type == zapcore.ErrorType {
				if err, ok := f.Interface.(error); ok && err != nil {
					message = err.Error()
				}
			}
			category := privateErrorCategory(message)
			if category == "" {
				category = "details_omitted"
			}
			safe = append(safe, zap.String("error_category", category))
		}
	}
	return safe
}

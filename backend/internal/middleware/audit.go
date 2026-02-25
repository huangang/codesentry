package middleware

import (
	"bytes"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/huangang/codesentry/backend/internal/services"
)

// AuditLog records admin write operations (POST/PUT/DELETE) to system_logs.
func AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		// Only audit write operations
		if method != "POST" && method != "PUT" && method != "DELETE" {
			c.Next()
			return
		}

		// Capture request body (up to 2000 chars for Extra)
		var bodySnippet string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			bodySnippet = string(bodyBytes)
			if len(bodySnippet) > 2000 {
				bodySnippet = bodySnippet[:2000] + "...[truncated]"
			}
			// Mask sensitive fields
			bodySnippet = maskSensitiveFields(bodySnippet)
		}

		// Process the request
		c.Next()

		// After handler — record audit log
		userID := GetUserID(c)
		username := GetUsername(c)
		ip := c.ClientIP()
		userAgent := c.Request.UserAgent()
		status := c.Writer.Status()

		module, action := parseRouteInfo(c.FullPath(), method)

		message := formatAuditMessage(username, method, c.Request.URL.Path, status)

		var uid *uint
		if userID > 0 {
			uid = &userID
		}

		services.LogInfo(module, action, message, uid, ip, userAgent, map[string]interface{}{
			"method": method,
			"path":   c.Request.URL.Path,
			"status": status,
			"body":   bodySnippet,
			"audit":  true,
		})
	}
}

// parseRouteInfo extracts module and action from a Gin route pattern.
// e.g. "/api/projects/:id" + "PUT" → module="Projects", action="Update"
func parseRouteInfo(fullPath, method string) (module, action string) {
	// Strip /api/ prefix
	path := strings.TrimPrefix(fullPath, "/api/")

	// Extract first segment as module
	parts := strings.SplitN(path, "/", 2)
	module = parts[0]
	if module == "" {
		module = "unknown"
	}

	// Capitalize and clean up module name (e.g. "llm-configs" → "LLM-Configs")
	module = strings.Title(strings.ReplaceAll(module, "-", " "))

	// Determine action from HTTP method
	switch method {
	case "POST":
		action = "Create"
	case "PUT":
		action = "Update"
	case "DELETE":
		action = "Delete"
	default:
		action = method
	}

	return module, action
}

// formatAuditMessage creates a human-readable audit message.
func formatAuditMessage(username, method, path string, status int) string {
	var b strings.Builder
	b.WriteString("[Audit] ")
	b.WriteString(username)
	b.WriteString(" ")
	b.WriteString(method)
	b.WriteString(" ")
	b.WriteString(path)
	b.WriteString(" → ")
	if status >= 200 && status < 300 {
		b.WriteString("OK")
	} else {
		b.WriteString("Failed")
	}
	return b.String()
}

// maskSensitiveFields replaces sensitive values in JSON body
func maskSensitiveFields(body string) string {
	sensitiveKeys := []string{"password", "api_key", "apiKey", "secret", "token", "access_token"}
	lower := strings.ToLower(body)
	for _, key := range sensitiveKeys {
		if strings.Contains(lower, key) {
			// Simple mask: replace the value after the key
			body = maskJSONValue(body, key)
		}
	}
	return body
}

// maskJSONValue does a best-effort mask of JSON string values for a given key
func maskJSONValue(body, key string) string {
	// Look for patterns like "key":"value" or "key": "value"
	lower := strings.ToLower(body)
	idx := strings.Index(lower, "\""+key+"\"")
	if idx == -1 {
		return body
	}

	// Find the colon after the key
	colonIdx := strings.Index(body[idx+len(key)+2:], ":")
	if colonIdx == -1 {
		return body
	}
	valueStart := idx + len(key) + 2 + colonIdx + 1

	// Skip whitespace
	for valueStart < len(body) && (body[valueStart] == ' ' || body[valueStart] == '\t') {
		valueStart++
	}

	if valueStart >= len(body) {
		return body
	}

	// If it's a quoted string, mask it
	if body[valueStart] == '"' {
		endQuote := strings.Index(body[valueStart+1:], "\"")
		if endQuote == -1 {
			return body
		}
		return body[:valueStart+1] + "***" + body[valueStart+1+endQuote:]
	}

	return body
}

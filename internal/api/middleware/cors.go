// Package middleware holds the api binary's HTTP middleware: CORS,
// rate limiting, request logging, etc.
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/0xfandom/chainpulse/internal/types"
)

// CORS returns a Gin middleware that applies the CORS policy from cfg.
// Allowed methods are fixed (GET, POST, OPTIONS) and headers cover the
// common JSON+auth set.
func CORS(cfg types.APICORSConfig) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	wildcard := false
	for _, o := range cfg.AllowedOrigins {
		if o == "*" {
			wildcard = true
		}
		allowed[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// Match allowlist. Wildcard accepts all but disables credentials.
		_, exact := allowed[origin]
		switch {
		case wildcard && cfg.AllowCredentials:
			// Spec: cannot send credentials with "*" origin. Echo the
			// caller's origin only if also explicitly listed.
			if exact {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		case wildcard:
			c.Header("Access-Control-Allow-Origin", "*")
		case exact:
			c.Header("Access-Control-Allow-Origin", origin)
			if cfg.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		// Headers common to all responses with an Origin.
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "600")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

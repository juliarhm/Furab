// Package middleware provides shared HTTP middleware for all Furab microservices.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"furab-backend/shared/utils"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user ID.
	UserIDKey contextKey = "userID"
	// UserRoleKey is the context key for the authenticated user role.
	UserRoleKey contextKey = "userRole"
)

// Claims represents the JWT token claims.
type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens and injects user info into the request context.
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.ErrorResponse(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			// Expect "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				utils.ErrorResponse(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Parse and validate token
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
				// Validate signing method
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				utils.ErrorResponse(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// Inject user info into context
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts the user ID from the request context.
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetUserRole extracts the user role from the request context.
func GetUserRole(ctx context.Context) string {
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role
	}
	return ""
}

// CORSMiddleware adds CORS headers to responses.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequestIDMiddleware adds a unique request ID to each request context.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		w.Header().Set("X-Request-ID", requestID)

		ctx := context.WithValue(r.Context(), contextKey("requestID"), requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID generates a simple unique request ID.
func generateRequestID() string {
	// Simple implementation - in production, use UUID
	return "req-" + strings.ReplaceAll(
		strings.ReplaceAll(
			strings.Split(
				strings.TrimPrefix(
					http.TimeFormat, "Mon, "),
				" ")[0],
			"-", ""),
		":", "")
}

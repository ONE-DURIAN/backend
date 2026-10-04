package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"community-backend/internal/domain"
	"community-backend/pkg/redisclient"
	"community-backend/pkg/response"
)

type contextKey string

const (
	UserIDContextKey contextKey = "user_id"
	RoleContextKey   contextKey = "user_role"
)

func AuthMiddleware(authSvc Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, http.StatusUnauthorized, "Invalid authorization format. Expected 'Bearer <token>'")
				return
			}

			tokenStr := parts[1]
			claims, err := authSvc.ValidateAccessToken(tokenStr)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, RoleContextKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(allowedRoles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, ok := r.Context().Value(RoleContextKey).(domain.Role)
			if !ok {
				response.Error(w, http.StatusForbidden, "Role permission denied")
				return
			}

			allowed := false
			for _, r := range allowedRoles {
				if r == userRole {
					allowed = true
					break
				}
			}

			if !allowed {
				response.Error(w, http.StatusForbidden, "You do not have permission to access this resource")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func GetUserID(ctx context.Context) string {
	if val, ok := ctx.Value(UserIDContextKey).(string); ok {
		return val
	}
	return ""
}

func GetUserRole(ctx context.Context) domain.Role {
	if val, ok := ctx.Value(RoleContextKey).(domain.Role); ok {
		return val
	}
	return ""
}

// RateLimitMiddleware limits the number of requests per client IP within a rolling time window using Redis
func RateLimitMiddleware(rdb *redisclient.Client, limit int, window time.Duration, keyPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Cloudflare Tunnel passes the actual client IP in CF-Connecting-IP
			clientIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
			if clientIP == "" {
				clientIP = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
			}
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			key := fmt.Sprintf("ratelimit:%s:%s", keyPrefix, clientIP)
			ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
			defer cancel()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				// Fail-open: If Redis is momentarily down, don't block legitimate users
				next.ServeHTTP(w, r)
				return
			}

			// Set expiration on first hit
			if count == 1 {
				_ = rdb.Expire(ctx, key, window).Err()
			}

			if count > int64(limit) {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", window.Seconds()))
				response.Error(w, http.StatusTooManyRequests, "Too many requests. Please slow down and try again shortly.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

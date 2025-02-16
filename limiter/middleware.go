package limiter

import (
	"context"
	"net/http"
	"strings"
)

func Middleware(rateLimiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()
			ip := strings.Split(r.RemoteAddr, ":")[0]
			token := r.Header.Get("API_KEY")

			var key string
			var limit int

			if token != "" {
				key = "token:" + token
				limit = rateLimiter.rateLimitToken
			} else {
				key = "ip:" + ip
				limit = rateLimiter.rateLimitIP
			}

			// Verifica bloqueio
			if blocked, _ := rateLimiter.storage.IsBlocked(ctx, key); blocked {
				http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
				return
			}

			// Verifica limite
			allowed, _ := rateLimiter.Allow(ctx, key, limit)
			if !allowed {
				http.Error(w, "you have reached the maximum number of requests or actions allowed within a certain time frame", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

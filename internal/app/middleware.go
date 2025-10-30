package app

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/pkg/user"
)

// SetupMiddleware wires all HTTP middlewares for the application.
func SetupMiddleware(r *mux.Router, deps *Dependencies, cfg config.Application) {

	// Propagate X-User-Id header into context for downstream services
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			userIdHeader := req.Header.Get("X-User-Id")
			ctx := req.Context()
			if userIdHeader != "" {
				ctx = user.WithId(ctx, userIdHeader)
			}
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
}

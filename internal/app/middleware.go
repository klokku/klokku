package app

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/pkg/user"
	log "github.com/sirupsen/logrus"
)

// SetupMiddleware wires all HTTP middlewares for the application.
func SetupMiddleware(r *mux.Router, deps *Dependencies, cfg config.Application) {

	// Propagate X-User-Id header into context for downstream services
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			log.Debug("Propagating user ID header")

			userIdHeader := req.Header.Get("X-User-Id")
			ctx := req.Context()

			if userIdHeader != "" {
				u, err := deps.UserService.GetUserByUid(ctx, userIdHeader)
				if err != nil {
					if errors.Is(err, user.ErrUserNotFound) {
						log.Debugf("user not found: %s", userIdHeader)
						http.Error(w, "user not found", http.StatusForbidden)
						return
					} else {
						log.Errorf("failed to get user: %v", err)
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
				} else {
					log.Debugf("user found: %s", u.Uid)
					ctx = user.WithUser(ctx, u)
				}
			}
			log.Debug("Propagated user ID header")
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	})
}

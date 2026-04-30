package middleware

import (
	"net/http"

	"github.com/ahoylog/kvik-tasks/internal/core"
)

// BasicAuth returns middleware that enforces HTTP Basic Auth.
// If no users with passwords exist, auth is skipped (open access).
func BasicAuth(userService *core.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth if no users are configured
			if !userService.HasUsers(r.Context()) {
				next.ServeHTTP(w, r)
				return
			}

			username, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="kvik-tasks"`)
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			user, valid := userService.Authenticate(r.Context(), username, password)
			if !valid {
				w.Header().Set("WWW-Authenticate", `Basic realm="kvik-tasks"`)
				http.Error(w, "Invalid credentials", http.StatusUnauthorized)
				return
			}

			ctx := core.ContextWithUser(r.Context(), user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

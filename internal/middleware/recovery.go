package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"
)

// Recoverer is a middleware that recovers from panics.
// It logs the panic and returns a 500 Internal Server Error.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rvr := recover(); rvr != nil {
					// Get request ID for correlation
					requestID := GetRequestID(r.Context())

					// Log the panic
					logger.Error("panic recovered",
						slog.String("request_id", requestID),
						slog.Any("panic", rvr),
						slog.String("stack", string(debug.Stack())),
					)

					// In development, also print to stderr for visibility
					if os.Getenv("APP_ENV") == "development" {
						debug.PrintStack()
					}

					// Return 500 error
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

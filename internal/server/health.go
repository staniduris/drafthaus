package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
)

var startTime = time.Now()

// HealthHandler returns a health check endpoint for the given store.
func HealthHandler(store draft.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		uptime := time.Since(startTime).Round(time.Second).String()

		if sqliteStore, ok := store.(*draft.SQLiteStore); ok {
			if err := sqliteStore.DB().PingContext(r.Context()); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
					"status":  "error",
					"message": err.Error(),
				})
				return
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{ //nolint:errcheck
			"status":  "ok",
			"version": "0.8.0",
			"uptime":  uptime,
		})
	}
}

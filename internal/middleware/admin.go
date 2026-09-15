package middleware

import (
	"net/http"
	"strings"
)

var adminIDs map[string]struct{}

func InitAdminIDs(raw string) {
	adminIDs = make(map[string]struct{})
	for _, id := range strings.Split(raw, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		adminIDs[id] = struct{}{}
	}
}

func AdminRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(UserIDKey).(string)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if _, allowed := adminIDs[userID]; !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

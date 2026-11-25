package transactiontracker

import (
	"crypto/subtle"
	"net/http"
)

func authorized(r *http.Request) bool {
	given := []byte(r.Header.Get("Authorization"))
	expected := []byte("Bearer " + apiToken)
	return apiToken != "" && subtle.ConstantTimeCompare(given, expected) == 1
}

func RequireToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

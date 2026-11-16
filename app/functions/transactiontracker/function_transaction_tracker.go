package transactiontracker

import (
	"io"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("transactionTracker", transactionTracker)
}

func transactionTracker(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}
	log.Printf("plaid webhook %s %s verification=%q body=%s", r.Method, r.URL.Path, r.Header.Get("Plaid-Verification"), body)
	w.WriteHeader(http.StatusOK)
}

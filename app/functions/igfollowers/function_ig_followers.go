package igfollowers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {

	functions.HTTP("fetchIgFollowers", fetchIgFollowers)
}

func fetchIgFollowers(w http.ResponseWriter, r *http.Request) {
	// TODO(franco): load both from Firestore. These are the only values that
	// expire — the session dies on logout, password change, or a forced
	// rotation by Instagram. They are paired: both must come from the same
	// browser session, so always update them together.
	sessionID := ""
	csrfToken := ""

	followers, err := fetchAllFollowers(r.Context(), sessionID, csrfToken)
	if err != nil {
		log.Printf("fetching followers: %v", err)
		http.Error(w, "failed to fetch followers", http.StatusInternalServerError)
		return
	}

	log.Printf("fetched %d followers", len(followers))

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{
		"count":     len(followers),
		"followers": followers,
	})
	if err != nil {
		log.Printf("encoding response: %v", err)
	}
}

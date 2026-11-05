package igfollowers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"francodominguez.com/personal-workflows/app/utils"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

var (
	firestoreClient *firestore.Client
)

func init() {
	ctx := context.Background()
	var err error

	projectId := os.Getenv("PROJECT_ID")

	functions.HTTP("fetchIgFollowers", fetchIgFollowers)
	firestoreClient, err = firestore.NewClient(ctx, projectId)
	if err != nil {
		log.Printf("Error starting client: %v", err)
		return
	}
}

func fetchIgFollowers(w http.ResponseWriter, r *http.Request) {
	// sessionID and csrfToken are the only values that expire — the session
	// dies on logout, password change, or a forced rotation by Instagram.
	// They are paired: both must come from the same browser session, so
	// always update them together in instagram/getFollowersRequestTokens.
	ctx := r.Context()
	var err error
	var sessionID string
	var csrfToken string


	sessionID, err = utils.GetItem[string](ctx, firestoreClient, "instagram", "getFollowersRequestTokens", "sessionID")
	if err != nil {
		// add logs here
		return
	}
	csrfToken, err = utils.GetItem[string](ctx, firestoreClient, "instagram", "getFollowersRequestTokens", "csrfToken")
	if err != nil {
		return
		// add logs here
	}

	followers, err := fetchAllFollowers(r.Context(), sessionID, csrfToken)
	if err != nil {
		log.Printf("fetching followers: %v", err)
		http.Error(w, "failed to fetch followers", http.StatusInternalServerError)
		return
	}

	// Validate format of followers here
	// We don't know all error formats so we should expect the correct one, else it's an error
	// 0 followers is also an error

	// if it's an error, send me a Slack message saying something went wrong fetching the list of followers and return

	// Read the stored list from Firestore: instagram/followers, field "followers",
	// an array of {id, username, fullName} maps (seeded 2026-11-05, 453 entries).
	// Use snapshot.DataTo into a typed struct — GetItem's type assertion can't
	// decode arrays of maps. Treat NotFound as an empty baseline, not an error.

	// Compare the list we have in firestore with the new one, store names of difs in two vars, one for the new ones, one for the ones that are no longer there

	// get webhook URL for the ig sub thing for slack
	// Send me a slack message with the list of followers that are no longer there, names and usernames, using the Slack util func

	// Set the firestore list to be the one we just pulled

	// Debug response: echoes the fetched list back to the caller. Replace with
	// a short status summary (counts of new/lost) once the diff + notify is in.
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

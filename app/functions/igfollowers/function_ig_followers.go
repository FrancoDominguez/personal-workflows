package igfollowers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"francodominguez.com/personal-workflows/app/utils"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

// follower is both the shape extracted from Instagram's API response and the
// schema of the entries stored in the instagram/followers Firestore document.
type follower struct {
	ID       string `firestore:"id" json:"id"`
	Username string `firestore:"username" json:"username"`
	FullName string `firestore:"fullName" json:"fullName"`
}

var firestoreClient *firestore.Client

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

// reportFailure logs the error, alerts Slack (this runs on a schedule, so a
// log line alone would never be seen), and returns a 500 to the caller.
func reportFailure(ctx context.Context, w http.ResponseWriter, webhookUrl string, message string, err error) {
	log.Printf("%s: %v", message, err)
	if webhookUrl != "" {
		slackErr := utils.SendSlackMessage(ctx, webhookUrl, fmt.Sprintf("IG followers job failed — %s: %v", message, err))
		if slackErr != nil {
			log.Printf("sending failure alert to slack: %v", slackErr)
		}
	}
	http.Error(w, message, http.StatusInternalServerError)
}

func fetchIgFollowers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	webhookUrl, err := utils.GetItem[string](ctx, firestoreClient, "slack", "webhookUrls", "igFollowers")
	if err != nil {
		reportFailure(ctx, w, "", "loading slack webhook URL from firestore", err)
		return
	}

	sessionID, err := utils.GetItem[string](ctx, firestoreClient, "instagram", "getFollowersRequestTokens", "sessionID")
	if err != nil {
		reportFailure(ctx, w, webhookUrl, "loading sessionID from firestore", err)
		return
	}
	csrfToken, err := utils.GetItem[string](ctx, firestoreClient, "instagram", "getFollowersRequestTokens", "csrfToken")
	if err != nil {
		reportFailure(ctx, w, webhookUrl, "loading csrfToken from firestore", err)
		return
	}

	current, err := fetchAllFollowers(ctx, sessionID, csrfToken)
	if err != nil {
		reportFailure(ctx, w, webhookUrl, "fetching followers", err)
		return
	}

	if len(current) == 0 {
		reportFailure(ctx, w, webhookUrl, "validating followers", fmt.Errorf("fetched 0 followers"))
		return
	}

	storedList, err := utils.GetItem[[]any](ctx, firestoreClient, "instagram", "followers", "followers")
	if err != nil {
		reportFailure(ctx, w, webhookUrl, "loading stored followers from firestore", err)
		return
	}
	previous := []follower{}
	for _, item := range storedList {
		fields := item.(map[string]any)
		previous = append(previous, follower{
			ID:       fields["id"].(string),
			Username: fields["username"].(string),
			FullName: fields["fullName"].(string),
		})
	}

	added, removed := diffFollowers(previous, current)

	if len(removed) > 0 {
		err = utils.SendSlackMessage(ctx, webhookUrl, unfollowMessage(removed))
		if err != nil {
			// Keep the old baseline: the next run re-detects these
			// unfollowers and retries the notification.
			reportFailure(ctx, w, webhookUrl, "sending unfollow message to slack", err)
			return
		}
	}

	err = utils.SetItem(ctx, firestoreClient, "instagram", "followers", "followers", current)
	if err != nil {
		reportFailure(ctx, w, webhookUrl, "storing followers in firestore", err)
		return
	}

	log.Printf("fetched %d followers (%d new, %d lost)", len(current), len(added), len(removed))

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]any{
		"count": len(current),
		"new":   usernames(added),
		"lost":  usernames(removed),
	})
	if err != nil {
		log.Printf("encoding response: %v", err)
	}
}

// diffFollowers compares by ID: usernames can change, the numeric ID can't.
func diffFollowers(previous []follower, current []follower) ([]follower, []follower) {
	previousIDs := map[string]bool{}
	for _, f := range previous {
		previousIDs[f.ID] = true
	}

	currentIDs := map[string]bool{}
	added := []follower{}
	removed := []follower{}

	for _, f := range current {
		currentIDs[f.ID] = true
		if !previousIDs[f.ID] {
			added = append(added, f)
		}
	}
	for _, f := range previous {
		if !currentIDs[f.ID] {
			removed = append(removed, f)
		}
	}

	return added, removed
}

func unfollowMessage(removed []follower) string {
	message := fmt.Sprintf("%d follower(s) unfollowed you:", len(removed))
	for _, f := range removed {
		if f.FullName != "" {
			message += fmt.Sprintf("\n• %s (@%s)", f.FullName, f.Username)
		} else {
			message += fmt.Sprintf("\n• @%s", f.Username)
		}
	}
	return message
}

func usernames(followers []follower) []string {
	names := []string{}
	for _, f := range followers {
		names = append(names, f.Username)
	}
	return names
}

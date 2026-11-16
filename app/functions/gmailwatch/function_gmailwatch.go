package gmailwatch

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"francodominguez.com/personal-workflows/app/utils"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/gmail/v1"
)

var (
	gmailService *gmail.Service
	topicName    string
)

func init() {
	functions.HTTP("armGmailWatch", armGmailWatch)

	ctx := context.Background()
	var err error

	topicName = os.Getenv("GMAIL_PUBSUB_TOPIC")
	gmailClientId := os.Getenv("GMAIL_CLIENT_ID")
	gmailClientSecret := os.Getenv("GMAIL_CLIENT_SECRET")
	gmailRefreshToken := os.Getenv("GMAIL_REFRESH_TOKEN")

	gmailService, err = utils.InitiateGmailService(ctx, gmailClientId, gmailClientSecret, gmailRefreshToken)
	if err != nil {
		log.Printf("Error starting gmail service: %v", err)
		return
	}
}

// armGmailWatch (re-)issues the Gmail users.watch() that pushes INBOX changes
// to the gmail-notifications Pub/Sub topic consumed by unsub. A watch lives 7
// days, so Cloud Scheduler calls this every 6; calling it early is harmless,
// Gmail just extends the expiry.
func armGmailWatch(w http.ResponseWriter, r *http.Request) {
	log.Print("function start ============================")
	defer log.Print("============================ function end")
	ctx := r.Context()

	res, err := gmailService.Users.Watch("me", &gmail.WatchRequest{
		TopicName:           topicName,
		LabelIds:            []string{"INBOX"},
		LabelFilterBehavior: "INCLUDE",
	}).Context(ctx).Do()
	if err != nil {
		log.Printf("arming gmail watch on %s: %v", topicName, err)
		// Non-2xx makes Cloud Scheduler mark the run failed (and retry).
		http.Error(w, fmt.Sprintf("arming gmail watch: %v", err), http.StatusInternalServerError)
		return
	}

	expires := time.UnixMilli(res.Expiration).UTC().Format(time.RFC3339)
	log.Printf("gmail watch armed on %s: historyId %d, expires %s", topicName, res.HistoryId, expires)
	fmt.Fprintf(w, "gmail watch armed, expires %s\n", expires)
}

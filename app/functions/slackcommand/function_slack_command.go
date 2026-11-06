package slackcommand

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("slackCommand", slackCommand)
}

func slackCommand(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}

	form, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, "bad form body", http.StatusBadRequest)
		return
	}

	if form.Get("ssl_check") == "1" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := verifySlackSignature(r.Header, body, os.Getenv("SLACK_SIGNING_SECRET"), time.Now()); err != nil {
		log.Printf("rejected request: %v", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// The signature only proves the request came from Slack, not who typed it.
	// Anyone in the workspace can use the command, so only allow one user ID.
	if form.Get("user_id") != os.Getenv("SLACK_ALLOWED_USER_ID") {
		log.Printf("user %s (%s) not allowed", form.Get("user_id"), form.Get("user_name"))
		reply(w, "You're not allowed to run this.")
		return
	}

	action := strings.ToLower(strings.TrimSpace(form.Get("text")))
	actionResponse, err := runAction(r.Context(), action)
	if err != nil {
		log.Printf("action %q failed: %v", action, err)
		reply(w, "Something went wrong running "+action+".")
		return
	}

	log.Printf("action response: %s", actionResponse)
	reply(w, actionResponse)
}

func runAction(ctx context.Context, action string) (string, error) {
	return "action response", nil
}

// reply answers the slash command. Always send a 200: Slack shows any other
// status as a generic error and drops the text.
func reply(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response_type": "ephemeral", // only you see it; "in_channel" posts publicly
		"text":          text,
	})
}

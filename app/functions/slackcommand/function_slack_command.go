package slackcommand

import (
	"context"
	"encoding/json"
	"fmt"
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

// slackCommand handles the `/run [action]` slash command. Slack gives a slash
// command 3 seconds to answer, so every action must return quickly: anything
// slow is kicked off elsewhere (see triggerIgFollowers) and only acknowledged here.
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
		reply(w, fmt.Sprintf("Something went wrong running `%s`: %v", action, err))
		return
	}

	log.Printf("action %q: %s", action, actionResponse)
	reply(w, actionResponse)
}

// runAction dispatches on the text after `/run`. Every action returns the
// message shown to the user (ephemeral, only they see it).
func runAction(ctx context.Context, action string) (string, error) {
	switch action {
	case "igfollowers":
		return triggerIgFollowers(ctx)
	case "":
		return "Usage: `/run <action>`. Available actions: `igfollowers`.", nil
	default:
		return fmt.Sprintf("Unknown action `%s`. Available actions: `igfollowers`.", action), nil
	}
}

func reply(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response_type": "ephemeral",
		"text":          text,
	})
}

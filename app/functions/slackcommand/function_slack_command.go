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
	"google.golang.org/api/idtoken"
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
	switch action {
	case "igfollowers":
		triggerIgFollowers(ctx)
		// post request to gcloud run function that triggers workflow that compares ig followers

	}

	return "action response", nil
}

func reply(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response_type": "ephemeral",
		"text":          text,
	})
}

func triggerIgFollowers(ctx context.Context) (string, error) {
	target := os.Getenv("IGFOLLOWERS_URL")

	// Audience == the Cloud Run URL. Cloud Run rejects tokens minted for anything else.
	client, err := idtoken.NewClient(ctx, target)
	if err != nil {
		return "", fmt.Errorf("creating id token client: %w", err)
	}
	client.Timeout = 2 * time.Second // see the Slack 3s note below

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling igfollowers: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("igfollowers returned %d: %s", resp.StatusCode, body)
	}
	return string(body), nil
}

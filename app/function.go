package function

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var (
	firestoreClient *firestore.Client
	gmailService    *gmail.Service
)

func init() {
	ctx := context.Background()
	var err error

	projectId := os.Getenv("PROJECT_ID")
	gmailClientId := os.Getenv("GMAIL_CLIENT_ID")
	gmailClientSecret := os.Getenv("GMAIL_CLIENT_SECRET")
	gmailRefreshToken := os.Getenv("GMAIL_REFRESH_TOKEN")

	firestoreClient, err = firestore.NewClient(ctx, projectId)
	if err != nil {
		log.Printf("Error starting client: %v", err)
		return
	}

	gmailService, err = initiateGmailService(ctx, gmailClientId, gmailClientSecret, gmailRefreshToken)
	if err != nil {
		log.Printf("Error starting gmail service: %v", err)
		return
	}

	functions.HTTP("autoEmailUnsubscribe", autoEmailUnsubscribe)
}

func initiateGmailService(ctx context.Context, gmailClientId string, gmailClientSecret string, gmailRefreshToken string) (*gmail.Service, error) {
	var err error
	oauthConfig := &oauth2.Config{
		ClientID:     gmailClientId,
		ClientSecret: gmailClientSecret,
		Endpoint:     google.Endpoint,
	}
	token := &oauth2.Token{RefreshToken: gmailRefreshToken}
	gmailService, err = gmail.NewService(ctx, option.WithTokenSource(oauthConfig.TokenSource(ctx, token)))
	return gmailService, err
}

func decodeRequestBody(r *http.Request) (*gmailNotification, error) {
	var err error
	var env pushEnvelope

	err = json.NewDecoder(r.Body).Decode(&env)
	if err != nil {
		return nil, err
	}

	var notif gmailNotification
	err = json.Unmarshal(env.Message.Data, &notif)
	if err != nil {
		return nil, err
	}
	return &notif, nil
}

func autoEmailUnsubscribe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error

	var notif *gmailNotification
	notif, err = decodeRequestBody(r)
	if err != nil {
		log.Printf("decoding request body: %v", err)
		return
	}

	log.Printf("mailbox %s changed, historyId %d", notif.EmailAddress, notif.HistoryID)

	document := firestoreClient.Collection("state").Doc("gmail")

	var snapshot *firestore.DocumentSnapshot
	snapshot, err = document.Get(ctx)
	if err != nil {
		log.Printf("fetching cursor document from firestore: %v", err)
		return
	}

	var historyIdValue any
	historyIdValue, err = snapshot.DataAt("historyId")
	if err != nil {
		log.Printf("reading historyId field: %v", err)
		return
	}
	historyId, ok := historyIdValue.(int64)
	if !ok {
		log.Printf("historyId is not an int64, got %T", historyIdValue)
		return
	}

	resp, err := gmailService.Users.History.
		List("me").
		StartHistoryId(uint64(historyId)).
		HistoryTypes("messageAdded").
		Context(ctx).
		Do()
	if err != nil {
		log.Printf("listing history from %d: %v", historyId, err)
		return
	}

	var messageIds []string
	for _, history := range resp.History {
		for _, messageAdded := range history.MessagesAdded{
			messageIds = append(messageIds, messageAdded.Message.Id)
		}
	}

	for _, messageId := range messageIds{
		msg, err := gmailService.Users.Messages.Get("me", messageId).Format("full").Context(ctx).Do()
		if err != nil {
			log.Printf("fetching message %s: %v", messageId, err)
			return
		}

		var textContent string
		var subject string
		var unsub string
		for _, part := range msg.Payload.Parts {
			if part.MimeType == "text/plain" {
				data, err := base64.URLEncoding.DecodeString(part.Body.Data)
				if err != nil {
					log.Printf("decoding body of message %s: %v", messageId, err)
					continue
				}
				textContent = string(data)
			}
		}
		for _, header := range msg.Payload.Headers { 
			if header.Name == "List-Unsubscribe" { unsub = header.Value } 
			if header.Name == "Subject" {subject = header.Value}
		}

		log.Printf("Subject: %s", subject)
		log.Printf("Unsubcribe address: %s", unsub)
		log.Printf("Content: %s", textContent)
	}
}
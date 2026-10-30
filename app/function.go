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
	"google.golang.org/api/gmail/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	historyId, err := getFirestoreItem[int64](ctx, "gmail", "historyId", "historyId")
	if err != nil {
		if status.Code(err) == codes.NotFound {
			err = setFirestoreItem(ctx, "gmail", "historyId", "historyId", int64(notif.HistoryID))
			if err != nil {
				log.Printf("seeding cursor: %v", err)
			}
		}
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
		for _, messageAdded := range history.MessagesAdded {
			messageIds = append(messageIds, messageAdded.Message.Id)
		}
	}

	for _, messageId := range messageIds {
		msg, err := gmailService.Users.Messages.Get("me", messageId).Format("full").Context(ctx).Do()
		if err != nil {
			log.Printf("fetching message %s: %v", messageId, err)
			continue
		}

		var textContent string
		var subject string
		var unsub string
		if msg.Payload.Parts != nil {
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
		} else {
			data, err := base64.URLEncoding.DecodeString(msg.Payload.Body.Data)
			if err != nil {
				log.Printf("decoding body of message %s: %v", messageId, err)
			}
			textContent = string(data)
		}
		for _, header := range msg.Payload.Headers {
			if header.Name == "List-Unsubscribe" {
				unsub = header.Value
			}
			if header.Name == "Subject" {
				subject = header.Value
			}
		}

		log.Printf("Subject: %s", subject)
		log.Printf("Unsubcribe address: %s", unsub)
		log.Print("===========================================")

		// Updating the content list
		contentList, err := getFirestoreItem[[]any](ctx, "gmail", "contentList", "contentList")
		if err != nil && status.Code(err) != codes.NotFound {
			return
		}

		contentList = append(contentList, textContent)
		err = setFirestoreItem(ctx, "gmail", "contentList", "contentList", contentList)
		if err != nil {
			return
		}

		// using JEV, validate whether we want to unsubscribe to these emails here

		// do the unsubscribing here
	}

	err = setFirestoreItem(ctx, "gmail", "historyId", "historyId", int64(resp.HistoryId))
	if err != nil {
		log.Printf("saving cursor: %v", err)
	}
}

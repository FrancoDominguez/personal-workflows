package unsub

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"francodominguez.com/gmail-webhook/app/utils"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	firestoreClient *firestore.Client
	gmailService    *gmail.Service
	typesafeClient  utils.TypesafeClient
)

func init() {
	ctx := context.Background()
	var err error

	projectId := os.Getenv("PROJECT_ID")
	gmailClientId := os.Getenv("GMAIL_CLIENT_ID")
	gmailClientSecret := os.Getenv("GMAIL_CLIENT_SECRET")
	gmailRefreshToken := os.Getenv("GMAIL_REFRESH_TOKEN")
	typesafeApiKey := os.Getenv("TYPESAFE_API_KEY")

	firestoreClient, err = firestore.NewClient(ctx, projectId)
	if err != nil {
		log.Printf("Error starting client: %v", err)
		return
	}

	typesafeClient = utils.NewTypesafeClient(typesafeApiKey)

	gmailService, err = utils.InitiateGmailService(ctx, gmailClientId, gmailClientSecret, gmailRefreshToken)
	if err != nil {
		log.Printf("Error starting gmail service: %v", err)
		return
	}

	functions.HTTP("autoEmailUnsubscribe", autoEmailUnsubscribe)
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

	historyId, err := utils.GetItem[int64](ctx, firestoreClient, "gmail", "historyId", "historyId")
	if err != nil {
		if status.Code(err) == codes.NotFound {
			err = utils.SetItem(ctx, firestoreClient, "gmail", "historyId", "historyId", int64(notif.HistoryID))
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
		var sender string
		var subject string
		var unsub string
		var oneClick bool
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
			if header.Name == "List-Unsubscribe-Post" {
				oneClick = true
			}
			if header.Name == "Subject" {
				subject = header.Value
			}
			if header.Name == "From" {
				sender = header.Value
			}
		}

		log.Printf("Sender: %s", sender)
		log.Printf("Subject: %s", subject)
		log.Printf("Unsubcribe address: %s", unsub)

		if !utils.Unsubable(unsub) {
			log.Printf("unsub unavailable for '%s'", subject)
			continue
		}

		contentList, err := utils.GetItem[[]any](ctx, firestoreClient, "gmail", "contentList", "contentList")
		if err != nil && status.Code(err) != codes.NotFound {
			return
		}

		contentList = append(contentList, textContent)
		err = utils.SetItem(ctx, firestoreClient, "gmail", "contentList", "contentList", contentList)
		if err != nil {
			return
		}

		instructions := "should I unsubscribe from this email? I am not interested in marketing products"
		state := fmt.Sprintf("sender: %s\nsubject: %s\nbody: %s", sender, subject, textContent)
		cutoff := 0.5

		var shouldUnsub bool
		shouldUnsub, err = typesafeClient.SingleNoulQuestion(ctx, instructions, state, cutoff)
		if err != nil {
			log.Printf("asking JEV about message %s: %v", messageId, err)
			return
		}

		log.Printf("JEV's unsubscription assesment: %t", shouldUnsub)

		if !shouldUnsub {
			continue
		}

		err = unsubscribe(ctx, unsub, oneClick)
		if err != nil {
			log.Printf("unsubscribing from '%s': %v", subject, err)
			continue
		}

		log.Printf("unsubscribed from '%s'", sender)
	}

	err = utils.SetItem(ctx, firestoreClient, "gmail", "historyId", "historyId", int64(resp.HistoryId))
	if err != nil {
		log.Printf("saving cursor: %v", err)
	}

	log.Print("===========================================")
}

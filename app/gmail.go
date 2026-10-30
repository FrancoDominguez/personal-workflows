package function

import (
	"context"
	"encoding/base64"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type pushEnvelope struct {
	Message struct {
		Data      []byte `json:"data"`
		MessageID string `json:"messageId"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

type gmailNotification struct {
	EmailAddress string `json:"emailAddress"`
	HistoryID    uint64 `json:"historyId"`
}

type email struct {
	to      string
	subject string
	body    string
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

func sendEmail(ctx context.Context, to string, subject string, body string) error {
	raw := "To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body

	msg := &gmail.Message{Raw: base64.URLEncoding.EncodeToString([]byte(raw))}

	_, err := gmailService.Users.Messages.Send("me", msg).Context(ctx).Do()
	return err
}

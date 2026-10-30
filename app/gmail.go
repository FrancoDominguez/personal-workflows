package function

import (
	"context"
	"encoding/base64"

	"google.golang.org/api/gmail/v1"
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

func sendEmail(ctx context.Context, to string, subject string, body string) error {
	raw := "To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body

	msg := &gmail.Message{Raw: base64.URLEncoding.EncodeToString([]byte(raw))}

	_, err := gmailService.Users.Messages.Send("me", msg).Context(ctx).Do()
	return err
}

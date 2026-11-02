package gmailutil

import (
	"context"
	"encoding/base64"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type UnsubMethod int

const (
	UnsubUnavailable UnsubMethod = iota
	UnsubHttp
	UnsubEmail
)

func GetUnsubMethod(unsub string) UnsubMethod {
	switch {
	case strings.Contains(unsub, "https://"), strings.Contains(unsub, "http://"):
		return UnsubHttp
	case strings.Contains(unsub, "mailto:"):
		return UnsubEmail
	default:
		return UnsubUnavailable
	}
}

func InitiateGmailService(ctx context.Context, gmailClientId string, gmailClientSecret string, gmailRefreshToken string) (*gmail.Service, error) {
	oauthConfig := &oauth2.Config{
		ClientID:     gmailClientId,
		ClientSecret: gmailClientSecret,
		Endpoint:     google.Endpoint,
	}
	token := &oauth2.Token{RefreshToken: gmailRefreshToken}
	return gmail.NewService(ctx, option.WithTokenSource(oauthConfig.TokenSource(ctx, token)))
}

func SendEmail(ctx context.Context, svc *gmail.Service, to string, subject string, body string) error {
	raw := "To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		body

	msg := &gmail.Message{Raw: base64.URLEncoding.EncodeToString([]byte(raw))}

	_, err := svc.Users.Messages.Send("me", msg).Context(ctx).Do()
	return err
}

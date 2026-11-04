package utils

import (
	"context"
	"encoding/base64"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// A List-Unsubscribe header value holds angle-bracketed, comma-separated
// targets, e.g. <mailto:unsub@example.com>, <https://example.com/unsub>.
func getUnsubTarget(unsub string, prefixes ...string) string {
	for _, part := range strings.Split(unsub, ",") {
		target := strings.Trim(strings.TrimSpace(part), "<>")
		for _, prefix := range prefixes {
			if strings.HasPrefix(target, prefix) {
				return target
			}
		}
	}
	return ""
}

func GetUnsubHttpTarget(unsub string) string {
	return getUnsubTarget(unsub, "https://", "http://")
}

func GetUnsubEmailTarget(unsub string) string {
	return getUnsubTarget(unsub, "mailto:")
}

func Unsubable(unsub string) bool {
	return GetUnsubHttpTarget(unsub) != "" || GetUnsubEmailTarget(unsub) != ""
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

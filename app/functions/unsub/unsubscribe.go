package unsub

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"francodominguez.com/gmail-webhook/app/utils"
)

// unsubscribe acts on a List-Unsubscribe header value, preferring the http
// target and falling back to the mailto: target if http fails or is absent.
func unsubscribe(ctx context.Context, unsub string, oneClick bool) error {
	var httpErr error
	if target := utils.GetUnsubHttpTarget(unsub); target != "" {
		log.Print("Unsubscribing with http")
		httpErr = unsubByHttp(ctx, target, oneClick)
		if httpErr == nil {
			return nil
		}
		log.Printf("http unsubscribe failed: %v", httpErr)
	}

	if target := utils.GetUnsubEmailTarget(unsub); target != "" {
		log.Print("Unsubscribing with email")
		return unsubByEmail(ctx, target)
	}

	return httpErr
}

// unsubByHttp calls the unsubscribe URL. When the message also carried a
// List-Unsubscribe-Post header it uses the one-click POST from RFC 8058;
// otherwise it falls back to a plain GET.
func unsubByHttp(ctx context.Context, target string, oneClick bool) error {
	var req *http.Request
	var err error
	if oneClick {
		body := strings.NewReader("List-Unsubscribe=One-Click")
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, target, body)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return err
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("%s returned %s", target, resp.Status)
	}
	return nil
}

// unsubByEmail sends an unsubscribe email to a mailto: target, honouring an
// optional ?subject= query.
func unsubByEmail(ctx context.Context, target string) error {
	u, err := url.Parse(target)
	if err != nil {
		return err
	}

	subject := u.Query().Get("subject")
	if subject == "" {
		subject = "unsubscribe"
	}

	return utils.SendEmail(ctx, gmailService, u.Opaque, subject, "unsubscribe")
}

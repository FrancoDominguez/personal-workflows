package unsub

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"francodominguez.com/personal-workflows/app/utils"
)

func unsubscribe(ctx context.Context, unsub string, oneClick bool) error {
	var httpErr error

	target := utils.GetUnsubHttpTarget(unsub)
	if target != "" {
		log.Print("Unsubscribing with http")
		httpErr = unsubByHttp(ctx, target, oneClick)
		if httpErr == nil {
			return nil
		}
		log.Printf("http unsubscribe failed: %v", httpErr)
	}

	target = utils.GetUnsubEmailTarget(unsub)
	if target != "" {
		log.Print("Unsubscribing with email")
		return unsubByEmail(ctx, target)
	}

	return httpErr
}

func unsubByHttp(ctx context.Context, target string, oneClick bool) error {
	var req *http.Request
	var err error
	if oneClick {
		// one-click POST body required by RFC 8058
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

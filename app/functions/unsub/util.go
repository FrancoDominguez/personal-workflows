package unsub

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"francodominguez.com/gmail-webhook/app/utils"
)

// unsubByHttp hits the http target of the List-Unsubscribe header. When the
// message also carried a List-Unsubscribe-Post header it uses the one-click
// POST from RFC 8058; otherwise it falls back to a plain GET.
func unsubByHttp(ctx context.Context, unsub string, oneClick bool) error {
	target := utils.GetUnsubTarget(unsub, utils.UnsubHttp)
	if target == "" {
		return fmt.Errorf("no http target in %q", unsub)
	}

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

// unsubByEmail sends an unsubscribe email to the mailto: target of the
// List-Unsubscribe header, honouring an optional ?subject= query.
func unsubByEmail(ctx context.Context, unsub string) error {
	target := utils.GetUnsubTarget(unsub, utils.UnsubEmail)
	if target == "" {
		return fmt.Errorf("no mailto target in %q", unsub)
	}

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

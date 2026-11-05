package igfollowers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Snapshot of one logged-in Chrome session (device cookies + fingerprint
// headers). Long-lived: these identify the browser, not the login, and only
// need refreshing if the whole curl is re-copied from DevTools.
const (
	igUserID = "3507767136"

	cookieDatr  = "KwXYaa0mrld4migKh57xnyZB"
	cookieIgDid = "642526F1-9B61-47C7-A168-E5EEEB7D54AA"
	cookieMid   = "adgFLAAEAAFX_y57I8eRgkd79BVu"
	cookieRur   = `"CCO\0543507767136\0541825021045:01ff09c80c19bc752af5d5e37a148cf96445402a129005f5dae0e11ff2761319e84985ef"`

	userAgent  = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
	igAppID    = "936619743392459"
	asbdID     = "359341"
	igWWWClaim = "hmac.AR098PzFO6kYksFJTPLBRnbtJ5UU7djX_ZAU2itz5132QVhf"
)

type followersPage struct {
	Users []struct {
		Pk       string `json:"pk"`
		Username string `json:"username"`
		FullName string `json:"full_name"`
	} `json:"users"`
	NextMaxID string `json:"next_max_id"`
	Status    string `json:"status"`
}

// fetchAllFollowers pages through the follower list until Instagram stops
// returning a next_max_id cursor. Instagram serves ~25 users per page no
// matter what count asks for, so the full list takes multiple requests.
func fetchAllFollowers(ctx context.Context, sessionID, csrfToken string) ([]follower, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var followers []follower
	maxID := ""
	for {
		page, err := fetchPage(ctx, client, sessionID, csrfToken, maxID)
		if err != nil {
			return nil, err
		}

		for _, user := range page.Users {
			followers = append(followers, follower{
				ID:       user.Pk,
				Username: user.Username,
				FullName: user.FullName,
			})
		}

		if page.NextMaxID == "" {
			break
		}
		maxID = page.NextMaxID

		// Instagram aggressively rate-limits accounts that page too fast.
		time.Sleep(time.Second)
	}

	return followers, nil
}

func fetchPage(ctx context.Context, client *http.Client, sessionID, csrfToken, maxID string) (*followersPage, error) {
	query := url.Values{}
	query.Set("count", "200")
	query.Set("search_surface", "follow_list_page")
	if maxID != "" {
		query.Set("max_id", maxID)
	}
	endpoint := fmt.Sprintf("https://www.instagram.com/api/v1/friendships/%s/followers/?%s", igUserID, query.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	cookie := fmt.Sprintf(
		"datr=%s; ig_did=%s; mid=%s; ig_nrcb=1; ds_user_id=%s; ps_l=1; ps_n=1; csrftoken=%s; dpr=2.5; wd=505x689; rur=%s; sessionid=%s",
		cookieDatr, cookieIgDid, cookieMid, igUserID, csrfToken, cookieRur, sessionID,
	)

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en,fr;q=0.9")
	req.Header.Set("cookie", cookie)
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", "https://www.instagram.com/franco_antonio_dominguez/")
	req.Header.Set("sec-ch-prefers-color-scheme", "dark")
	req.Header.Set("sec-ch-ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`)
	req.Header.Set("sec-ch-ua-full-version-list", `"Chromium";v="146.0.7680.178", "Not-A.Brand";v="24.0.0.0", "Google Chrome";v="146.0.7680.178"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-model", `""`)
	req.Header.Set("sec-ch-ua-platform", `"macOS"`)
	req.Header.Set("sec-ch-ua-platform-version", `"27.0.1"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("x-asbd-id", asbdID)
	req.Header.Set("x-csrftoken", csrfToken)
	req.Header.Set("x-ig-app-id", igAppID)
	req.Header.Set("x-ig-max-touch-points", "0")
	req.Header.Set("x-ig-www-claim", igWWWClaim)
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instagram returned %d (session likely expired): %.200s", resp.StatusCode, body)
	}

	var page followersPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing response (got HTML? session likely expired): %w", err)
	}

	// Errors can arrive as well-formed 200 JSON, e.g.
	// {"message": "login_required", "status": "fail"}.
	if page.Status != "ok" {
		return nil, fmt.Errorf("instagram returned status %q: %.200s", page.Status, body)
	}

	return &page, nil
}

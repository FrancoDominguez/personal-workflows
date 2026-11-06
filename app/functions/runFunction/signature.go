package runFunction

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const maxSlackRequestAge = 5 * time.Minute

func verifySlackSignature(h http.Header, body []byte, secret string, now time.Time) error {
	if secret == "" {
		return errors.New("SLACK_SIGNING_SECRET is not set")
	}

	tsHeader := h.Get("X-Slack-Request-Timestamp")
	ts, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return fmt.Errorf("bad timestamp %q", tsHeader)
	}
	age := now.Sub(time.Unix(ts, 0))
	if age > maxSlackRequestAge || age < -maxSlackRequestAge {
		return fmt.Errorf("timestamp out of range (age %s)", age)
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("v0:" + tsHeader + ":"))
	mac.Write(body)
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(h.Get("X-Slack-Signature"))) {
		return errors.New("signature mismatch")
	}
	return nil
}

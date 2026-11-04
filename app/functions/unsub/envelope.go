package unsub

import (
	"encoding/json"
	"net/http"
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

func decodeRequestBody(r *http.Request) (*gmailNotification, error) {
	var err error
	var env pushEnvelope

	err = json.NewDecoder(r.Body).Decode(&env)
	if err != nil {
		return nil, err
	}

	var notif gmailNotification
	err = json.Unmarshal(env.Message.Data, &notif)
	if err != nil {
		return nil, err
	}
	return &notif, nil
}

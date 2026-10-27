package function

type pushEnvelope struct {
    Message struct {
        Data      []byte `json:"data"` // base64 in the JSON → auto-decoded bytes
        MessageID string `json:"messageId"`
    } `json:"message"`
    Subscription string `json:"subscription"`
}

type gmailNotification struct {
    EmailAddress string `json:"emailAddress"`
    HistoryID    uint64 `json:"historyId"`
}
package function

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	functions.HTTP("scanEmail", scanEmail)
}

func scanEmail(w http.ResponseWriter, r *http.Request) {
	log.Printf("Scanning email!")

    var env pushEnvelope
    err := json.NewDecoder(r.Body).Decode(&env); 
	if err != nil {
        http.Error(w, "bad envelope", http.StatusBadRequest)
        return
    }

    var notif gmailNotification
    err = json.Unmarshal(env.Message.Data, &notif); 
	if err != nil {	
        http.Error(w, "bad notification", http.StatusBadRequest)
        return
    }

    log.Printf("mailbox %s changed, historyId %d", notif.EmailAddress, notif.HistoryID)
    w.WriteHeader(http.StatusOK)
}
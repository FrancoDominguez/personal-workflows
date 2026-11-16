package transactiontracker

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"cloud.google.com/go/firestore"
	"francodominguez.com/personal-workflows/app/utils"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	firestoreClient *firestore.Client
	plaidClient     utils.PlaidClient
)

type webhook struct {
	WebhookType string `json:"webhook_type"`
	WebhookCode string `json:"webhook_code"`
	BankID      string `json:"item_id"`
}

func init() {
	functions.HTTP("transactionTracker", transactionTracker)
	plaidClient = utils.NewPlaidClient(os.Getenv("PLAID_CLIENT_ID"), os.Getenv("PLAID_SECRET"), map[string]string{"td": os.Getenv("PLAID_ACCESS_TOKEN")})

	var err error
	firestoreClient, err = firestore.NewClient(context.Background(), os.Getenv("PROJECT_ID"))
	if err != nil {
		log.Printf("creating firestore client: %v", err)
	}
}

func transactionTracker(w http.ResponseWriter, r *http.Request) {
	defer log.Print("===========================================")
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}
	log.Printf("plaid webhook %s %s verification=%q body=%s", r.Method, r.URL.Path, r.Header.Get("Plaid-Verification"), body)

	var hook webhook
	if err := json.Unmarshal(body, &hook); err != nil || hook.WebhookType != "TRANSACTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := syncTransactions(r.Context(), hook.BankID); err != nil {
		log.Printf("syncing bank %s: %v", hook.BankID, err)
		http.Error(w, "sync failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func syncTransactions(ctx context.Context, bankID string) error {
	cursor, err := utils.GetItem[string](ctx, firestoreClient, "plaid", "cursors", bankID)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	for {
		page, err := plaidClient.SyncTransactions(ctx, bankID, cursor)
		if err != nil {
			return err
		}

		for _, t := range page.Added {
			logTransaction("added", t)
		}
		for _, t := range page.Modified {
			logTransaction("modified", t)
		}
		for _, t := range page.Removed {
			logTransaction("removed", t)
		}

		cursor = page.NextCursor
		if err := utils.SetItem(ctx, firestoreClient, "plaid", "cursors", bankID, cursor); err != nil {
			return err
		}
		if !page.HasMore {
			return nil
		}
	}
}

func logTransaction(change string, t utils.PlaidTransaction) {
	name := t.MerchantName
	if name == "" {
		name = t.Name
	}
	log.Printf("transaction %s id=%s account=%s date=%s amount=%.2f pending=%t %q", change, t.TransactionID, t.AccountID, t.Date, t.Amount, t.Pending, name)
}

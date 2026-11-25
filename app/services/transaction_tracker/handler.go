package transactiontracker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"francodominguez.com/personal-workflows/app/utils"
)

var (
	firestoreClient *firestore.Client
	plaidClient     utils.PlaidClient
	timezone        *time.Location
	apiToken        string
)

type webhook struct {
	WebhookType string `json:"webhook_type"`
	WebhookCode string `json:"webhook_code"`
	BankID      string `json:"item_id"`
}

type allowanceEntry struct {
	AmountCents int64  `json:"amountCents"`
	AsOf        string `json:"asOf"`
}

func init() {
	apiToken = os.Getenv("API_TOKEN")
	plaidClient = utils.NewPlaidClient(os.Getenv("PLAID_CLIENT_ID"), os.Getenv("PLAID_SECRET"), map[string]string{"td": os.Getenv("PLAID_ACCESS_TOKEN")})

	var err error
	firestoreClient, err = firestore.NewClient(context.Background(), os.Getenv("PROJECT_ID"))
	if err != nil {
		log.Printf("creating firestore client: %v", err)
	}

	timezone, err = time.LoadLocation("America/Toronto")
	if err != nil {
		log.Printf("loading timezone: %v", err)
		timezone = time.UTC
	}
}

func HandleDailyAllowance(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Amount *float64 `json:"amount"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&request); err != nil {
		http.Error(w, "body must be JSON like {\"amount\": 100}", http.StatusBadRequest)
		return
	}
	if request.Amount == nil || *request.Amount < 0 {
		http.Error(w, "amount is required and cannot be negative", http.StatusBadRequest)
		return
	}
	asOf := time.Now().In(timezone).Format("2006-01-02")

	entry := map[string]any{"amountCents": toCents(*request.Amount), "asOf": asOf}
	err := utils.SetItem(r.Context(), firestoreClient, "transactionTracker", "allowance", "history", firestore.ArrayUnion(entry))
	if err != nil {
		http.Error(w, "could not save the allowance", http.StatusInternalServerError)
		return
	}

	log.Printf("daily allowance set to %.2f as of %s", *request.Amount, asOf)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"dailyAllowance": *request.Amount, "asOf": asOf})
}

func HandleBalance(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	today := time.Now().In(timezone)
	thisMonth := today.Format("2006-01")

	history, err := utils.GetItem[[]allowanceEntry](ctx, firestoreClient, "transactionTracker", "allowance", "history")
	if err != nil && !utils.IsNotFound(err) {
		http.Error(w, "could not load the allowance", http.StatusInternalServerError)
		return
	}

	spent, err := utils.GetItem[int64](ctx, firestoreClient, "transactionTracker", "spent", thisMonth)
	if err != nil && !utils.IsNotFound(err) {
		http.Error(w, "could not load the amount spent", http.StatusInternalServerError)
		return
	}

	var accrued int64
	for day := 1; day <= today.Day(); day++ {
		date := time.Date(today.Year(), today.Month(), day, 0, 0, 0, 0, timezone).Format("2006-01-02")
		accrued += allowanceOn(history, date)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"balance":        toDollars(accrued - spent),
		"accrued":        toDollars(accrued),
		"spent":          toDollars(spent),
		"dailyAllowance": toDollars(allowanceOn(history, today.Format("2006-01-02"))),
		"month":          thisMonth,
	})
}

func HandleWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}
	log.Printf("plaid webhook body=%s", body)

	var hook webhook
	if err := json.Unmarshal(body, &hook); err != nil || hook.WebhookType != "TRANSACTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	today := time.Now().In(timezone)
	firstOfMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, timezone)

	spent, err := recalculateSpent(ctx, hook.BankID, firstOfMonth, today)
	if err != nil {
		log.Printf("recalculating %s: %v", today.Format("2006-01"), err)
		http.Error(w, "recalculation failed", http.StatusInternalServerError)
		return
	}

	// A charge from the end of last month can still post or change for a few days.
	if today.Day() <= 7 {
		lastOfPreviousMonth := firstOfMonth.AddDate(0, 0, -1)
		firstOfPreviousMonth := firstOfMonth.AddDate(0, -1, 0)
		if _, err := recalculateSpent(ctx, hook.BankID, firstOfPreviousMonth, lastOfPreviousMonth); err != nil {
			log.Printf("recalculating %s: %v", firstOfPreviousMonth.Format("2006-01"), err)
			http.Error(w, "recalculation failed", http.StatusInternalServerError)
			return
		}
	}

	webhookUrl, err := utils.GetItem[string](ctx, firestoreClient, "slack", "webhookUrls", "dm")
	if err == nil {
		err = utils.SendSlackMessage(ctx, webhookUrl, fmt.Sprintf("Spending for %s recalculated to $%.2f", today.Format("2006-01"), toDollars(spent)))
	}
	if err != nil {
		log.Printf("sending slack message: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

// recalculateSpent sums every transaction dated from..to that is not in an
// excluded category and stores the total under that month. Plaid amounts are
func recalculateSpent(ctx context.Context, bankID string, from time.Time, to time.Time) (int64, error) {
	transactions, err := plaidClient.GetTransactions(ctx, bankID, from.Format("2006-01-02"), to.Format("2006-01-02"))
	if err != nil {
		return 0, err
	}

	excludedCategories, err := utils.GetItem[[]string](ctx, firestoreClient, "transactionTracker", "config", "excludedCategories")
	if err != nil && !utils.IsNotFound(err) {
		return 0, err
	}

	var spent int64
	for _, t := range transactions {
		if slices.Contains(excludedCategories, t.Category.Primary) || slices.Contains(excludedCategories, t.Category.Detailed) {
			log.Printf("excluded %s amount=%.2f category=%s %q", t.Date, t.Amount, t.Category.Detailed, t.Name)
			continue
		}
		spent += toCents(t.Amount)
	}

	month := from.Format("2006-01")
	if err := utils.SetItem(ctx, firestoreClient, "transactionTracker", "spent", month, spent); err != nil {
		return 0, err
	}
	log.Printf("month %s: spent %.2f across %d transactions", month, toDollars(spent), len(transactions))
	return spent, nil
}

func allowanceOn(history []allowanceEntry, date string) int64 {
	var amount int64
	var asOf string
	for _, entry := range history {
		if entry.AsOf <= date && entry.AsOf >= asOf {
			amount, asOf = entry.AmountCents, entry.AsOf
		}
	}
	return amount
}

func toCents(dollars float64) int64 {
	return int64(math.Round(dollars * 100))
}

func toDollars(cents int64) float64 {
	return float64(cents) / 100
}

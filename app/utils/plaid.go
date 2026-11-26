package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PlaidClient struct {
	baseUrl      string
	clientId     string
	secret       string
	accessTokens map[string]string
	httpClient   *http.Client
}

type PlaidTransaction struct {
	TransactionID string  `json:"transaction_id"`
	AccountID     string  `json:"account_id"`
	Date          string  `json:"date"`
	Amount        float64 `json:"amount"`
	Name          string  `json:"name"`
	MerchantName  string  `json:"merchant_name"`
	Pending       bool    `json:"pending"`
	Category      struct {
		Primary  string `json:"primary"`
		Detailed string `json:"detailed"`
	} `json:"personal_finance_category"`
}

// accessTokens maps bank ID → access token. A bank ID is Plaid's item_id:
// one per bank login, with its own token, and every webhook carries it, so
// the client resolves the token per call. Today there is a single bank (TD)
// and its ID is not configured anywhere, so a map with exactly one entry is
// used for every bank ID. Once a second bank is linked, build the map keyed
// by real IDs (JSON map in the secret, one secret per bank, Firestore, ...)
// and the lookup becomes exact; nothing else needs to change.
func NewPlaidClient(clientId string, secret string, accessTokens map[string]string) PlaidClient {
	return PlaidClient{
		baseUrl:      "https://production.plaid.com",
		clientId:     clientId,
		secret:       secret,
		accessTokens: accessTokens,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *PlaidClient) getAccessToken(bankID string) (string, error) {
	if token, ok := c.accessTokens[bankID]; ok && token != "" {
		return token, nil
	}
	if len(c.accessTokens) == 1 {
		for _, token := range c.accessTokens {
			if token != "" {
				return token, nil
			}
		}
	}
	return "", fmt.Errorf("no access token configured for bank %s", bankID)
}

func (c *PlaidClient) GetTransactions(ctx context.Context, bankID string, startDate string, endDate string) ([]PlaidTransaction, error) {
	accessToken, err := c.getAccessToken(bankID)
	if err != nil {
		return nil, err
	}

	const pageSize = 500
	var transactions []PlaidTransaction
	for {
		var page struct {
			Transactions      []PlaidTransaction `json:"transactions"`
			TotalTransactions int                `json:"total_transactions"`
		}
		err = c.CallPlaid(ctx, "/transactions/get", map[string]any{
			"access_token": accessToken,
			"start_date":   startDate,
			"end_date":     endDate,
			"options":      map[string]any{"count": pageSize, "offset": len(transactions)},
		}, &page)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, page.Transactions...)
		if len(page.Transactions) == 0 || len(transactions) >= page.TotalTransactions {
			return transactions, nil
		}
	}
}

func (c *PlaidClient) RefreshTransactions(ctx context.Context, bankID string) error {
	accessToken, err := c.getAccessToken(bankID)
	if err != nil {
		return err
	}

	var response struct{}
	return c.CallPlaid(ctx, "/transactions/refresh", map[string]any{"access_token": accessToken}, &response)
}

func (c *PlaidClient) CallPlaid(ctx context.Context, path string, body map[string]any, out any) error {
	body["client_id"] = c.clientId
	body["secret"] = c.secret
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseUrl+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(res.Body)
		return fmt.Errorf("plaid %s: %s", path, msg)
	}
	return json.NewDecoder(res.Body).Decode(out)
}

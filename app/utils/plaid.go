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
}

type PlaidSyncPage struct {
	Added      []PlaidTransaction `json:"added"`
	Modified   []PlaidTransaction `json:"modified"`
	Removed    []PlaidTransaction `json:"removed"`
	NextCursor string             `json:"next_cursor"`
	HasMore    bool               `json:"has_more"`
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

func (c *PlaidClient) SyncTransactions(ctx context.Context, bankID string, cursor string) (PlaidSyncPage, error) {
	var page PlaidSyncPage
	accessToken, err := c.getAccessToken(bankID)
	if err != nil {
		return page, err
	}
	err = c.CallPlaid(ctx, "/transactions/sync", map[string]any{
		"access_token": accessToken,
		"cursor":       cursor,
		"count":        500,
	}, &page)
	return page, err
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

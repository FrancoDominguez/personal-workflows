package function

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type JevClient struct {
	baseUrl    string
	apiKey     string
	httpClient *http.Client
}

func NewJevClient(apiKey string) JevClient {
	return JevClient{
		baseUrl:    "https://api.typesafe.ai/v1/systemone",
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

type JevQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
}

type JevNoulRequest struct {
	Model     string                 `json:"model"`
	State     string                 `json:"state"`
	Questions map[string]JevQuestion `json:"questions"`
}

type NoulAnswer struct {
	Type string  `json:"type"`
	Noul float64 `json:"noul"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type JevResponse struct {
	Model   string                `json:"model"`
	Answers map[string]NoulAnswer `json:"answers"`
	Usage   Usage                 `json:"usage"`
}

func (c *JevClient) jevSingleNoulQuestion(ctx context.Context, instructions, state string, cutoff float64) (bool, error) {
	var err error
	var reqBody []byte
	questionName := "singleQuestion"

	reqBody, err = json.Marshal(
		JevNoulRequest{
			Model: "jev-latest",
			State: state,
			Questions: map[string]JevQuestion{
				fmt.Sprintf("%s", questionName): {
					Type:         "noul",
					Instructions: instructions,
				},
			},
		},
	)
	if err != nil {
		return false, err
	}

	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, c.baseUrl, bytes.NewReader(reqBody))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK || (resp.StatusCode >= 500 && resp.StatusCode <= 599) {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return false, fmt.Errorf("jev returned %d: %s", resp.StatusCode, b)
	}

	var out JevResponse
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return false, err
	}

	log.Printf("noul value: %f, input tokens: %d, output tokens: %d", out.Answers[questionName].Noul, out.Usage.InputTokens, out.Usage.OutputTokens)

	if out.Answers[questionName].Noul > cutoff {
		return true, nil
	} else {
		return false, nil
	}
}

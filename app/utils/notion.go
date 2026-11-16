package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func GetNotionPageText(ctx context.Context, token string, pageId string) (string, error) {
	endpoint := "https://api.notion.com/v1/blocks/" + pageId + "/children?page_size=100"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", "2025-09-03")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("notion returned %d: %s", resp.StatusCode, b)
	}

	var page struct {
		Results []map[string]json.RawMessage `json:"results"`
		HasMore bool                         `json:"has_more"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return "", err
	}
	if page.HasMore {
		return "", fmt.Errorf("notion page %s has more than 100 blocks", pageId)
	}

	var lines []string
	for _, block := range page.Results {
		var blockType string
		json.Unmarshal(block["type"], &blockType)

		var content struct {
			RichText []struct {
				PlainText string `json:"plain_text"`
			} `json:"rich_text"`
		}
		json.Unmarshal(block[blockType], &content)

		var text strings.Builder
		for _, segment := range content.RichText {
			text.WriteString(segment.PlainText)
		}
		if text.Len() > 0 {
			lines = append(lines, text.String())
		}
	}
	return strings.Join(lines, "\n"), nil
}

func GetNotionPageIdByTitle(ctx context.Context, token string, title string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"query":  title,
		"filter": map[string]string{"property": "object", "value": "page"},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.notion.com/v1/search", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Notion-Version", "2025-09-03")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("notion returned %d: %s", resp.StatusCode, b)
	}

	var search struct {
		Results []struct {
			Id         string `json:"id"`
			Properties map[string]struct {
				Type  string `json:"type"`
				Title []struct {
					PlainText string `json:"plain_text"`
				} `json:"title"`
			} `json:"properties"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&search); err != nil {
		return "", err
	}

	var matches []string
	for _, result := range search.Results {
		for _, property := range result.Properties {
			if property.Type != "title" {
				continue
			}
			var resultTitle strings.Builder
			for _, segment := range property.Title {
				resultTitle.WriteString(segment.PlainText)
			}
			if resultTitle.String() == title {
				matches = append(matches, result.Id)
			}
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no notion page titled %q", title)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("%d notion pages titled %q", len(matches), title)
	}
}

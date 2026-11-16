package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const notionVersion = "2025-09-03"

type NotionClient struct {
	baseUrl    string
	token      string
	httpClient *http.Client
}

func NewNotionClient(token string) NotionClient {
	return NotionClient{
		baseUrl:    "https://api.notion.com/v1",
		token:      token,
		httpClient: &http.Client{Timeout: 20 * time.Second},
	}
}

type notionRichText struct {
	PlainText string `json:"plain_text"`
}

type notionBlockContent struct {
	RichText []notionRichText `json:"rich_text"`
}

type notionBlock struct {
	Id          string `json:"id"`
	Type        string `json:"type"`
	HasChildren bool   `json:"has_children"`
}

type notionChildrenPage struct {
	Results    []json.RawMessage `json:"results"`
	HasMore    bool              `json:"has_more"`
	NextCursor string            `json:"next_cursor"`
}

// GetPageText returns the plain text of every block on a page, one block per
// line, in page order. Nested blocks (toggles, sub-bullets) are included.
func (c *NotionClient) GetPageText(ctx context.Context, pageId string) (string, error) {
	var lines []string
	if err := c.collectText(ctx, pageId, &lines); err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

func (c *NotionClient) collectText(ctx context.Context, blockId string, lines *[]string) error {
	cursor := ""
	for {
		page, err := c.fetchChildren(ctx, blockId, cursor)
		if err != nil {
			return err
		}

		for _, raw := range page.Results {
			var block notionBlock
			if err := json.Unmarshal(raw, &block); err != nil {
				return fmt.Errorf("decoding notion block: %w", err)
			}

			var byType map[string]json.RawMessage
			if err := json.Unmarshal(raw, &byType); err != nil {
				return fmt.Errorf("decoding notion block: %w", err)
			}

			var content notionBlockContent
			if body, ok := byType[block.Type]; ok {
				if err := json.Unmarshal(body, &content); err != nil {
					return fmt.Errorf("decoding notion %s block: %w", block.Type, err)
				}
			}

			var text strings.Builder
			for _, rt := range content.RichText {
				text.WriteString(rt.PlainText)
			}
			if text.Len() > 0 {
				*lines = append(*lines, text.String())
			}

			if block.HasChildren && block.Type != "child_page" && block.Type != "child_database" {
				if err := c.collectText(ctx, block.Id, lines); err != nil {
					return err
				}
			}
		}

		if !page.HasMore {
			return nil
		}
		cursor = page.NextCursor
	}
}

func (c *NotionClient) fetchChildren(ctx context.Context, blockId string, cursor string) (notionChildrenPage, error) {
	query := url.Values{"page_size": {"100"}}
	if cursor != "" {
		query.Set("start_cursor", cursor)
	}
	endpoint := fmt.Sprintf("%s/blocks/%s/children?%s", c.baseUrl, url.PathEscape(blockId), query.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return notionChildrenPage{}, fmt.Errorf("building notion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Notion-Version", notionVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return notionChildrenPage{}, fmt.Errorf("calling notion: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return notionChildrenPage{}, fmt.Errorf("notion returned %d: %s", resp.StatusCode, b)
	}

	var page notionChildrenPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return notionChildrenPage{}, fmt.Errorf("decoding notion response: %w", err)
	}
	return page, nil
}

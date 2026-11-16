package utils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetPageText(t *testing.T) {
	responses := map[string]string{
		"page-1|": `{
			"results": [
				{"id": "b1", "type": "heading_1", "has_children": false,
				 "heading_1": {"rich_text": [{"plain_text": "System "}, {"plain_text": "prompt"}]}},
				{"id": "b2", "type": "toggle", "has_children": true,
				 "toggle": {"rich_text": [{"plain_text": "Rules"}]}},
				{"id": "b3", "type": "child_page", "has_children": true, "child_page": {"title": "Other"}}
			],
			"has_more": true,
			"next_cursor": "c2"
		}`,
		"page-1|c2": `{
			"results": [
				{"id": "b4", "type": "divider", "has_children": false, "divider": {}},
				{"id": "b5", "type": "code", "has_children": false,
				 "code": {"language": "plain text", "rich_text": [{"plain_text": "You are a helpful bot.\nAnswer briefly."}]}}
			],
			"has_more": false,
			"next_cursor": null
		}`,
		"b2|": `{
			"results": [
				{"id": "b6", "type": "bulleted_list_item", "has_children": false,
				 "bulleted_list_item": {"rich_text": [{"plain_text": "Be polite"}]}}
			],
			"has_more": false,
			"next_cursor": null
		}`,
	}

	var seenVersion, seenAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenVersion = r.Header.Get("Notion-Version")
		seenAuth = r.Header.Get("Authorization")

		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/blocks/"), "/")
		key := parts[0] + "|" + r.URL.Query().Get("start_cursor")
		body, ok := responses[key]
		if !ok {
			t.Errorf("unexpected request %s", key)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("page_size") != "100" {
			t.Errorf("page_size = %q, want 100", r.URL.Query().Get("page_size"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
	defer server.Close()

	client := NewNotionClient("secret")
	client.baseUrl = server.URL

	got, err := client.GetPageText(context.Background(), "page-1")
	if err != nil {
		t.Fatalf("GetPageText: %v", err)
	}

	want := strings.Join([]string{
		"System prompt",
		"Rules",
		"Be polite",
		"You are a helpful bot.\nAnswer briefly.",
	}, "\n")
	if got != want {
		t.Errorf("GetPageText =\n%q\nwant\n%q", got, want)
	}
	if seenVersion != notionVersion {
		t.Errorf("Notion-Version = %q, want %q", seenVersion, notionVersion)
	}
	if seenAuth != "Bearer secret" {
		t.Errorf("Authorization = %q", seenAuth)
	}
}

func TestGetPageTextError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"code": "object_not_found", "message": "Could not find block"})
	}))
	defer server.Close()

	client := NewNotionClient("secret")
	client.baseUrl = server.URL

	_, err := client.GetPageText(context.Background(), "missing")
	if err == nil || !strings.Contains(err.Error(), "object_not_found") {
		t.Fatalf("err = %v, want object_not_found", err)
	}
}

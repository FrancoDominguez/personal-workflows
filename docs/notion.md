# Notion as a prompt store

Prompts for workflows live on Notion pages so they can be edited live without a
deploy. `utils.NotionClient.GetPageText(ctx, pageId)` reads one page and returns
its plain text, one block per line, in page order.

## One-time setup

1. Create an internal integration at https://www.notion.so/profile/integrations.
   Read-only content capability is enough. The secret starts with `ntn_`.
2. On each page (or a parent "Prompts" page), `...` → Connections → add the
   integration. Child pages inherit the connection.
3. The secret lives in Secret Manager as `notion_api_key` (resource in
   `terraform/secrets.tf`, value added by hand). Mount it on the function's
   service with a `secret_key_ref` env like the Slack signing secret, grant the
   runtime SA `secretmanager.secretAccessor` on it, and build the client with
   `utils.NewNotionClient(os.Getenv(...))`.

Making a page public does nothing for the API. Every call needs the token and
the page must be shared with the integration, public or not.

## What the util does

- Calls `GET /v1/blocks/{pageId}/children` with `Notion-Version: 2025-09-03`.
  A page's body is its child blocks, so the page id is the block id.
- Follows `next_cursor` pagination (100 blocks per call).
- Recurses into blocks with `has_children` (toggles, nested bullets), but not
  into `child_page` or `child_database` blocks, so a page that links to other
  pages only returns its own text.
- Joins every rich-text segment of a block (bold or a link splits one paragraph
  into several segments) and skips blocks with no text (dividers, images).

## Writing prompts on the page

Put the whole prompt in a single code block. It comes back as one line with its
whitespace intact and Notion shows it in a monospace box that is still editable.
Headings and bullets work too; they come back as plain lines with no markers.

## Gotchas

- Page id = the 32 hex characters at the end of the page URL. Dashes are
  optional.
- A 404 `object_not_found` almost always means the page is not shared with the
  integration, not that the id is wrong.
- Rate limit is about 3 requests per second per integration. One read per
  workflow run is nothing.

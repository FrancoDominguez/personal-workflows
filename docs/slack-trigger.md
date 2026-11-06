# Slack trigger (`/run <action>`)

Run workflows on demand from Slack. Typing `/run igfollowers` fires the same job that
runs every Monday.

```
/run igfollowers ─POST─▶ trigger (public Cloud Run, verifies Slack signature + user ID)
                           └─ Cloud Scheduler jobs.run(igfollowers-weekly) → replies "Started…" in < 3s
                                 └─▶ igfollowers (private, OIDC, same path as the weekly run) ─▶ Slack webhook
```

## Why it's built this way

- **Slack can't authenticate to Cloud Run.** A slash command only lets you set a
  Request URL. Slack always sends the same form-encoded POST, with `X-Slack-Signature`
  and `X-Slack-Request-Timestamp`, and you can't add an `Authorization` header. So the
  URL Slack hits has to be public (`allUsers` invoker), and the Go code checks the HMAC
  signature.
- **Slack gives up after 3 seconds.** igfollowers takes longer than that, so the trigger
  never does the work itself. It presses "run now" on the existing scheduler job, which
  calls the private service with its own OIDC token.
- **Don't reply first and keep working in a goroutine.** Cloud Run throttles CPU after
  the response is sent, so the work can stall. Firing the scheduler job sidesteps this.

## What Slack sends (you can't add fields to it)

The body is fixed: `command`, `text` (whatever you typed after the command), `user_id`,
`user_name`, `channel_id`, `response_url`, and so on. To choose what runs, you can use:

- **`text`** (what this repo uses): one `/run` command, where the action is the word you
  type after it.
- **`command`**: several slash commands (`/igfollowers`, `/unsub`) all pointing at the
  same URL, switching on `command`.
- **A query string on the Request URL** (`…?action=x`). This works, but the signature
  covers only the body, not the URL, so prefer the two options above.

## Adding an action

1. Give the target a Cloud Scheduler job, the way `igfollowers_weekly` is set up.
2. Pass the job's `.id` to the `trigger` service as an env var in
   `terraform/service_trigger.tf`.
3. Map the action name to that env var in `actions` in
   `app/functions/trigger/actions.go`.

## One-time setup

1. Create a Slack app at api.slack.com/apps (or reuse the one that owns the igfollowers
   webhook). Copy the **Signing Secret** from **Basic Information**.
2. Add `slack_allowed_user_id` to `terraform.tfvars`. To find your member ID, open your
   Slack profile → ⋮ → **Copy member ID**.
3. Cloud Run won't create a revision if the secret has no version, so create the secret
   first, add its value, then apply everything:
   ```sh
   terraform apply -target=google_secret_manager_secret.slack_signing_secret
   printf '%s' 'SIGNING_SECRET' | gcloud secrets versions add slack_signing_secret --data-file=-
   terraform apply
   terraform output trigger_url
   ```
4. In the Slack app, go to **Slash Commands** → create `/run` → Request URL =
   `trigger_url`. Then reinstall the app to your workspace.
5. Deploy the code: CI's `gcloud run deploy trigger` line replaces the placeholder image.

## Gotchas

- `igfollowers` only posts to Slack when someone unfollowed you. A manual run with no
  changes ends silently: you get only the trigger's "Started igfollowers" reply.
- `roles/cloudscheduler.jobRunner` is granted project-wide because Cloud Scheduler has no
  per-job IAM. It can only force-run existing jobs.
- Every function package's `init()` runs in every service, because they all share one
  image. That's why `trigger` sets `PROJECT_ID` even though it doesn't use Firestore.

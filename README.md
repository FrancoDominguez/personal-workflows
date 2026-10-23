# gmail-webhook

Logs "you got a new email" whenever something lands in my Gmail inbox. Costs ~$0/month
(everything fits in GCP free tiers). Deliberately minimal — built brick by brick; this is brick one.

Gmail can't call arbitrary webhooks — its only push mechanism is `users.watch()` → a Pub/Sub
topic. So the whole thing is:

```
scripts/watch.py (run locally) ─── arms the watch ───▶ Gmail
Gmail inbox change ──▶ Pub/Sub topic (gmail-notify) ──▶ Cloud Run function (app/) ──▶ log line
```

## Setup (one time)

1. Create a GCP project with billing enabled, then `gcloud auth application-default login`.
2. In the console: **APIs & Services → OAuth consent screen** → External, add your Gmail address
   as a test user. Then **Credentials → Create credentials → OAuth client ID → Desktop app** →
   note the client id and secret. (Needed by `watch.py`, not by the deployed function.)
3. ```sh
   cd terraform
   cp terraform.tfvars.example terraform.tfvars   # fill in your project id
   terraform init
   terraform apply   # if the first apply trips on freshly-enabled APIs, just re-run it
   ```
4. Arm the watch (opens a browser for OAuth):
   ```sh
   pip install google-auth-oauthlib google-api-python-client
   python scripts/watch.py CLIENT_ID CLIENT_SECRET PROJECT_ID
   ```

## Try it

Send yourself an email, then:

```sh
gcloud functions logs read gmail-webhook --region northamerica-northeast1 --project YOUR_PROJECT
```

## Notes

- A watch expires after **7 days** — re-run `watch.py` to keep it alive. Automating the renewal
  is a later brick.
- Gmail notifies on any inbox *change* (new mail, but also archiving/labeling), and the message
  carries only a `historyId` cursor — no content. Fetching the actual email is a later brick.
- The function needs no secrets: Gmail publishes into the topic, the function just logs.

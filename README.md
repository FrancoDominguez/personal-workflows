# gmail-webhook

Logs every email that lands in my Gmail inbox. Costs ~$0/month (everything fits in GCP free tiers).

Gmail can't call arbitrary webhooks — its only push mechanism is `users.watch()` → a Pub/Sub
topic. So the whole thing is:

```
Gmail inbox change ──▶ Pub/Sub topic (gmail-notify) ──▶ Cloud Run function (app/main.py) ──▶ logs
Cloud Scheduler (daily "renew") ──▶ same topic ────────▶ same function re-arms the 7-day watch
```

The Pub/Sub payload contains no mail content, just a `historyId` cursor. The function keeps the
last cursor in a GCS object and fetches whatever messages were added since.

## Setup (one time)

1. Create a GCP project with billing enabled, then `gcloud auth application-default login`.
2. In the console: **APIs & Services → OAuth consent screen** → External, add your Gmail address
   as a test user. Then **Credentials → Create credentials → OAuth client ID → Desktop app** →
   note the client id and secret.
3. Get a refresh token for your Gmail account:
   ```sh
   pip install google-auth-oauthlib
   python scripts/get_refresh_token.py CLIENT_ID CLIENT_SECRET
   ```
4. `cp terraform.tfvars.example terraform.tfvars` and fill it in.
5. ```sh
   terraform init
   terraform apply   # if the first apply trips on freshly-enabled APIs, just re-run it
   ```
6. Arm the watch now instead of waiting for the daily scheduler:
   ```sh
   gcloud pubsub topics publish gmail-notify --message=renew --project YOUR_PROJECT
   ```

## Try it

Send yourself an email, then:

```sh
gcloud functions logs read gmail-webhook --region northamerica-northeast1 --project YOUR_PROJECT
```

You should see the From, Subject, and body.

## Notes

- OAuth consent screens in "Testing" mode expire refresh tokens after 7 days — either publish
  the consent screen (still private to you) or expect to re-run step 3 weekly.
- The Gmail secrets live as plain env vars on the function and in the Terraform state
  (gitignored, local). Fine for a toy; move them to Secret Manager before this does anything real.
- `history.list` results aren't paginated here; fine for a personal inbox's burst sizes.

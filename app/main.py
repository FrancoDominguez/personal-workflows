"""Logs every email that lands in the Gmail inbox.

Two kinds of Pub/Sub messages arrive here:
  - b"renew" (from Cloud Scheduler, daily): re-arm the Gmail watch, which
    expires after 7 days.
  - Gmail notifications: {"emailAddress": ..., "historyId": ...}. The payload
    carries no mail content, only a cursor — we diff against the last cursor
    we stored in GCS and fetch whatever messages were added since.
"""

import base64
import json
import os

import functions_framework
from google.cloud import storage
from google.oauth2.credentials import Credentials
from googleapiclient.discovery import build


def gmail():
    creds = Credentials(
        token=None,
        refresh_token=os.environ["GMAIL_REFRESH_TOKEN"],
        client_id=os.environ["GMAIL_CLIENT_ID"],
        client_secret=os.environ["GMAIL_CLIENT_SECRET"],
        token_uri="https://oauth2.googleapis.com/token",
    )
    return build("gmail", "v1", credentials=creds)


def state_blob():
    return storage.Client().bucket(os.environ["STATE_BUCKET"]).blob("last_history_id")


def body_text(payload):
    """First text/plain part in the MIME tree, or ""."""
    if payload["mimeType"] == "text/plain" and payload["body"].get("data"):
        return base64.urlsafe_b64decode(payload["body"]["data"]).decode(errors="replace")
    for part in payload.get("parts", []):
        if text := body_text(part):
            return text
    return ""


@functions_framework.cloud_event
def handle(event):
    data = base64.b64decode(event.data["message"]["data"]).decode()

    if data == "renew":
        resp = gmail().users().watch(
            userId="me", body={"topicName": os.environ["TOPIC"], "labelIds": ["INBOX"]}
        ).execute()
        if not state_blob().exists():
            state_blob().upload_from_string(resp["historyId"])
        print(f"watch renewed, expires {resp['expiration']}")
        return

    notification = json.loads(data)
    blob = state_blob()
    start = blob.download_as_text() if blob.exists() else str(notification["historyId"])

    svc = gmail()
    history = svc.users().history().list(
        userId="me", startHistoryId=start, historyTypes=["messageAdded"]
    ).execute()

    for record in history.get("history", []):
        for added in record.get("messagesAdded", []):
            msg = svc.users().messages().get(userId="me", id=added["message"]["id"]).execute()
            headers = {h["name"]: h["value"] for h in msg["payload"]["headers"]}
            print(f"From: {headers.get('From')}")
            print(f"Subject: {headers.get('Subject')}")
            print(body_text(msg["payload"]) or msg.get("snippet", ""))

    blob.upload_from_string(str(notification["historyId"]))

"""Arm (or re-arm) the Gmail watch: tells Gmail to publish to our Pub/Sub topic
on every inbox change. A watch expires after 7 days — re-run this to keep it alive.

    pip install google-auth-oauthlib google-api-python-client
    python watch.py CLIENT_ID CLIENT_SECRET PROJECT_ID
"""

import sys

from google_auth_oauthlib.flow import InstalledAppFlow
from googleapiclient.discovery import build

client_id, client_secret, project = sys.argv[1], sys.argv[2], sys.argv[3]

flow = InstalledAppFlow.from_client_config(
    {
        "installed": {
            "client_id": client_id,
            "client_secret": client_secret,
            "auth_uri": "https://accounts.google.com/o/oauth2/auth",
            "token_uri": "https://oauth2.googleapis.com/token",
        }
    },
    scopes=["https://www.googleapis.com/auth/gmail.readonly"],
)
creds = flow.run_local_server(port=0)

resp = build("gmail", "v1", credentials=creds).users().watch(
    userId="me",
    body={"topicName": f"projects/{project}/topics/gmail-notify", "labelIds": ["INBOX"]},
).execute()
print(f"watching until (epoch ms): {resp['expiration']}")

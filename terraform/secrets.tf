resource "google_secret_manager_secret" "gmail_refresh_token" {
  secret_id = "gmail-refresh-token"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret" "gmail_client_secret" {
  secret_id = "gmail_client_secret"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret" "typesafe_api_key" {
  secret_id = "typesafe_api_key"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_iam_member" "function_runtime_reads_gmail_refresh_token" {
  secret_id = google_secret_manager_secret.gmail_refresh_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.function_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "function_runtime_reads_gmail_client_secret" {
  secret_id = google_secret_manager_secret.gmail_client_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.function_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "function_runtime_reads_typesafe_api_key" {
  secret_id = google_secret_manager_secret.typesafe_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.function_runtime.email}"
}

# Slack app "Signing Secret" (api.slack.com/apps → Basic Information). The
# trigger service uses it to verify that requests really come from Slack. The
# value is added by hand, not from terraform: see docs/slack-trigger.md.
resource "google_secret_manager_secret" "slack_signing_secret" {
  secret_id = "slack_signing_secret"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_iam_member" "trigger_runtime_reads_slack_signing_secret" {
  secret_id = google_secret_manager_secret.slack_signing_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.trigger_runtime.email}"
}

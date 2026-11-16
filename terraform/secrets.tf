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

resource "google_secret_manager_secret" "slack_signing_secret" {
  secret_id = "slack_signing_secret"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret" "notion_api_key" {
  secret_id = "notion_api_key"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret" "plaid_secret" {
  secret_id = "plaid_secret"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret" "plaid_access_token" {
  secret_id = "plaid_access_token"
  replication {
    auto {}
  }
  depends_on = [google_project_service.apis]
}

resource "google_secret_manager_secret_iam_member" "unsub_runtime_reads_gmail_refresh_token" {
  secret_id = google_secret_manager_secret.gmail_refresh_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.unsub_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "unsub_runtime_reads_gmail_client_secret" {
  secret_id = google_secret_manager_secret.gmail_client_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.unsub_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "unsub_runtime_reads_typesafe_api_key" {
  secret_id = google_secret_manager_secret.typesafe_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.unsub_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "gmailwatch_runtime_reads_gmail_refresh_token" {
  secret_id = google_secret_manager_secret.gmail_refresh_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.gmailwatch_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "gmailwatch_runtime_reads_gmail_client_secret" {
  secret_id = google_secret_manager_secret.gmail_client_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.gmailwatch_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "slack_command_runtime_reads_slack_signing_secret" {
  secret_id = google_secret_manager_secret.slack_signing_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.slack_command_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "unsub_runtime_reads_notion_api_key" {
  secret_id = google_secret_manager_secret.notion_api_key.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.unsub_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "transaction_tracker_runtime_reads_plaid_secret" {
  secret_id = google_secret_manager_secret.plaid_secret.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.transaction_tracker_runtime.email}"
}

resource "google_secret_manager_secret_iam_member" "transaction_tracker_runtime_reads_plaid_access_token" {
  secret_id = google_secret_manager_secret.plaid_access_token.secret_id
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.transaction_tracker_runtime.email}"
}

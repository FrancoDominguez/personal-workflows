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
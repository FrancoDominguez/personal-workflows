resource "google_service_account" "transaction_tracker_runtime" {
  account_id   = "transaction-tracker-runtime"
  display_name = "Transaction Tracker Runtime"
  depends_on   = [google_project_service.apis]
}

resource "google_service_account_iam_member" "ci_acts_as_transaction_tracker_runtime" {
  service_account_id = google_service_account.transaction_tracker_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_cloud_run_v2_service" "transaction_tracker" {
  name     = "transaction-tracker"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"
  template {
    service_account = google_service_account.transaction_tracker_runtime.email
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "FUNCTION_TARGET"
        value = "transactionTracker"
      }
    }
  }
  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      client,
      client_version,
    ]
  }
  depends_on = [google_project_service.apis]
}

resource "google_cloud_run_v2_service_iam_member" "transaction_tracker_public_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.transaction_tracker.location
  name     = google_cloud_run_v2_service.transaction_tracker.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

output "transaction_tracker_webhook_url" {
  value       = google_cloud_run_v2_service.transaction_tracker.uri
  description = "Public URL of the transaction-tracker service, for the Plaid webhook"
}

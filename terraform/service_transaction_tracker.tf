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

resource "google_project_iam_member" "transaction_tracker_runtime_uses_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.transaction_tracker_runtime.email}"
}

resource "google_cloud_run_v2_service" "transaction_tracker" {
  name     = "transaction-tracker"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"
  template {
    service_account                  = google_service_account.transaction_tracker_runtime.email
    max_instance_request_concurrency = 1
    scaling {
      max_instance_count = 1
    }
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "PLAID_CLIENT_ID"
        value = var.plaid_client_id
      }
      env {
        name = "PLAID_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.plaid_secret.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "PLAID_ACCESS_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.plaid_access_token.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "API_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.transaction_tracker_api_token.secret_id
            version = "latest"
          }
        }
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

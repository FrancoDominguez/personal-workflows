resource "google_service_account" "run_runtime" {
  account_id   = "run-runtime"
  display_name = "Run Runtime"
  depends_on   = [google_project_service.apis]
}

resource "google_service_account_iam_member" "ci_acts_as_run_runtime" {
  service_account_id = google_service_account.run_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_cloud_run_v2_service" "run" {
  name     = "run"
  location = var.region
  ingress = "INGRESS_TRAFFIC_ALL"
  template {
    service_account                  = google_service_account.run_runtime.email
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "FUNCTION_TARGET"
        value = "run"
      }
      env {
        name = "SLACK_SIGNING_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.slack_signing_secret.secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "SLACK_ALLOWED_USER_ID"
        value = var.slack_allowed_user_id
      }
    }
  }
  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
  depends_on = [google_project_service.apis]
}

resource "google_cloud_run_v2_service_iam_member" "run_public_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.run.location
  name     = google_cloud_run_v2_service.run.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Paste this as the Request URL of the slash command in the Slack app config.
output "run_slack_request_url" {
  value       = google_cloud_run_v2_service.run.uri
  description = "Public URL of the run service, for the Slack slash command"
}
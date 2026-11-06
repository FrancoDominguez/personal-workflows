resource "google_service_account" "slack_command_runtime" {
  account_id   = "slack-command-runtime"
  display_name = "Slack Command Runtime"
  depends_on   = [google_project_service.apis]
}

resource "google_service_account_iam_member" "ci_acts_as_slack_command_runtime" {
  service_account_id = google_service_account.slack_command_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_cloud_run_v2_service" "slack_command" {
  name     = "slack-command"
  location = var.region
  ingress  = "INGRESS_TRAFFIC_ALL"
  template {
    service_account = google_service_account.slack_command_runtime.email
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      env {
        name  = "FUNCTION_TARGET"
        value = "slackCommand"
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

resource "google_cloud_run_v2_service_iam_member" "slack_command_public_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.slack_command.location
  name     = google_cloud_run_v2_service.slack_command.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Paste this as the Request URL of the slash command in the Slack app config.
output "slack_command_request_url" {
  value       = google_cloud_run_v2_service.slack_command.uri
  description = "Public URL of the slack-command service, for the Slack slash command"
}

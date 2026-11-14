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

# Lets the service force-run Cloud Scheduler jobs. Cloud Scheduler has no
# per-job IAM, so this is project-wide; igfollowers-weekly is the only job.
resource "google_project_iam_member" "slack_command_runs_scheduler_jobs" {
  project = var.project_id
  role    = "roles/cloudscheduler.jobRunner"
  member  = "serviceAccount:${google_service_account.slack_command_runtime.email}"
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
      env {
        # `/run igfollowers` runs this Cloud Scheduler job now (RunJob) rather
        # than calling the igfollowers service: the scrape outlives Slack's 3s
        # reply window, and a run that Slack hangs up on would be cancelled.
        name  = "IGFOLLOWERS_JOB"
        value = google_cloud_scheduler_job.igfollowers_weekly.id
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

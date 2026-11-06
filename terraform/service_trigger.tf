# Public entrypoint for Slack slash commands ("/run <action>").
#
# Slack can't attach Google credentials to a slash-command request, so this is
# the one service that allows unauthenticated invocations. It is protected in
# code instead: app/functions/trigger verifies Slack's HMAC signature and only
# accepts SLACK_ALLOWED_USER_ID. It never does the work itself: it fires the
# matching Cloud Scheduler job with jobs.run, so the target services stay
# private and keep their existing OIDC auth. See docs/slack-trigger.md.

# Dedicated runtime identity: it can run scheduler jobs and read the Slack
# signing secret, nothing else.
resource "google_service_account" "trigger_runtime" {
  account_id   = "trigger-runtime"
  display_name = "Slack Trigger Runtime"
  depends_on   = [google_project_service.apis]
}

# Cloud Scheduler has no per-job IAM, so this has to be project-wide. jobRunner
# can only force-run existing jobs; it can't create, edit, or delete them.
resource "google_project_iam_member" "trigger_runtime_runs_scheduler_jobs" {
  project = var.project_id
  role    = "roles/cloudscheduler.jobRunner"
  member  = "serviceAccount:${google_service_account.trigger_runtime.email}"
}

# CI's `gcloud run deploy trigger` must be able to act as the runtime SA.
resource "google_service_account_iam_member" "ci_acts_as_trigger_runtime" {
  service_account_id = google_service_account.trigger_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_cloud_run_v2_service" "trigger" {
  name     = "trigger"
  location = var.region
  template {
    service_account                  = google_service_account.trigger_runtime.email
    max_instance_request_concurrency = 1
    # Slack gives up after 3s; the handler only makes one jobs.run call.
    timeout = "10s"
    scaling {
      max_instance_count = 1
    }
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
      # Not used by the trigger itself, but every function package's init()
      # runs in every service (they share one image), and the Firestore
      # clients log an error at startup without it.
      env {
        name  = "PROJECT_ID"
        value = var.project_id
      }
      # The functions-framework reads this at runtime to pick which registered
      # function to serve, so the same app image works for every function.
      env {
        name  = "FUNCTION_TARGET"
        value = "slackTrigger"
      }
      env {
        name  = "SLACK_ALLOWED_USER_ID"
        value = var.slack_allowed_user_id
      }
      # One env var per action. The Go `actions` map points at these names.
      # The id is the full "projects/<p>/locations/<r>/jobs/<name>" path that
      # jobs.run expects.
      env {
        name  = "IGFOLLOWERS_JOB"
        value = google_cloud_scheduler_job.igfollowers_weekly.id
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
    }
  }
  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
  depends_on = [
    google_project_service.apis,
    # Cloud Run checks secret access when it creates a revision.
    google_secret_manager_secret_iam_member.trigger_runtime_reads_slack_signing_secret,
  ]
}

# This is what makes the URL reachable by Slack. The signature check in Go is
# what keeps it from being reachable by everyone else.
resource "google_cloud_run_v2_service_iam_member" "trigger_public" {
  project  = var.project_id
  location = google_cloud_run_v2_service.trigger.location
  name     = google_cloud_run_v2_service.trigger.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Paste this into the Slack app's slash command "Request URL".
output "trigger_url" {
  value = google_cloud_run_v2_service.trigger.uri
}

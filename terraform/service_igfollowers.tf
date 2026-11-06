# Dedicated runtime identity for igfollowers, so it doesn't share unsub's
# unsub-runtime account (which can read the Gmail and Typesafe secrets).
# igfollowers only needs Firestore: it reads its Slack webhook + Instagram
# tokens and writes the follower list there.
resource "google_service_account" "igfollowers_runtime" {
  account_id   = "igfollowers-runtime"
  display_name = "IG Followers Runtime"
  depends_on   = [google_project_service.apis]
}

# Firestore IAM can't be scoped below the database, so this is project-wide
# datastore access — still far narrower than unsub-runtime's grants.
resource "google_project_iam_member" "igfollowers_runtime_uses_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.igfollowers_runtime.email}"
}

# CI's `gcloud run deploy igfollowers` must be able to act as the runtime SA.
resource "google_service_account_iam_member" "ci_acts_as_igfollowers_runtime" {
  service_account_id = google_service_account.igfollowers_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_cloud_run_v2_service" "igfollowers" {
  name     = "igfollowers"
  location = var.region
  template {
    service_account                  = google_service_account.igfollowers_runtime.email
    max_instance_request_concurrency = 1
    timeout                          = "300s"
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
        name  = "FUNCTION_TARGET"
        value = "fetchIgFollowers"
      }
    }
  }
  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
  depends_on = [google_project_service.apis]
}

resource "google_service_account" "igfollowers_scheduler" {
  account_id   = "igfollowers-scheduler"
  display_name = "IG Followers Scheduler Invoker"
  depends_on   = [google_project_service.apis]
}

resource "google_cloud_run_v2_service_iam_member" "igfollowers_scheduler_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.igfollowers.location
  name     = google_cloud_run_v2_service.igfollowers.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.igfollowers_scheduler.email}"
}

resource "google_cloud_scheduler_job" "igfollowers_weekly" {
  name      = "igfollowers-weekly"
  region    = var.region
  schedule  = "0 9 * * 1" # Mondays at 09:00
  time_zone = "America/Montreal"

  http_target {
    http_method = "GET"
    uri         = google_cloud_run_v2_service.igfollowers.uri
    oidc_token {
      service_account_email = google_service_account.igfollowers_scheduler.email
      audience              = google_cloud_run_v2_service.igfollowers.uri
    }
  }

  depends_on = [google_project_service.apis]
}

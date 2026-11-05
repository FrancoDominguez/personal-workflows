resource "google_project_service" "cloudscheduler" {
  project            = var.project_id
  service            = "cloudscheduler.googleapis.com"
  disable_on_destroy = true
}

resource "google_cloud_run_v2_service" "igfollowers" {
  name     = "igfollowers"
  location = var.region
  template {
    service_account                  = google_service_account.function_runtime.email
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
      # The functions-framework reads this at runtime to pick which registered
      # function to serve, so the same app image works for every function.
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

  depends_on = [google_project_service.cloudscheduler]
}

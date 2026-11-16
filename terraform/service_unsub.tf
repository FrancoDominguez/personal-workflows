resource "google_service_account" "unsub_runtime" {
  account_id   = "unsub-runtime"
  display_name = "Unsub Runtime"
  depends_on   = [google_project_service.apis]
}

resource "google_project_iam_member" "unsub_runtime_uses_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.unsub_runtime.email}"
}

resource "google_cloud_run_v2_service" "unsub" {
  name     = "unsub"
  location = var.region
  template {
    service_account                  = google_service_account.unsub_runtime.email
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
        value = "autoEmailUnsubscribe"
      }
      env {
        name = "GMAIL_REFRESH_TOKEN"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.gmail_refresh_token.secret_id
            version = "latest"
          }
        }
      }
      env {
        name  = "GMAIL_CLIENT_ID"
        value = var.gcp_client_id
      }
      env {
        name = "GMAIL_CLIENT_SECRET"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.gmail_client_secret.secret_id
            version = "latest"
          }
        }
      }
      env {
        name = "TYPESAFE_API_KEY"
        value_source {
          secret_key_ref {
            secret  = google_secret_manager_secret.typesafe_api_key.secret_id
            version = "latest"
          }
        }
      }
    }
  }
  lifecycle {
    ignore_changes = [template[0].containers[0].image]
  }
  depends_on = [google_project_service.apis]
}

resource "google_project_iam_member" "ci_run_developer" {
  project = var.project_id
  role    = "roles/run.developer"
  member  = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_service_account_iam_member" "ci_acts_as_unsub_runtime" {
  service_account_id = google_service_account.unsub_runtime.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_cloud_run_v2_service_iam_member" "me_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.unsub.location
  name     = google_cloud_run_v2_service.unsub.name
  role     = "roles/run.invoker"
  member   = "user:${var.test_user_account_email}"
}

# Scheduled re-activation resources for the email watcher subscription

resource "google_service_account" "subscription_reactivation_runtime" {
  account_id   = "igfollowers-runtime"
  display_name = "IG Followers Runtime"
  depends_on   = [google_project_service.apis]
}

resource "google_project_iam_member" "igfollowers_runtime_uses_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.igfollowers_runtime.email}"
}

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
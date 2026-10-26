resource "google_service_account" "function_runtime" {
    account_id = "function-runtime"
    display_name = "Function Runtime"
    depends_on = [ google_project_service.apis ]
}

resource "google_cloud_run_v2_service" "function" {
  name = "my-first-function"
  location = var.region
  template {
    service_account = google_service_account.function_runtime.email
    containers {
      image = "us-docker.pkg.dev/cloudrun/container/hello"
        env {
            name = "GMAIL_REFRESH_TOKEN"
            value_source {
                secret_key_ref {
                    secret  = google_secret_manager_secret.gmail_refresh_token.secret_id
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
  role = "roles/run.developer"
  member = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_service_account_iam_member" "ci_acts_as_runtime" {
  service_account_id = google_service_account.function_runtime.name
  role = "roles/iam.serviceAccountUser"
  member = "serviceAccount:${google_service_account.ci.email}"
}

resource "google_cloud_run_v2_service_iam_member" "me_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.function.location
  name     = google_cloud_run_v2_service.function.name
  role     = "roles/run.invoker"
  member   = "user:${var.test_user_account_email}"
}
resource "google_artifact_registry_repository" "functions" {
  project                = var.project_id
  location               = var.region
  repository_id          = "functions"
  format                 = "DOCKER"
  description            = "Container images for cloud functions"
  cleanup_policy_dry_run = false

  cleanup_policies {
    id     = "delete-all-candidates"
    action = "DELETE"
    condition {
      older_than = "0s"
    }
  }

  cleanup_policies {
    id     = "keep-recent"
    action = "KEEP"
    most_recent_versions {
      keep_count = 3
    }
  }

  depends_on = [google_project_service.apis]
}

resource "google_service_account" "ci" {
  account_id   = "ci-deployer"
  display_name = "CI deployer"
  depends_on   = [google_project_service.apis]
}

resource "google_artifact_registry_repository_iam_member" "ci_writer" {
  project    = var.project_id
  location   = google_artifact_registry_repository.functions.location
  repository = google_artifact_registry_repository.functions.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:${google_service_account.ci.account_id}@${var.project_id}.iam.gserviceaccount.com"
}

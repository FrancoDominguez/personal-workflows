resource "google_artifact_registry_repository" "functions" {
  project       = var.project_id
  location      = "us-central1"
  repository_id = "functions"
  format        = "DOCKER"
  description   = "Container images for cloud functions"
  depends_on = [ google_project_service.apis ]
}

resource "google_service_account" "ci" {
  account_id   = "ci-deployer"
  display_name = "CI deployer"
  depends_on = [ google_project_service.apis ]
}

resource "google_artifact_registry_repository_iam_member" "ci_writer" {
  project = var.project_id
  location = google_artifact_registry_repository.functions.location
  repository = google_artifact_registry_repository.functions.name
  role = "roles/artifactregistry.writer"
  member = "serviceAccount:${google_service_account.ci.account_id}@${var.project_id}.iam.gserviceaccount.com"
}

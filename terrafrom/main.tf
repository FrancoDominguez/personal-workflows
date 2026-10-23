provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_project_service" "artifactregistry" {
  project = var.project_id
  service = "artifactregistry.googleapis.com"
}

resource "google_artifact_registry_repository_iam_member" "ci_writer" {
  project    = var.project_id
  location   = google_artifact_registry_repository.functions.location
  repository = google_artifact_registry_repository.functions.name
  role       = "roles/artifactregistry.writer"
  member     = "serviceAccount:ci@${var.project_id}.iam.gserviceaccount.com"
}

resource "google_artifact_registry_repository" "functions" {
  project       = var.project_id
  location      = "us-central1"
  repository_id = "functions"
  format        = "DOCKER"
  description   = "Container images for cloud functions"
}
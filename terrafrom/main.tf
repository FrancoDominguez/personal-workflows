provider "google" {
  project = var.project_id
  region  = var.region
}

resource "google_artifact_registry_repository" "functions" {
  project       = var.project_id
  location      = "us-central1"
  repository_id = "functions"
  format        = "DOCKER"
  description   = "Container images for cloud functions"
}
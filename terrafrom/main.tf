terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
    }
    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

provider "github" {
  owner = var.github_username
}

resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    "artifactregistry.googleapis.com",
    "pubsub.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "iam.googleapis.com"
  ])
  project            = var.project_id
  service            = each.value
  disable_on_destroy = true
}

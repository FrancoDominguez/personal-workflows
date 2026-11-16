terraform {
  # Shared state so every machine plans against the same reality. The bucket
  # is created out-of-band (see docs/gcs-backend.md): it can't be managed by
  # the state it stores. Backend blocks can't use variables, hence hardcoded.
  backend "gcs" {
    bucket = "gmail-scanner-512523-tfstate"
    prefix = "terraform/state"
  }

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
  project               = var.project_id
  region                = var.region
  user_project_override = true
  billing_project       = var.project_id
}

provider "github" {
  owner = var.github_username
}

resource "google_project_service" "apis" {
  for_each = toset([
    "run.googleapis.com",
    # "compute.googleapis.com", # uncomment when migrating teams-presence (see teams-presence.tf.disabled)
    "artifactregistry.googleapis.com",
    "pubsub.googleapis.com",
    "iamcredentials.googleapis.com",
    "sts.googleapis.com",
    "iam.googleapis.com",
    "secretmanager.googleapis.com",
    "gmail.googleapis.com",
    "firestore.googleapis.com",
    "billingbudgets.googleapis.com",
    "cloudscheduler.googleapis.com",
  ])
  project            = var.project_id
  service            = each.value
  disable_on_destroy = true
}

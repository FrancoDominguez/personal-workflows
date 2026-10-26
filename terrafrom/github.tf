resource "google_iam_workload_identity_pool" "github" {
  workload_identity_pool_id = "github-pool"
  depends_on                = [google_project_service.apis]
}

resource "google_iam_workload_identity_pool_provider" "github" {
  workload_identity_pool_id          = google_iam_workload_identity_pool.github.workload_identity_pool_id
  workload_identity_pool_provider_id = "github-provider"
  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
  }
  attribute_condition = "assertion.repository == \"${var.github_username}/${var.github_repo_name}\""
  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }
}

resource "google_service_account_iam_member" "github_wif" {
  service_account_id = google_service_account.ci.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/${google_iam_workload_identity_pool.github.name}/attribute.repository/${var.github_username}/${var.github_repo_name}"
}

resource "github_actions_variable" "wif_provider" {
  repository    = var.github_repo_name
  variable_name = "GCP_WIF_PROVIDER"
  value         = google_iam_workload_identity_pool_provider.github.name
}

resource "github_actions_variable" "ci_sa_email" {
  repository    = var.github_repo_name
  variable_name = "GCP_CI_SA_EMAIL"
  value         = google_service_account.ci.email
}

resource "github_actions_variable" "gcp_region" {
  repository    = var.github_repo_name
  variable_name = "GCP_REGION"
  value         = var.region
}

resource "github_actions_variable" "gcp_project" {
  repository    = var.github_repo_name
  variable_name = "GCP_PROJECT_ID"
  value         = var.project_id
}
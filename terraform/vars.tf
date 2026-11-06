variable "project_id" {
  type        = string
  description = "GCP project to deploy into"
}

variable "region" {
  type        = string
  description = "Default GCP region"
  default     = "us-central1"
}

variable "github_username" {
  type        = string
  description = "Username of Github account running the CI"
}

variable "github_repo_name" {
  type        = string
  description = "Name of the github repo"
}

variable "test_user_account_email" {
  type        = string
  description = "Email of test user"
}

variable "gcp_client_id" {
  type        = string
  description = "OAuth client id from GCP consent screen (public, not a secret)"
}

variable "billing_account_id" {
  type        = string
  description = "Account ID for billing account"
}

variable "slack_allowed_user_id" {
  type        = string
  description = "Slack member id allowed to invoke the /run slash command"
}

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

variable "gcp_client_secret" {
  type        = string
  description = "Client secret from GCP consent screen"
}

variable "gcp_client_id" {
  type        = string
  description = "Client id from GCP consent screen"
}
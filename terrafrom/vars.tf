variable "project_id" {
    type = string
    description = "GCP project to deploy into"
}

variable "region" {
  type        = string
  description = "Default GCP region"
  default     = "us-central1"
}
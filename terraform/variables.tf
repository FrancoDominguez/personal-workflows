variable "project_id" {
  type = string
}

variable "region" {
  type    = string
  default = "northamerica-northeast1" # Montreal
}

variable "gmail_client_id" {
  type = string
}

variable "gmail_client_secret" {
  type      = string
  sensitive = true
}

variable "gmail_refresh_token" {
  type      = string
  sensitive = true
}

# Holds the zipped function source and the last-seen Gmail history id.
resource "google_storage_bucket" "bucket" {
  name                        = "${var.project_id}-gmail-webhook"
  location                    = var.region
  uniform_bucket_level_access = true
  force_destroy               = true
}

data "archive_file" "source" {
  type        = "zip"
  source_dir  = "${path.module}/../app"
  output_path = "${path.module}/.build/app.zip"
}

resource "google_storage_bucket_object" "source" {
  name   = "source-${data.archive_file.source.output_md5}.zip"
  bucket = google_storage_bucket.bucket.name
  source = data.archive_file.source.output_path
}

resource "google_service_account" "fn" {
  account_id = "gmail-webhook"
}

resource "google_storage_bucket_iam_member" "fn_state" {
  bucket = google_storage_bucket.bucket.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.fn.email}"
}

# The Pub/Sub trigger delivers events by invoking the function's Run service as this SA.
resource "google_project_iam_member" "fn_invoker" {
  project = var.project_id
  role    = "roles/run.invoker"
  member  = "serviceAccount:${google_service_account.fn.email}"
}

resource "google_project_iam_member" "fn_event_receiver" {
  project = var.project_id
  role    = "roles/eventarc.eventReceiver"
  member  = "serviceAccount:${google_service_account.fn.email}"
}

resource "google_cloudfunctions2_function" "fn" {
  name     = "gmail-webhook"
  location = var.region

  build_config {
    runtime     = "python312"
    entry_point = "handle"
    source {
      storage_source {
        bucket = google_storage_bucket.bucket.name
        object = google_storage_bucket_object.source.name
      }
    }
  }

  service_config {
    available_memory      = "256M"
    max_instance_count    = 1
    service_account_email = google_service_account.fn.email
    environment_variables = {
      TOPIC               = google_pubsub_topic.gmail.id
      STATE_BUCKET        = google_storage_bucket.bucket.name
      GMAIL_CLIENT_ID     = var.gmail_client_id
      GMAIL_CLIENT_SECRET = var.gmail_client_secret
      GMAIL_REFRESH_TOKEN = var.gmail_refresh_token
    }
  }

  event_trigger {
    trigger_region        = var.region
    event_type            = "google.cloud.pubsub.topic.v1.messagePublished"
    pubsub_topic          = google_pubsub_topic.gmail.id
    retry_policy          = "RETRY_POLICY_DO_NOT_RETRY"
    service_account_email = google_service_account.fn.email
  }

  depends_on = [google_project_service.apis]
}

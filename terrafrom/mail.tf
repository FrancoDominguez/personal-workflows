resource "google_pubsub_topic" "gmail" {
  name       = "gmail-notifications"
  depends_on = [google_project_service.apis]
}

# Gmail publishes via this Google-owned system account — same address for everyone
resource "google_pubsub_topic_iam_member" "gmail_publisher" {
  topic  = google_pubsub_topic.gmail.name
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:gmail-api-push@system.gserviceaccount.com"
}

# Identity that Pub/Sub uses when pushing to your Cloud Run service
resource "google_service_account" "pubsub_pusher" {
  account_id   = "pubsub-pusher"
  display_name = "Pub/Sub push to Cloud Run"
  depends_on   = [google_project_service.apis]
}

resource "google_cloud_run_v2_service_iam_member" "pubsub_invoker" {
  project  = var.project_id
  location = google_cloud_run_v2_service.function.location
  name     = google_cloud_run_v2_service.function.name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${google_service_account.pubsub_pusher.email}"
}

resource "google_pubsub_subscription" "gmail_push" {
  name  = "gmail-push-to-function"
  topic = google_pubsub_topic.gmail.id

  push_config {
    push_endpoint = "${google_cloud_run_v2_service.function.uri}/webhook" # whatever route your Go app handles

    oidc_token {
      service_account_email = google_service_account.pubsub_pusher.email
    }
  }

  ack_deadline_seconds = 30
}

# database to store history id
resource "google_firestore_database" "database" {
  project     = var.project_id
  name        = "(default)"
  location_id = "nam5"
  type        = "FIRESTORE_NATIVE"
  depends_on  = [google_project_service.apis]
}

resource "google_project_iam_member" "function_uses_firestore" {
  project = var.project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.function_runtime.email}"
}
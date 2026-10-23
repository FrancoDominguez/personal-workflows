# Gmail publishes a tiny {emailAddress, historyId} message here on every inbox change.
resource "google_pubsub_topic" "gmail" {
  name       = "gmail-notify"
  depends_on = [google_project_service.apis]
}

# Gmail's push service account must be allowed to publish to the topic.
resource "google_pubsub_topic_iam_member" "gmail_publisher" {
  topic  = google_pubsub_topic.gmail.id
  role   = "roles/pubsub.publisher"
  member = "serviceAccount:gmail-api-push@system.gserviceaccount.com"
}

# A Gmail watch expires after 7 days; this publishes "renew" daily and the
# function re-arms it. Same topic, so one function handles both message kinds.
resource "google_cloud_scheduler_job" "renew_watch" {
  name      = "gmail-watch-renew"
  schedule  = "0 9 * * *"
  time_zone = "America/Montreal"

  pubsub_target {
    topic_name = google_pubsub_topic.gmail.id
    data       = base64encode("renew")
  }

  depends_on = [google_project_service.apis]
}

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

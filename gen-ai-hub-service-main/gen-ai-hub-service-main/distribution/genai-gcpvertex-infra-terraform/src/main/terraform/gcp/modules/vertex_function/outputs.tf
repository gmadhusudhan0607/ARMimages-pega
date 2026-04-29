/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

output "service_account_email" {
  value = google_service_account.vertex_function_service_account.email
  description = "Email of the Vertex AI service account"
}

output "function_id" {
  value = google_cloudfunctions2_function.vertex_function.id
  description = "ID of the Vertex AI Cloud Function"
}

output "function_url" {
  value = google_cloudfunctions2_function.vertex_function.url
  description = "Public URL of the Vertex AI Cloud Function"
}

output "function_uri" {
  value = google_cloudfunctions2_function.vertex_function.service_config[0].uri
  description = "Backend URI of the Vertex AI Cloud Function"
}

output "function_name" {
  value = google_cloudfunctions2_function.vertex_function.name
  description = "Name of the Vertex AI Cloud Function"
}

output "function_location" {
  value = google_cloudfunctions2_function.vertex_function.location
  description = "Location of the Vertex AI Cloud Function"
}

output "bucket_name" {
  value = google_storage_bucket.bucket.name
  description = "Name of the Cloud Storage bucket"
}

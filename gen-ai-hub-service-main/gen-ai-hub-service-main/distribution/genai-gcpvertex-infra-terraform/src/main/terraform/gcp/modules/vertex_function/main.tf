/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

resource "google_service_account" "vertex_function_service_account" {
  account_id   = var.service_account_name
  display_name = "Vertex AI Invoker Service Account"
  project      = var.gcp_project_id
}

resource "google_project_iam_member" "vertex_invoker_permission" {
  project = var.gcp_project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${google_service_account.vertex_function_service_account.email}"
}

# Create a Cloud Storage bucket
resource "google_storage_bucket" "bucket" {
  name                        = var.bucket_name
  location                    = var.region
  project                     = var.gcp_project_id
  uniform_bucket_level_access = true
  labels                      = var.provisioning_tags
}

# Archive the function source code as a zip file for Cloud Functions v2
data "archive_file" "function_zip" {
  type        = "zip"
  output_path = "${path.module}/function-source.zip"
  source {
    content  = templatefile("${path.module}/templates/main.py.tpl", {})
    filename = "main.py"
  }
  source {
    content  = templatefile("${path.module}/templates/requirements.txt.tpl", {})
    filename = "requirements.txt"
  }
}

# Upload the zip archive to the bucket
resource "google_storage_bucket_object" "function_zip" {
  name   = "function-source.zip"
  bucket = google_storage_bucket.bucket.name
  source = data.archive_file.function_zip.output_path
}

resource "google_cloudfunctions2_function" "vertex_function" {
  project  = var.gcp_project_id
  name     = var.cloud_run_name
  location = var.region
  build_config {
    runtime     = "python312"
    entry_point = "handle_request"
    source {
      storage_source {
        bucket     = google_storage_bucket.bucket.name
        object     = google_storage_bucket_object.function_zip.name
        generation = google_storage_bucket_object.function_zip.generation
      }
    }
  }
  service_config {
    max_instance_request_concurrency = 50
    max_instance_count               = 50
    available_memory                 = "1G"
    available_cpu                    = "1"
    timeout_seconds                  = var.function_timeout_seconds
    ingress_settings                 = "ALLOW_ALL"
    environment_variables = {
      GCP_PROJECT_ID          = var.gcp_project_id
      GCP_REGION              = var.region
      VERTEX_INFERENCE_REGION = var.inference_region != null ? var.inference_region : ""
      FUNCTION_TIMEOUT        = var.function_timeout_seconds
    }
    service_account_email = google_service_account.vertex_function_service_account.email
  }
  labels = merge(var.provisioning_tags, {
    source-hash = substr(data.archive_file.function_zip.output_sha256, 0, 63)
  })
}

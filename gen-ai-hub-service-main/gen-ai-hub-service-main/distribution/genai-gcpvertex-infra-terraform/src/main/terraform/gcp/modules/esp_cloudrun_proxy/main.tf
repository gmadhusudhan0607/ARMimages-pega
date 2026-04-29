/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

resource "google_cloud_run_service_iam_member" "apigw_invoker" {
  location = var.function_location
  service  = var.function_name
  role     = "roles/run.invoker"
  member   = "serviceAccount:${var.espv2_service_account_email}"
}

# Grant Cloud Functions invoker role to ESPv2 service account (required when using custom service account)
resource "google_cloudfunctions2_function_iam_member" "apigw_function_invoker" {
  project        = var.gcp_project_id
  location       = var.function_location
  cloud_function = var.function_name
  role           = "roles/cloudfunctions.invoker"
  member         = "serviceAccount:${var.espv2_service_account_email}"
}

# Grant project-level service management controller role to ESPv2 service account
resource "google_project_iam_member" "apigw_service_controller" {
  project = var.gcp_project_id
  role    = "roles/servicemanagement.serviceController"
  member  = "serviceAccount:${var.espv2_service_account_email}"
}

# Grant Cloud Trace agent role to ESPv2 service account (required for tracing/spans)
resource "google_project_iam_member" "apigw_trace_agent" {
  project = var.gcp_project_id
  role    = "roles/cloudtrace.agent"
  member  = "serviceAccount:${var.espv2_service_account_email}"
}

# Grant Service Control Reporter role to ESPv2 service account (required for service control API)
resource "google_project_iam_member" "apigw_service_control_reporter" {
  project = var.gcp_project_id
  role    = "roles/servicemanagement.reporter"
  member  = "serviceAccount:${var.espv2_service_account_email}"
}

# Deploy ESPv2 as a Cloud Run service (replaces placeholder service)
resource "google_cloud_run_service" "espv2_proxy" {
  name     = var.endpoints_service_name  # Same name as placeholder
  location = var.region
  project  = var.gcp_project_id

  template {
    spec {
      containers {
        image = "gcr.io/endpoints-release/endpoints-runtime-serverless:2"
        env {
          name  = "ENDPOINTS_SERVICE_NAME"
          value = var.endpoints_service_name_full
        }
        resources {
          limits = {
            cpu    = "1"
            memory = "512Mi"
          }
        }
      }

      service_account_name = var.espv2_service_account_email
    }
  }

  autogenerate_revision_name = true

  traffic {
    percent         = 100
    latest_revision = true
  }

  lifecycle {
    # Ignore operation-id changes during import and updates
    ignore_changes = [
      metadata,
      template[0].metadata,
      template[0].spec[0].container_concurrency
    ]
  }
}

# Make the Cloud Run service public
resource "google_cloud_run_service_iam_member" "public_access" {
  location = google_cloud_run_service.espv2_proxy.location
  service  = google_cloud_run_service.espv2_proxy.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

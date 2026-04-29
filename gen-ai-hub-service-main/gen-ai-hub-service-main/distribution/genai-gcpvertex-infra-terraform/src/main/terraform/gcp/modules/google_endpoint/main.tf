/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

resource "google_service_account" "espv2_service_account" {
  account_id   = var.apigw_sa_name
  project = var.gcp_project_id
  display_name = "GenAI API Gateway Service Account"
}

resource "google_endpoints_service" "vertex_endpoints" {
  service_name   = var.google_endpoint_service_name
  project        = var.gcp_project_id
  openapi_config = var.spec_file
}

# Enable the custom Endpoints service in the project
resource "google_project_service" "custom_endpoints_service" {
  project = var.gcp_project_id
  service = google_endpoints_service.vertex_endpoints.service_name

  disable_dependent_services = true

  depends_on = [
    google_endpoints_service.vertex_endpoints
  ]
}

# Create Endpoints Config
resource "google_endpoints_service_iam_member" "endpoints_service_account" {
  service_name = google_endpoints_service.vertex_endpoints.service_name
  role         = "roles/servicemanagement.serviceController"
  member       = "serviceAccount:${google_service_account.espv2_service_account.email}"
}

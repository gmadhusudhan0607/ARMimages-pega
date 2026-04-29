/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

output "service_name" {
  value = google_endpoints_service.vertex_endpoints.service_name
  description = "Name of the Google Endpoints service"
}

output "service_account_email" {
  value = google_service_account.espv2_service_account.email
  description = "Email of the ESPv2 service account"
}

output "endpoints_service" {
  value = google_endpoints_service.vertex_endpoints
  description = "The Google Endpoints service resource"
}

output "project_service" {
  value = google_project_service.custom_endpoints_service
  description = "The project service resource"
}

output "endpoints_iam_member" {
  value = google_endpoints_service_iam_member.endpoints_service_account
  description = "The endpoints service IAM member resource"
}

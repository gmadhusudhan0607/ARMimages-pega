/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

variable "gcp_project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "google_endpoint_service_name" {
  description = "Name for the Google Endpoints service"
  type        = string
}

variable "spec_file" {
  description = "OpenAPI specification file content"
  type        = string
}

variable "function_uri" {
  description = "URI of the Cloud Function"
  type        = string
}

variable "apigw_sa_name" {
  description = "Name for the API Gateway service account"
  type        = string
}

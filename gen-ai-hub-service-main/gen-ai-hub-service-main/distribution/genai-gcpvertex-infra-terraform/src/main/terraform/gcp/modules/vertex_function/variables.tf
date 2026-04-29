/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

variable "gcp_project_id" {
  description = "The GCP project ID"
  type        = string
}

variable "region" {
  description = "The GCP region"
  type        = string
}

variable "service_account_name" {
  description = "Name for the Vertex AI service account"
  type        = string
}

variable "bucket_name" {
  description = "Name for the Cloud Storage bucket"
  type        = string
}

variable "cloud_run_name" {
  description = "Name for the Cloud Functions/Cloud Run service"
  type        = string
}

variable "provisioning_tags" {
  type        = map(any)
  description = "Tags to apply to resources"
  default     = {}
}

variable "inference_region" {
  description = "The specific region to use for Vertex AI inference. If provided, overrides the default region detection."
  type        = string
  default     = null
}

variable "function_timeout_seconds" {
  description = "Timeout in seconds for the Cloud Function execution and client calls"
  type        = number
  default     = 600
}

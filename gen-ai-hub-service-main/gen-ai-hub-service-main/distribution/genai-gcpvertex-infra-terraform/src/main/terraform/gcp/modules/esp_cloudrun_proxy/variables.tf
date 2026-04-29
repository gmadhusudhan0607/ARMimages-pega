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

variable "espv2_service_account_email" {
  description = "Email of the ESPv2 service account"
  type        = string
}

variable "endpoints_service_name" {
  description = "Name for the endpoints service"
  type        = string
}

variable "function_location" {
  description = "Location of the Cloud Function"
  type        = string
}

variable "function_name" {
  description = "Name of the Cloud Function"
  type        = string
}

variable "endpoints_service_name_full" {
  description = "Full name of the Google Endpoints service"
  type        = string
}

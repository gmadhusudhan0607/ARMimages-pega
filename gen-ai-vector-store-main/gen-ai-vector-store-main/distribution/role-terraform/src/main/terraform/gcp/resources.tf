/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################

resource "google_service_account" "vector_store_service_account" {
  project = var.AccountID
  account_id = "vector-store-${substr(var.ResourceGUID,0,7)}-sa"
  display_name = "GenAI Vector-Store SA for service: ${var.ResourceGUID}"
}


resource "google_service_account_iam_binding" "vector_store_sa_binding" {
  service_account_id = google_service_account.vector_store_service_account.name
  role               = "roles/iam.workloadIdentityUser"
  members = [
    "serviceAccount:${var.AccountID}.svc.id.goog[${var.Namespace}/genai-vector-store]",
    "serviceAccount:${var.AccountID}.svc.id.goog[${var.Namespace}/genai-vector-store-sa]",
    "serviceAccount:${var.AccountID}.svc.id.goog[${var.Namespace}/genai-vector-store-ops-sa]",
    "serviceAccount:${var.AccountID}.svc.id.goog[${var.Namespace}/genai-vector-store-background-sa]",
  ]
}

resource "google_project_iam_member" "db" {
  project = var.AccountID
  role = "roles/cloudsql.client"
  member = "serviceAccount:${google_service_account.vector_store_service_account.email}"
}

resource "google_secret_manager_secret_iam_member" "db_secret" {
  for_each = local.DBSecretsARNs
  project = var.AccountID
  secret_id = each.value
  role = "roles/secretmanager.secretAccessor"
  member = "serviceAccount:${google_service_account.vector_store_service_account.email}"
}

resource "google_secret_manager_secret_iam_member" "sax_client_secret" {
  project   = var.AccountID
  secret_id = var.SaxClientSecret
  role      = "roles/secretmanager.secretAccessor"
  member    = "serviceAccount:${google_service_account.vector_store_service_account.email}"
}

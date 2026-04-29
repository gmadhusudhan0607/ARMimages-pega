/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

# Import the placeholder Cloud Run service using Terraform 1.6+ import block
import {
  to = module.esp_cloudrun_proxy.google_cloud_run_service.espv2_proxy
  id = "locations/${var.Region}/namespaces/${var.GcpProjectId}/services/${local.endpoints_service_name}"
}

module "vertex_function" {
  source = "./modules/vertex_function"

  gcp_project_id           = var.GcpProjectId
  region                   = var.Region
  service_account_name     = local.service_account_name
  bucket_name              = local.bucket_name
  cloud_run_name           = local.cloud_run_name
  provisioning_tags        = local.provisioning_tags
  inference_region         = var.ModelInferenceRegionOverride
  function_timeout_seconds = local.function_timeout_seconds
}

module "google_endpoint" {
  source = "./modules/google_endpoint"

  gcp_project_id               = var.GcpProjectId
  google_endpoint_service_name = local.endpoints_host
  spec_file                    = local.spec_file
  function_uri                 = module.vertex_function.function_uri
  apigw_sa_name                = local.apigw_sa_name

  depends_on = [
    module.vertex_function,
  ]
}

module "esp_cloudrun_proxy" {
  source = "./modules/esp_cloudrun_proxy"

  gcp_project_id              = var.GcpProjectId
  region                      = var.Region
  espv2_service_account_email = module.google_endpoint.service_account_email
  endpoints_service_name      = local.endpoints_service_name
  function_location           = module.vertex_function.function_location
  function_name               = module.vertex_function.function_name
  endpoints_service_name_full = module.google_endpoint.service_name

  depends_on = [
    module.vertex_function,
    module.google_endpoint,
  ]
}

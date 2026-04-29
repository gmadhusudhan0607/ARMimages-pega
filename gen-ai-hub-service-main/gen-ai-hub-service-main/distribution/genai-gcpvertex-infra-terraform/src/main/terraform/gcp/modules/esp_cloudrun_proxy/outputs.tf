/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */


output "cloud_run_service_url" {
  value = google_cloud_run_service.espv2_proxy.status[0].url
  description = "URL of the ESPv2 Cloud Run service"
}

output "cloud_run_service" {
  value = google_cloud_run_service.espv2_proxy
  description = "The ESPv2 Cloud Run service resource"
}

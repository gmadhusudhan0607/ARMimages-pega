/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */


output "ServiceAccountApiGatewayEmail" {
  value = module.google_endpoint.service_account_email
}

output "VertexCloudRunFunctionId" {
  value = module.vertex_function.function_id
}

output "VertexCloudRunFunctionPublicUrl" {
  value = module.vertex_function.function_url
}

output "VertexCloudRunFunctionBackendUri" {
  value = module.vertex_function.function_uri
}

output "VetexCloudRunFunctionGcfUri" {
  value = module.vertex_function.function_uri
}

output "CloudStorageBucketCloudRunSource" {
  value = module.vertex_function.bucket_name
}

output "ServiceAccountCloudRunInvokerEmail" {
  value = module.vertex_function.service_account_email
}

output "EndpointsGatewayUrl" {
  value = module.esp_cloudrun_proxy.cloud_run_service_url
}

output "EndpointsServiceName" {
  value = module.google_endpoint.service_name
}

output "APIConfigSpecFile" {
  value = local.spec_file
}

/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */


output "GcpVertexAIApiGatewayHost" {
  value = local.endpoints_host
}

output "GcpVertexAIServiceName" {
  value = local.endpoints_service_name
}

output "ResourcesSuffixId" {
  value = local.suffix
}

output "Owner" {
  value = var.Owner
}

output "GcpProjectId" {
  value = var.GcpProjectId
}

output "Region" {
  value = var.Region
}
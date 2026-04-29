/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
####################################################################################################


# Storage Service
########################################################################################
output "SaxJWKSEndpoint" {
  value = data.pegasec_backing_service.GenAIGatewayService.jwks_endpoint
}

output "SaxIssuer" {
  value = data.pegasec_backing_service.GenAIGatewayService.issuer
}

output "SaxAudience" {
  value = data.pegasec_backing_service.GenAIGatewayService.audience
}

output "SaxScopesString" {
  value = data.pegasec_backing_service.GenAIGatewayService.scopes_string
}
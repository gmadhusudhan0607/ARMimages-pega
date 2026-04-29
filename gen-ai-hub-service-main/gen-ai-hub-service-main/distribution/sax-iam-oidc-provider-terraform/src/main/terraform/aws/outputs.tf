/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

output "OidcProviderArn" {
  value = aws_iam_openid_connect_provider.backing_services_provider.arn
}

output "OidcProviderUrl" {
  value = aws_iam_openid_connect_provider.backing_services_provider.url
}

output "SaxCell" {
  value = var.SaxCell
}

output "SaxStage" {
  value = local.sax_stage
}

output "GetBedrockModelMappingOidcRole" {
  value = module.iam_iam-assumable-role-with-oidc.iam_role_name
}

output "GetBedrockModelMappingOidcRoleArn" {
  value = module.iam_iam-assumable-role-with-oidc.iam_role_arn
}
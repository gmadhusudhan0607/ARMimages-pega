/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

locals {
  roleName = "genai-oidcrole-${var.SaxCell}-${random_string.id_suffix.result}"

  // Extract region prefix from AWS region name by removing the last two parts (location and number)
  // Examples: "us-east-1" -> "us", "ap-southeast-1" -> "ap", "us-gov-west-1" -> "us-gov"
  region_parts       = split("-", local.target_model_region)
  region_parts_count = length(local.region_parts)
  aws_region_prefix  = join("-", slice(local.region_parts, 0, local.region_parts_count - 2))

  stage_name       = var.StageName == "production-usgov" ? "prod-usgov" : var.StageName
  targetAwsAccount = "arn:${data.aws_partition.cp_partition.partition}:iam::${var.AccountID}:role/provisioning-service-${local.stage_name}-PEAccessRole"

  is_usgov_region            = strcontains(var.Region, "gov")
  fallback_region            = local.is_usgov_region && var.UseRegionalInferenceProfile ? "us-gov-east-1" : var.Region
  target_model_region        = trimspace(var.InferenceRegion) != "" ? trimspace(var.InferenceRegion) : local.fallback_region
  policyName                 = "genai-oidcpolicy-${random_string.id_suffix.result}"
  genai_secret_name          = "genai_infra/${var.StageName}/${var.SaxCell}/${var.ModelMapping}/${random_string.id_suffix.result}"
  bedrock_endpoint           = "https://${local.aws_service_name}.${local.target_model_region}.amazonaws.com"
  aws_service_name           = local.is_usgov_region ? "bedrock-runtime-fips" : "bedrock-runtime"
  model_path                 = "/model/${local.model_inference_profile_id}/${lower(var.TargetApi)}"
  model_inference_profile_id = var.UseRegionalInferenceProfile ? "${var.InferenceProfilePrefix}.${var.ModelID}" : "${var.ModelID}"
  # ARNs are constructed deterministically (not from data sources) so that
  # Inactive=true deployments can plan/apply/destroy without querying AWS
  # for model existence. Data sources serve only as validation gates.
  foundation_model_arn  = "arn:${data.aws_partition.cp_partition.partition}:bedrock:${local.target_model_region}::foundation-model/${var.ModelID}"
  inference_profile_arn = "arn:${data.aws_partition.cp_partition.partition}:bedrock:${local.target_model_region}:${var.AccountID}:inference-profile/${local.model_inference_profile_id}"

  policy_resources_arns = var.UseRegionalInferenceProfile ? concat(
    [local.inference_profile_arn],
    [local.reginal_inference_models_arn]
  ) : [local.foundation_model_arn]

  reginal_inference_models_arn = var.UseRegionalInferenceProfile ? "arn:${data.aws_partition.cp_partition.partition}:bedrock:${local.aws_region_prefix}-*::foundation-model/${var.ModelID}" : ""

  genai_secret_json = jsonencode({
    "ModelMapping"                = var.ModelMapping
    "ModelId"                     = local.model_inference_profile_id
    "OIDCIAMRoleArn"              = module.iam_iam-assumable-role-with-oidc.iam_role_arn
    "Region"                      = local.target_model_region
    "Endpoint"                    = local.bedrock_endpoint
    "Path"                        = local.model_path
    "TargetApi"                   = var.TargetApi
    "Inactive"                    = var.Inactive
    "UseRegionalInferenceProfile" = var.UseRegionalInferenceProfile
  })

  tags = merge(
    var.provisioningTags,
    tomap({
      "Owner" = var.Owner
    })
  )
}

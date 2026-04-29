/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

# Validation gate: confirms model exists in AWS when active.
# Output NOT used by other resources — ARNs are constructed deterministically.
data "aws_bedrock_foundation_model" "model" {
  provider = aws.genai_inference_region_provider
  model_id = var.ModelID
  count    = !var.Inactive && !var.UseRegionalInferenceProfile ? 1 : 0
}

data "aws_partition" "cp_partition" {
  provider = aws.cp_account
}

resource "random_string" "id_suffix" {
  length  = 5
  special = false
  upper   = false
}

// To do next:
//   when set to use an inference profile, allow call to model only through Inference Profile.
//  this documentation is the reference to set it:
//    https://docs.aws.amazon.com/bedrock/latest/userguide/inference-profiles-prereq.html

# Validation gate: confirms inference profile exists in AWS when active.
# Output NOT used by other resources — ARNs are constructed deterministically.
data "aws_bedrock_inference_profile" "inference_profile" {
  provider             = aws.genai_inference_region_provider
  inference_profile_id = local.model_inference_profile_id
  count                = !var.Inactive && var.UseRegionalInferenceProfile ? 1 : 0
}

data "aws_iam_policy_document" "policy" {
  provider = aws.genai_account
  statement {
    effect = "Allow"
    actions = [
      "bedrock:InvokeModel",
      "bedrock:InvokeModelWithResponseStream"
    ]
    resources = local.policy_resources_arns
  }
}

module "iam_policy" {
  providers = {
    aws = aws.genai_account
  }

  source  = "terraform-aws-modules/iam/aws//modules/iam-policy"
  version = "5.44.0"
  name    = local.policyName
  path    = "/"

  policy = data.aws_iam_policy_document.policy.json
}

module "iam_iam-assumable-role-with-oidc" {
  providers = {
    aws = aws.genai_account
  }
  source       = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version      = "5.44.0"
  create_role  = true
  role_name    = local.roleName
  provider_url = var.OidcProviderUrl
  role_policy_arns = [
    module.iam_policy.arn
  ]
  tags = local.tags
  depends_on = [
    module.iam_policy
  ]
}

# secret gets created in the AWS LLM Account
resource "aws_secretsmanager_secret" "bedrock_model_secret" {
  provider                = aws.genai_account
  name                    = local.genai_secret_name
  description             = "hold mapping for a genai model"
  recovery_window_in_days = 0
  tags                    = local.tags
}

resource "aws_secretsmanager_secret_version" "bedrock_model_secret_version" {
  provider      = aws.genai_account
  secret_id     = aws_secretsmanager_secret.bedrock_model_secret.id
  secret_string = local.genai_secret_json
}
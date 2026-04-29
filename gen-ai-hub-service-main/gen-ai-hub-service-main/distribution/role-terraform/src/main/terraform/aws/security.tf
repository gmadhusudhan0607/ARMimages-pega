/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
 
locals {
  KMSKeyArn = length(regexall("^arn:aws:kms:", var.KMSKeyForSecrets)) > 0 ? var.KMSKeyForSecrets : "arn:aws:kms:${var.Region}:${var.AccountID}:key/${var.KMSKeyForSecrets}"
}

module "service_account_role" {
  source       = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version      = "5.8.0"
  create_role  = true
  role_name    = "genai-hub-service-${var.ResourceGUID}-sa-role"
  provider_url = replace(var.ClusterOIDCIssuerURL, "https://", "")
  role_policy_arns = [
    aws_iam_policy.sax_client_secret_access.arn
  ]
  tags = var.provisioningTags
}

data "aws_iam_policy_document" "sax_client_secret_access" {
  statement {
    actions = [
      "secretsmanager:GetSecretValue"
    ]
    effect    = "Allow"
    resources = [var.SaxClientSecret]
    sid       = "SecretsManager"
  }

  statement {
    actions = [
      "kms:Decrypt"
    ]
    effect    = "Allow"
    resources = [local.KMSKeyArn]
    sid       = "KMS"
  }
}

resource "aws_iam_policy" "sax_client_secret_access" {
  name   = "genai-hub-service-${var.ResourceGUID}-sax-client-secret-access-policy"
  policy = data.aws_iam_policy_document.sax_client_secret_access.json
  path   = "/"
}

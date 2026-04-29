/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
 
locals {
  # Check if the provided key is already a full ARN
  is_full_kms_arn = length(regexall("^arn:aws:kms:", var.KMSKeyForSecrets)) > 0

  # Check if the privided ActiveKMSKeyForSecrets is a full ARN
  is_full_active_kms_arn = length(regexall("^arn:aws:kms:", var.ActiveKMSKeyForSecrets)) > 0

  # Determine if we're in a GovCloud region
  is_gov_cloud = strcontains(var.Region, "us-gov")

  # Set the appropriate ARN prefix based on cloud type
  gov_cloud_prefix = "arn:aws-us-gov:kms:${var.Region}:"
  commercial_prefix = "arn:aws:kms:${var.Region}:"

  # Choose the correct prefix
  kms_arn_prefix = local.is_gov_cloud ? local.gov_cloud_prefix : local.commercial_prefix

  # Build the complete ARN if needed
  constructed_kms_arn = "${local.kms_arn_prefix}${var.AccountID}:key/${var.KMSKeyForSecrets}"

  # Build the complete Active ARN if needed
  constructed_active_kms_arn = "${local.kms_arn_prefix}${var.AccountID}:key/${var.KMSKeyForSecrets}"

  # Final KMS ARN value
  KMSKeyArn = local.is_full_kms_arn ? var.KMSKeyForSecrets : local.constructed_kms_arn

  # Final Active KMS ARN value
  ActiveKMSkeyArn = local.is_full_active_kms_arn ? var.ActiveKMSKeyForSecrets : local.constructed_active_kms_arn

  //Using toset to remove duplicates and guarantee that the list of ARN is unique (in case both values point to the same secret)
  DBSecretsARNs = toset([var.DatabaseSecret, var.ActiveDatabaseSecret])
  //Using toset to remove duplicates and guarantee that the list of ARN is unique (in case both values point to the same secret)
  KMSKeySecretsARNs = toset([local.KMSKeyArn, local.ActiveKMSkeyArn])
}

module "service_account_role" {
  source       = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version      = "5.52.1"
  create_role  = true
  role_name    = "genai-vector-store-${var.ResourceGUID}-sa-role"
  provider_url = replace(var.ClusterOIDCIssuerURL, "https://", "")
  role_policy_arns = [
    aws_iam_policy.db_secret_access.arn,
    aws_iam_policy.sax_client_secret_access.arn,
  ]
  tags = var.provisioningTags
}

data "aws_iam_policy_document" "db_secret_access" {
  statement {
    actions = [
      "secretsmanager:GetSecretValue"
    ]
    effect    = "Allow"
    resources = local.DBSecretsARNs
    sid       = "SecretsManager"
  }

  statement {
    actions = [
      "kms:Decrypt"
    ]
    effect    = "Allow"
    resources = local.KMSKeySecretsARNs
    sid       = "KMS"
  }
}

resource "aws_iam_policy" "db_secret_access" {
  name   = "genai-vector-store-${var.ResourceGUID}-db-secret-access-policy"
  policy = data.aws_iam_policy_document.db_secret_access.json
  path   = "/"
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
  name   = "genai-vector-store-${var.ResourceGUID}-sax-client-secret-access-policy"
  policy = data.aws_iam_policy_document.sax_client_secret_access.json
  path   = "/"
}

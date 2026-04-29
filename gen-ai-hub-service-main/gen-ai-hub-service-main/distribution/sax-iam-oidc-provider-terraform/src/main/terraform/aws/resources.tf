/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

# used to cleanup a leftover resource
# import {
#   to = aws_iam_openid_connect_provider.backing_services_provider
#   id = "arn:aws:iam::045663071481:oidc-provider/stg-fcp-us-1.oktapreview.com/oauth2/ausg5ldmi6IpvdpXX1d6"
# }

# Fetch TLS certificate thumbprint
data "tls_certificate" "thumbprint_provider" {
  url = local.sax_issuer_url
}

data "aws_partition" "cp_partition" {
  provider = aws.cp_account
}

// Create an IAM OIDC Provider
resource "aws_iam_openid_connect_provider" "backing_services_provider" {
  provider = aws.genai_account
  url = local.sax_issuer_url

  client_id_list = [
    local.backingServicesAud
  ]
  thumbprint_list = [data.tls_certificate.thumbprint_provider.certificates[0].sha1_fingerprint]
  tags            = local.tags
}

module "iam_policy" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-policy"
  version = "5.44.0"
  providers = {
    aws = aws.genai_account
  }

  name    = "genai-get-bedrock-model-mapping-secrets-${var.StageName}-${var.SaxCell}"
  path    = "/"

  policy = <<EOF
  {
    "Version": "2012-10-17",
    "Statement": [
        {
        "Sid": "ListAndGetSecrets",
        "Action": [
            "secretsmanager:GetSecretValue",
            "secretsmanager:ListSecretVersionIds",
            "secretsmanager:ListSecrets"
        ],
        "Effect": "Allow",
        "Resource": "*"
        }
    ]
    }
  EOF
}

module "iam_iam-assumable-role-with-oidc" {
  source       = "terraform-aws-modules/iam/aws//modules/iam-assumable-role-with-oidc"
  version      = "5.44.0"
  providers = {
    aws = aws.genai_account
  }

  create_role  = true
  role_name    = "genai-oidcrole-get-secrets-${var.StageName}-${var.SaxCell}"
  provider_url = aws_iam_openid_connect_provider.backing_services_provider.url
  role_policy_arns = [
    module.iam_policy.arn
  ]
  tags = local.tags
  depends_on = [
    module.iam_policy
  ]
}
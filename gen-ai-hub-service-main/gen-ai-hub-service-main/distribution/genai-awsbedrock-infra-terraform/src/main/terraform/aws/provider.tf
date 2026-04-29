/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

provider "aws" {
  alias  = "genai_account"
  region = var.Region
  assume_role {
    role_arn = local.targetAwsAccount
  }
}

provider "aws" {
  alias  = "genai_inference_region_provider"
  region = local.target_model_region
  assume_role {
    role_arn = local.targetAwsAccount
  }
}

provider "aws" {
  alias = "cp_account"
}
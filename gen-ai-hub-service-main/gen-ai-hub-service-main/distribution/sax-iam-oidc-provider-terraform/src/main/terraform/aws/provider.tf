/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

provider "aws" {
  alias = "cp_account"
}

provider "aws" {
  alias = "genai_account"
  assume_role {
    role_arn = local.llmAccountRoleArn
  }
}
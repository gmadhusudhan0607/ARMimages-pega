/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
 
provider "aws" {
  region  = var.Region
  dynamic "assume_role" {
    for_each = var.RoleArn == "" ? [] : [1]
    content {
      role_arn = var.RoleArn
    }
  }
}

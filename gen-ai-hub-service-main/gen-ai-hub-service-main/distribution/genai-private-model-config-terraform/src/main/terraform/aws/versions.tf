/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.60.0"
    }
  }
  required_version = ">= 0.13"
}

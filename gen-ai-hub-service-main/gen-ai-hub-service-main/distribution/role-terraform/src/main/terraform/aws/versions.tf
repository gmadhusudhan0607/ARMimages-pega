/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */
 
terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
      version = "4.41.0"
    }
  }
  required_version = ">= 0.13"
}

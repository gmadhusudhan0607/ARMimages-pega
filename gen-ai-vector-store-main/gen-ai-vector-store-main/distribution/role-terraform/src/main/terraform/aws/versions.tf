/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
 
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.83.1" #for >= 5.83.1 but < 5.84.0
    }
  }
  required_version = ">= 1.0"
}

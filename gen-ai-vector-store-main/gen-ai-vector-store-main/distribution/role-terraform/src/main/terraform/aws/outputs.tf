/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
 
output "ServiceAccountRole" {
  value = module.service_account_role.iam_role_arn
}

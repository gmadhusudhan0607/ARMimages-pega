/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

output "PrivateModelConfigSecret" {
  value = aws_secretsmanager_secret.private_model_config_secret.id
}

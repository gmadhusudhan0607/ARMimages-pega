/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

output "BedrockModelAwsSecretName" {
  value = aws_secretsmanager_secret.bedrock_model_secret.name
}

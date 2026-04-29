/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */


output "GenAIDefaultModelSecretName" {
  value = aws_secretsmanager_secret.genai_defaults_secret.name
}
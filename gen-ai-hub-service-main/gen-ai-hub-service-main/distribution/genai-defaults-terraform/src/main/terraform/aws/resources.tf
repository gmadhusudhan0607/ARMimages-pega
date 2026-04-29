/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

# secret gets created in the AWS LLM Account
resource "aws_secretsmanager_secret" "genai_defaults_secret" {
  name                    = local.genai_defaults_secret_name
  description             = "holds default values for smart and fast LLM models"
  recovery_window_in_days = 0
  tags                    = local.tags
}

resource "aws_secretsmanager_secret_version" "genai_defaults_secret_version" {
  secret_id     = aws_secretsmanager_secret.genai_defaults_secret.id
  secret_string = local.genai_defaults_secret_json
}
/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

resource "aws_secretsmanager_secret" "private_model_config_secret" {
  name                    = "genai_private_model/${var.ClusterGUID}/model/${var.Model}"
  kms_key_id              = var.KmsKeyId
  description             = ""
  recovery_window_in_days = 0
  tags                    = local.tags
}

resource "aws_secretsmanager_secret_version" "private_model_config_secret_version" {
  secret_id = aws_secretsmanager_secret.private_model_config_secret.id
  secret_string = templatefile("${path.module}/templates/model-secret-template.tftpl", {
    "Model"         = var.Model
    "ModelEndpoint" = var.ModelEndpoint
    "ModelProvider" = var.ModelProvider
    "ApiKey"        = var.APIKey
    "Active"        = var.Active
  })
}
/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

import com.pega.gradle.plugin.model.DeploymentRefConfig
import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
}

group = "com.pega.provisioning.services"
description = "GenAI Private Model SCE"

dependencies {
    assets(project(":distribution:genai-private-model-config-terraform", "archives"))
}

sar {
    name = "GenAIPrivateModelConfig"
    description = "GenAI Private Model Configuration"

    dynamicParam("ClusterGUID", "GUID of CloudK cluster where service is deployed")

    dynamicParam("Model", "The kind of model that is being configured")
    dynamicParam("ModelProvider", "The model provider")
    dynamicParam("VersionCurrent", "The general supported model version (Ex: 1106, 002, 20240210)")
    dynamicParam("VersionDeprecated", "The model version being deprecated, end-of-life or being replaced (Ex: 1106, 002, 20240210)")
    dynamicParam("VersionNext", "The model version being rolled out as new general available version supported (Ex: 1106, 002, 20240210)")
    dynamicParam("ModelEndpoint", "The URL endpoint that is accessible to GenAI Gateway Service for model inference calls")
    dynamicParam("APIKey", "The API Key that grants access to the Generative Model Endpoint", true) // the true flag here is for encrypting the parameter value.
    dynamicParam("Active", "The Model mapping is active")
    dynamicParam("Owner", "Resource owner for labelling purpose")

    // KMSKeyForSecrets is planned to be decommissioned from Cluster product.
    // This current KMS is a single for Account, and Infrastructure team will implement KMS key per cluster solution
    // in the near future. When this is done, it is needed to provide the new Cluster output as an upgradeValue in the
    // SCE definition.
    fixedParam("KmsKeyId", "cmdbService.cluster(dynamicParams.ClusterGUID).get('KMSKeyForSecrets')")
    fixedParam("Region", "cmdbService.cluster(dynamicParams.ClusterGUID).get('Region')")

    output("SecretID", "Secret created with BYOM details", null)

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "terraform"
        template = configurations.assets.get().files.first().name // resource, which contains the packaged helm
        version = "1.8.0" // Terraform version
        timeout = "10m"  // Terraform timeout
        upgradeAllInstances = "true" // Upgrade version of all SCEs in different namespaces than "Default"
    } as groovy.lang.Closure<DeploymentRefConfig>)
}
/*
 * Copyright (c) 2025 Pegasystems Inc.
 * All rights reserved.
 */

import com.pega.gradle.plugin.model.DeploymentRefConfig

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
}

group = "com.pega.provisioning.services"
description = "GenAI GCP Vertex Infrastructure"

dependencies {
    assets(project(":distribution:genai-gcpvertex-infra-terraform", "archives"))
    services(project(":distribution:genai-gcpvertex-host-sce"))
}

sar {
    name = "GenAIGCPVertexInfra"
    description = "GenAI GCP Vertex Infra"

    dynamicParam("OidcIssuer", "OIDC Issuer URL")
    dynamicParam("ModelInferenceRegionOverride", "The specific region to use for Vertex AI inference. If provided, overrides the default region detection.")
    dynamicParam("FunctionTimeoutSeconds", "Timeout in seconds for the Cloud Function execution (default: 600)")

    fixedParam("Owner", "cmdbService.get('Owner')")
    fixedParam("Region", "cmdbService.get('Region')")
    fixedParam("GcpProjectId", "cmdbService.get('GcpProjectId')")
    fixedParam("GcpVertexAIApiGatewayHost", "cmdbService.get('GcpVertexAIApiGatewayHost')")
    fixedParam("GcpVertexAIServiceName", "cmdbService.get('GcpVertexAIServiceName')")
    fixedParam("ResourcesSuffixId", "cmdbService.get('ResourcesSuffixId')")

    output("GcpVertexAIApiGatewayEndpoint","", null)
    output("ApiGatewayId","", null)
    output("ServiceAccountApiGatewayEmail","", null)
    output("CloudRunFunctionId","", null)
    output("CloudStorageBucketCloudRunSource" ,"", null)
    output("ServiceAccountCloudRunInvokerEmail","", null)

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "terraform"
        template = configurations.assets.get().files.first().name // resource, which contains the packaged helm
        version = "1.8.0" // Terraform version
        timeout = "10m"  // Terraform timeout
        upgradeAllInstances = "true" // Upgrade version of all SCEs in different namespaces than "Default"
    } as groovy.lang.Closure<DeploymentRefConfig>)
}

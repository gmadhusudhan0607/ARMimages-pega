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
description = "GenAI GCP Vertex Host Provisioning"

dependencies {
    assets(project(":distribution:genai-gcpvertex-host-terraform", "archives"))
}

sar {
    name = "GenAIGCPVertexHost"
    description = "GenAI GCP Vertex Host Provisioning"

    dynamicParam("Owner", "Resource owner for labelling purpose")
    dynamicParam("Region", "GCP Region where models will be deployed")
    dynamicParam("AccountID", "GCP Project ID")

    fixedParam("GcpProjectId", "dynamicParams.AccountID")

    output("GcpVertexAIApiGatewayHost","", null)
    output("GcpVertexAIServiceName","", null)
    output("ResourcesSuffixId","", null)
    output("Owner", "", null)
    output("Region", "", null)
    output("GcpProjectId","", null)

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "terraform"
        template = configurations.assets.get().files.first().name // resource, which contains the packaged helm
        version = "1.8.0" // Terraform version
        timeout = "10m"  // Terraform timeout
        upgradeAllInstances = "true" // Upgrade version of all SCEs in different namespaces than "Default"
    } as groovy.lang.Closure<DeploymentRefConfig>)
}

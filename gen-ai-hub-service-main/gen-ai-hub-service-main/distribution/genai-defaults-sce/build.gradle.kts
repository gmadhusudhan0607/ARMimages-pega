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
description = "GenAI Defaults SCE"

dependencies {
    assets(project(":distribution:genai-defaults-terraform", "archives"))
    //services(project(":distribution:genai-awsbedrock-infra-sce"))
    services(project(":distribution:sax-iam-oidc-provider-sce"))
}

sar {
    name = "GenAIDefaults"
    description = "GenAI Defaults SCE containing default fast and smart models per GenAI LLM Account"

    dynamicParam("Owner", "Resource owner for labelling purpose")
    dynamicParam("Region" , "Region where the LLM Models are deployed")
    dynamicParam("AccountID", "Account ID where LLM Models are deployed")
    dynamicParam("Fast", "Default Fast LLM Model")
    dynamicParam("Smart", "Default Smart LLM Model")
    dynamicParam("Pro", "Default Pro LLM Model")

    fixedParam("SaxCell", "cmdbService.get('SaxCell')")
    fixedParam("StageName", "cp.getStageName()")

    output("GenAIDefaultModelSecretName", "", null)

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "terraform"
        template = configurations.assets.get().files.first().name // resource, which contains the packaged helm
        version = "1.8.0" // Terraform version
        timeout = "10m"  // Terraform timeout
        upgradeAllInstances = "true" // Upgrade version of all SCEs in different namespaces than "Default"
    } as groovy.lang.Closure<DeploymentRefConfig>)
}
/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

import com.pega.gradle.plugin.model.DeploymentRefConfig

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
}

group = "com.pega.provisioning.services"
description = "SAX IAM OIDC Provider"

dependencies {
    assets(project(":distribution:sax-iam-oidc-provider-terraform", "archives"))
}

sar {
    name = "SaxIamOidcProvider"
    description = "SAX IAM OIDC Provider"

    dynamicParam("Owner", "Resource owner for labelling purpose")
    dynamicParam("AccountID", "GenAI Account for configuring OIDC Provider")
    dynamicParam("Region", "AWS Region for labelling purpose")
    dynamicParam("SaxCell", "SAX Cell for OIDC calls - US, EU or APAC")

    fixedParam("StageName", "cp.getStageName()")

    output("OidcProviderArn", "ARN of IAM OIDC Provider for the SAX Cell", null)
    output("OidcProviderUrl", "The IAM OIDC Provider URL to be used in Assume Role With OIDC calls", null)
    output("SaxStage", "The SAX Stage used for this OIDC provider", null)
    output("SaxCell", "The SAX Cell in the Stage", null)

    output("GetBedrockModelMappingOidcRole", "The OIDC assumable role that allows List, Read access to Model Mappings in AWS Secrets Manager", null)

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "terraform"
        template = configurations.assets.get().files.first().name // resource, which contains the packaged helm
        version = "1.8.0" // Terraform version
        timeout = "10m"  // Terraform timeout
        upgradeAllInstances = "true" // Upgrade version of all SCEs in different namespaces than "Default"
    } as groovy.lang.Closure<DeploymentRefConfig>)
}

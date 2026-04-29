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
description = "GenAI AWS Bedrock Infrastructre"

dependencies {
    assets(project(":distribution:genai-awsbedrock-infra-terraform", "archives"))
    services(project(":distribution:sax-iam-oidc-provider-sce"))
}

sar {
    name = "GenAIAWSBedrockInfra"
    description = "GenAI AWS Bedrock Infra"

    dynamicParam("Owner", "Resource owner for labelling purpose")
    dynamicParam("AccountID", "")
    dynamicParam("Region", "AWS Region where mappings will be stored")
    dynamicParam("ModelMapping", "Which endpoint of the GenAI Gateway must provide this Model. Ex: gpt-35-turbo, llama3-8b-instruct, ...")
    dynamicParam("ModelID", "ID of the AWS Bedrock Model")
    dynamicParam("TargetApi", "API of the AWS Bedrock Model")
    dynamicParam("UseRegionalInferenceProfile", "Use regional inference profile for this model calls")
    dynamicParam("InferenceProfilePrefix", "CRIS prefix to provision Bedrock resources with Cross Regional Inference Profile")
    dynamicParam("Inactive", "The model mapping is marked as inactive, it will not be used by the GenAI Gateway.")
    dynamicParam("InferenceRegion", "AWS Region for the model endpoint to be invoked. If not give, it defaults to Region.")
    // To fetch from namespace 'default'.
    // If the search on default namespace continue failing, will try to use cmdbService.collectOutputAsString() which is
    // supposed to look for it in all cmdb node namespaces.
    fixedParam("OidcProviderUrl", "cmdbService.get('OidcProviderUrl')")
    fixedParam("SaxCell", "cmdbService.get('SaxCell')")
    fixedParam("StageName", "cp.getStageName()")


    output("BedrockModelAwsSecretName", "", null)

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "terraform"
        template = configurations.assets.get().files.first().name // resource, which contains the packaged helm
        version = "1.8.0" // Terraform version
        timeout = "10m"  // Terraform timeout
        upgradeAllInstances = "true" // Upgrade version of all SCEs in different namespaces than "Default"
    } as groovy.lang.Closure<DeploymentRefConfig>)
}

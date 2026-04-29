/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */

import com.pega.gradle.plugin.model.PluginConfig

group = "com.pega.provisioning.services"
plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
    id("com.pega.controlplane-deployment")
}

val pegasecProviderVersion: String by rootProject.extra
val terraformVersion: String by project
val terraformOutput by configurations.creating

dependencies {
    terraformOutput(project(mapOf(
        "path" to ":distribution:sax-registration-terraform",
        "configuration" to "archives")))
}

description =  "Used to register GenAIVectorStore Service Side in SAX"

val pluginsSCE = listOf(PluginConfig().also {
    it.name = "pegasec"
    it.version = pegasecProviderVersion
})

sar {
    name = "GenAIVectorStoreSaxRegistration"
    description = "Used to register GenAIVectorStore Service Side in SAX (Service and OpsService)"

    dynamicParam("Owner", "Owner of the deployment (for tagging purpose)")
    dynamicParam("Region", "Specifies region in which clients resides")

    output("SaxIssuer", "Identity of OAuth server that creates tokens.", null)
    output("SaxJWKSEndpoint", "Endpoint to obtain JWKS object for key signature validation.", null)
    output("SaxAudience", "Audience that's part of created tokens.", null)
    output("SaxScopesString", "Full names of created scopes, separated with space.", null)

    output("SaxOpsIssuer", "Identity of OAuth server that creates tokens.", null)
    output("SaxOpsJWKSEndpoint", "Endpoint to obtain JWKS object for key signature validation.", null)
    output("SaxOpsAudience", "Audience that's part of created tokens.", null)
    output("SaxOpsScopesString", "Full names of created scopes, separated with space.", null)

    deploymentRef("terraform",
            terraformOutput.files.first().name,
            terraformVersion.substringBeforeLast("."),
            "60m",  //TF timeout
            pluginsSCE
    )
}

tasks.archiveRegistryEntry {
    from(terraformOutput) { into("assets") }
}
/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */


import com.pega.gradle.plugin.model.PluginConfig
import com.pega.gradle.plugin.model.DeploymentRefConfig
import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
}

group = "com.pega.provisioning.services"
description = "Create genai-vector-store isolation"

evaluationDependsOn(":distribution:service-docker")

val psRestApiProviderName: String by project
val psRestApiProviderVersion: String by project
val infinityIsolationVersion: String by project
val terraformVersion: String by project
val autopilotServiceVersion: String by project
val PDCProvisioningServiceVersion: String by project

dependencies {
    assets(project(":distribution:isolation-terraform", "archives"))
    optionalServices(group="com.pega.provisioning.services", name="InfinityIsolation", version=infinityIsolationVersion)
    optionalServices(group="com.pega.provisioning.services", name="autopilot-service", version=autopilotServiceVersion)
    optionalServices(group="com.pega.provisioning.services", name="PDCProvisioning", version=PDCProvisioningServiceVersion)
}

val pluginsSCE = listOf(PluginConfig().also {
    it.name = psRestApiProviderName
    it.version = psRestApiProviderVersion
})

sar {
    name = "GenAIVectorStoreIsolation"
    description = "GenAI Vector Store Isolation"

    dynamicParam("ClusterGUID", "EKS Cluster GUID")
    dynamicParam("MaxStorageSize", "Isolation allocated storage")
    dynamicParam("OpsServiceEndpoint", "GenAI Vector Store Ops Service Endpoint")

    fixedParam("Region", "cmdbService.cluster(dynamicParams.ClusterGUID ?: cmdbService.find('ClusterGUID')).get('Region')")
    fixedParam("AccountID", "cmdbService.cluster(dynamicParams.ClusterGUID ?: cmdbService.find('ClusterGUID')).get('AccountID')")
    fixedParam("Isolation","(cmdbService.find('IsolationID', cmdbService.serviceNameFilter('InfinityIsolation'))) ?: cmdbService.get('IsolationID')")
    fixedParam("DeploymentMode", "cmdbService.getDeploymentMode()")
    fixedParam("PDCEndpointURL", """
    |({ ->
    |   String pdcURL = cmdbService.find('PDCEndpointURL', cmdbService.serviceNameFilter('PDCProvisioning'))?.trim()
    |   if (pdcURL) {
    |     return pdcURL
    |   } else {
    |       return ''
    |   }
    |})()""".trimMargin())

    // Print values returned by Ops Service
    output("IsolationID", "Isolation ID", null)
    output("MaxStorageSize", "Isolation Max Storage Size", null)
    output("VSPDCEndpointURL", "PDC Endpoint URL", null)

    @Suppress("UNCHECKED_CAST")
    deploymentRef(closureOf<DeploymentRefConfig> {
        name = "terraform"
        template = configurations.assets.get().files.first().name
        version = terraformVersion.substringBeforeLast(".")
        timeout = "5m"
        plugins = pluginsSCE
        skipUpdateDuringMRDRFailover = false
    } as groovy.lang.Closure<DeploymentRefConfig>)
}

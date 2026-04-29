/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */


import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
}

group = "com.pega.provisioning.services"
description = "Deployment template to provision genai-vector-store on k8s cluster"

val serviceAuthenticationClientServiceVersion: String by project
val dbInstanceVersion: String by project
val terraformVersion: String by project

dependencies {
    assets(project(":distribution:role-terraform", "archives"))
    services(group="com.pega.provisioning.services", name="DBInstance", version=dbInstanceVersion)
    services(group="com.pega.provisioning.services",name="ServiceAuthenticationClientService", version=serviceAuthenticationClientServiceVersion)
}

sar {
    name = "GenAIVectorStoreRole"
    description = "GenAI Vector Store Role"

    dynamicParam("ClusterGUID", "GUID of CloudK cluster where service is deployed")
    dynamicParam("Namespace", "K8s namespace where service is deployed")

    fixedParam("AccountID", "cmdbService.cluster(dynamicParams.ClusterGUID).get('AccountID')")
    fixedParam("Region", "cmdbService.cluster(dynamicParams.ClusterGUID).get('Region')")
    fixedParam("ActiveRegion", "cmdbService.isDeploymentActive() ? cmdbService.cluster(dynamicParams.ClusterGUID).get('Region') : cmdbService.activeResource().labels().find('region')")
    fixedParam("ActiveKMSKeyForSecrets", "cmdbService.isDeploymentActive() ? cmdbService.cluster(dynamicParams.ClusterGUID).get('KMSKeyForSecrets') : cmdbService.activeResource().get('ActiveKMSKeyForSecrets')", "String", "aws")
    fixedParam("ResourceGUID", "cmdbService.getResourceGUID()")
    fixedParam("DatabaseID", "cmdbService.get('DatabaseID')")
    fixedParam("DatabaseSecret", "cmdbService.get('DatabaseSecret')")
    //Active Database Secret refers to Master Secret for Database in Active Region used fpr MRDR purposes
    fixedParam("ActiveDatabaseSecret", "(cmdbService.isDeploymentActive() ? cmdbService.get('DatabaseSecret') : (cmdbService.activeResource().get('DatabaseSecret')))")
    fixedParam("SaxClientSecret", "cmdbService.get('ServAuthSecretARN')")
    fixedParam(
        "KMSKeyForSecrets",
        "cmdbService.cluster(dynamicParams.ClusterGUID).get('KMSKeyForSecrets')",
        "String",
        "aws"
    )
    //For MRDR purposes we are setting the value to 'NotConfigured' in case when don't have the parameter value yet on BACKUP
    fixedParam(
        "ClusterOIDCIssuerURL",
        "cmdbService.cluster(dynamicParams.ClusterGUID).get('ClusterOIDCIssuerURL') ?: 'NotConfigured'",
        "String",
        "aws"
    )

    output("ServiceAccountRole", "IAM role for GenAI Vector Store k8s service account", null)
    output("ActiveDatabaseSecret", "Master Database Secret in Active Region for MRDR", "(cmdbService.isDeploymentActive() ? cmdbService.get('DatabaseSecret') : (cmdbService.activeResource().get('DatabaseSecret')))")
    output("ActiveKMSKeyForSecrets", "KMSKeyForSecrets in Active Region for MRDR", "cmdbService.isDeploymentActive() ? cmdbService.cluster(dynamicParams.ClusterGUID).find('KMSKeyForSecrets') : cmdbService.activeResource().get('ActiveKMSKeyForSecrets')")

    deploymentRef("terraform",
            configurations.assets.get().files.first().name, // resource, which contains the packaged helm
            terraformVersion.substringBeforeLast("."), // Terraform version
            "10m"  // Terraform timeout
    )
}

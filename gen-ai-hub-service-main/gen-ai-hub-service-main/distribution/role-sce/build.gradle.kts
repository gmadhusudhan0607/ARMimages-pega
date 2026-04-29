/*
 * Copyright (c) 2024 Pegasystems Inc.
 * All rights reserved.
 */


import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
}

group = "com.pega.provisioning.services"
description = "Deployment template to provision gen-ai-hub-service on k8s cluster"

val serviceAuthenticationClientServiceVersion: String by project

dependencies {
    assets(project(":distribution:role-terraform", "archives"))
    services(group="com.pega.provisioning.services",name="ServiceAuthenticationClientService", version=serviceAuthenticationClientServiceVersion)
}

sar {
    name = "GenAIHubServiceRole"
    description = "GenAI Hub Service Role"

    dynamicParam("ClusterGUID", "GUID of CloudK cluster where service is deployed")

    fixedParam("AccountID", "cmdbService.cluster(dynamicParams.ClusterGUID).get('AccountID')")
    fixedParam("Region", "cmdbService.cluster(dynamicParams.ClusterGUID).get('Region')")
    fixedParam("ResourceGUID", "cmdbService.getResourceGUID()")
    fixedParam("SaxClientSecret", "cmdbService.get('ServAuthSecretARN')")
    fixedParam("KMSKeyForSecrets", "cmdbService.cluster(dynamicParams.ClusterGUID).get('KMSKeyForSecrets')")
    fixedParam("ClusterOIDCIssuerURL","cmdbService.cluster(dynamicParams.ClusterGUID).get('ClusterOIDCIssuerURL')")

    output("ServiceAccountRole", "IAM role for GenAI Hub Service k8s service account", null)

    deploymentRef("terraform",
            configurations.assets.get().files.first().name, // resource, which contains the packaged helm
            "1.8.0", // Terraform version
            "10m"  // Terraform timeout
    )
}

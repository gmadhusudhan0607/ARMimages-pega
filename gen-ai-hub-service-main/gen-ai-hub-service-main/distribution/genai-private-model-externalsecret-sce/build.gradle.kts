import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

import com.pega.gradle.plugin.tasks.RetrieveServiceOutputTask
import com.google.gson.Gson
import com.google.gson.reflect.TypeToken
import java.util.UUID

group = "com.pega.provisioning.services"
description = "Deployment template to provision genai-hub-service in k8s cluster"

plugins {
    id("com.pega.sce.plugin")
    id("com.pega.sce.publishing")
    id("com.pega.helmcli.helm")
}

// List dependencies. Can have assets, services, optionalServices, testCompile
dependencies {
    assets(project(":distribution:genai-private-model-externalsecret-helm", "archives"))
}

val serviceNamespace: String by project
val helmVersion: String by project

sar {
    name = "GenAIPrivateModelExternalSecret"
    description = "GenAI Private Model External Secret"

    dynamicParam("ClusterGUID", "GenAI Gateway Service Product that will have Private Models enabled")
    dynamicParam("GatewayGUID", "GenAI Gateway Service Product that will have Private Models enabled")

    fixedParam("kubeconfig", "cmdbService.cluster(dynamicParams.ClusterGUID).get('kubeconfig')")
    fixedParam("AccountID","cmdbService.labels().get('accountid')")
    fixedParam("namespace","cmdbService.backingService(dynamicParams.GatewayGUID).get('ServiceNamespace')")
    fixedParam("StageName", "cp.getStageName()")
    fixedParam("SaxCell", "cmdbService.backingService(dynamicParams.GatewayGUID).get('SaxCell')")

    deploymentRef("helm",
        configurations.assets.get().files.first().name, // resource, which contains the packaged helm
        "3.18", //Helm version
        "10m"  //Helm timeout
    )
}

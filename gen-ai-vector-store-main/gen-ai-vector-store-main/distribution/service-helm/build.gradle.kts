/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    id("com.pega.helmcli.helm") //This plugin packages, publishes, installs, upgrades, and deletes helm charts.
}

val serviceNamespace: String by project

//make sure the docker project is evaluated first so we can get at its data
evaluationDependsOn(":distribution:service-docker")
evaluationDependsOn(":distribution:ops-docker")
//you want to get this directly from the docker sub-project.  If you try to get it
//from the dockerRepo gradle property you will only pick up the value in
//the root gradle.properties file, which *could* be overridden in the
//docker sub-project's gradle.properties
val dkBuildImage = project(":distribution:service-docker").tasks.named<DockerImageBuild>("buildImage")
val opsDkBuildImage = project(":distribution:ops-docker").tasks.named<DockerImageBuild>("buildImage")
val bkgDkBuildImage = project(":distribution:background-docker").tasks.named<DockerImageBuild>("buildImage")

//Use templated values when packaging
helm {
    charts {
        named("main") {
            filtering {
                values.putAll(project.provider {
                    mapOf(
                        "bkgImageName" to bkgDkBuildImage.get().image.name.get(),
                        "opsImageName" to opsDkBuildImage.get().image.name.get(),
                        "imageName" to dkBuildImage.get().image.name.get(),
                        "imageTag" to dkBuildImage.get().image.tag.get(),
                        "namespace" to serviceNamespace
                    )
                })
            }
        }
    }
    releases {
        named("main") {
            if (buildInfo.isLocalBuild) {
                //customized overrides when running locally.  these allow you to launch on a local minikube
                valueFiles.from("local-deploy-values.yaml")
            }
        }
    }
}

project.ext["helmChartName"] = "genai-vector-store"
project.ext["serviceName"] = "genai-vector-store"
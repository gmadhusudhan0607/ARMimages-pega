import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    id("com.pega.helmcli.helm") // This plugin packages, publishes, installs, upgrades, and deletes helm charts.
}

val serviceNamespace: String by project

// Make sure that the docker project is evaluated first, so it's possible to retrieve data
evaluationDependsOn(":distribution:genai-hub-service-docker")

// Retrieve docker image name and tag directly from the docker sub-project.
// Retrieving from dockerRepo gradle property picks up the value from the root gradle.properties file,
// which *could* be overridden in the docker sub-project's gradle.properties
val dockerBuildImage = project(":distribution:genai-hub-service-docker").tasks.named<DockerImageBuild>("buildImage")
val dockerBuildOpsImage = project(":distribution:genai-gateway-ops-docker").tasks.named<DockerImageBuild>("buildImage")

//Use templated values when packaging
helm {
    charts {
        named("main") {
            filtering {
                values.putAll(project.provider {
                    mapOf(
                        "imageName" to dockerBuildImage.get().image.name.get(),
                        "imageTag" to dockerBuildImage.get().image.tag.get(),
                        "opsImageName" to dockerBuildOpsImage.get().image.name.get(),
                        "opsImageTag" to dockerBuildOpsImage.get().image.tag.get(),
                        "namespace" to serviceNamespace
                    )
                })
            }
        }
    }
    
}

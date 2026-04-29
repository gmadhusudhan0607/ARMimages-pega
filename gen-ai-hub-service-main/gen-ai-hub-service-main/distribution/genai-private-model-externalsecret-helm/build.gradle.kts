import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    id("com.pega.helmcli.helm") // This plugin packages, publishes, installs, upgrades, and deletes helm charts.
}

val serviceNamespace: String by project

//Use templated values when packaging
helm {
    charts {
        named("main") {
            filtering {
                values.putAll(project.provider {
                    mapOf(
                        "namespace" to serviceNamespace
                    )
                })
            }
        }
    }
    
}

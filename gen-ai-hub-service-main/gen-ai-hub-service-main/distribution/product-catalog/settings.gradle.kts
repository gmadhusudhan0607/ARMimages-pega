pluginManagement {
    val artifactoryURL: String by settings
    val artifactoryUser: String by settings
    val artifactoryPassword: String by settings
    repositories {
        mavenLocal()
        maven {
            setUrl("${artifactoryURL}/repo2")
            credentials {
                setUsername("${artifactoryUser}")
                setPassword("${artifactoryPassword}")
            }
            metadataSources {
                ignoreGradleMetadataRedirection()
                mavenPom()
            }
        }
    }
    resolutionStrategy {
        val scePluginVersion: String by settings
        eachPlugin {
            if (requested.id.id == "com.pega.sce.plugin") {
                useModule("com.pega.gradle.plugins:pega-sce-gradle-plugin:${requested.version ?: "${scePluginVersion}"}")
            }
        }
    }
    plugins {
        val sdeaOpinionatedPluginsVersion: String by settings
        val cloudServicesPluginsVersion: String by settings
        //plugins to *actually* apply to the root project

        //SQuID Opinionated Plugins
        //All versions must be the same
        id("com.pega.release.cibuildversion") version "${sdeaOpinionatedPluginsVersion}"
        id("com.pega.java") version "${sdeaOpinionatedPluginsVersion}"
        id("com.pega.quality") version "${sdeaOpinionatedPluginsVersion}"
        id("com.pega.build-cache") version "${sdeaOpinionatedPluginsVersion}"
        id("com.pega.sce.publishing") version "${sdeaOpinionatedPluginsVersion}"
        id("com.pega.cloud.services.changelog") version "${cloudServicesPluginsVersion}"
        //End SQuID Opinionated Plugins
    }
}

plugins {
    id("com.pega.build-cache")
}

rootProject.name = "genai-gateway-service-product"


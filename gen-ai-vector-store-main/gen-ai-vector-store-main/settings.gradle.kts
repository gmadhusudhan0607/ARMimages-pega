/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */
 
pluginManagement {
    val artifactoryURL: String by settings
    val artifactoryUser: String by settings
    val artifactoryPassword: String by settings
    val cloudServicesPluginsVersion: String by settings
    repositories {
        mavenLocal()
        maven {
            setUrl("${artifactoryURL}/repo2")
            credentials {
                username = artifactoryUser
                password = artifactoryPassword
            }
            metadataSources {
                ignoreGradleMetadataRedirection()
                mavenPom()
            }
        }
    }

    plugins {
        //START - SDEA Opinionated plugins, all use the same version
        val sdeaOpinionatedPluginsVersion: String by settings
        id("com.pega.quality") version sdeaOpinionatedPluginsVersion
        id("com.pega.release.smartversion") version sdeaOpinionatedPluginsVersion
//        id("com.pega.release.veracode") version sdeaOpinionatedPluginsVersion
        id("com.pega.controlplane-deployment") version sdeaOpinionatedPluginsVersion
        id("com.pega.dockercli.image") version sdeaOpinionatedPluginsVersion
        id("com.pega.helmcli.helm") version sdeaOpinionatedPluginsVersion
        id("com.pega.terraform") version sdeaOpinionatedPluginsVersion
        id("com.pega.sce.publishing") version sdeaOpinionatedPluginsVersion
        id("com.pega.build-cache") version sdeaOpinionatedPluginsVersion
        //https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse/projects/go-plugin
        id("com.pega.go") version sdeaOpinionatedPluginsVersion
        // "com.pega.performance" temporarily disabled due to issues with performance tests
        //id("com.pega.performance") version sdeaOpinionatedPluginsVersion
        //https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse/projects/security-base-plugin
        id("com.pega.securitybase") version sdeaOpinionatedPluginsVersion
        //https://git.pega.io/projects/PP/repos/gradle-prpc-platform-plugins/browse/projects/contract-test-runner-plugin
        id("com.pega.contract") version sdeaOpinionatedPluginsVersion
        //END - SDEA opinionated plugins

        //Cloud team's SCE plugin
        val scePluginVersion: String by settings
        // https://git.pega.io/projects/CTRL/repos/pega-sce-gradle-plugin/browse/RELEASE_NOTES.md
        id("com.pega.sce.plugin") version scePluginVersion

        //START - cloud-services-plugins
        // https://git.pega.io/projects/PCLD/repos/gradle-cloud-services-plugins/browse
        id("com.pega.cloud.services.changelog") version cloudServicesPluginsVersion
        //END - cloud-services-plugins

        //plugins for microBenchmark Testing
        id("io.morethan.jmhreport") version "0.9.0"
    }
}

plugins {
   id("com.pega.build-cache")
}

rootProject.name = "genai-vector-store"

//sub-projects to include
include("distribution:sax-registration-sce")
include("distribution:sax-registration-terraform")
include("distribution:service-infrastructure-terraform")
include("distribution:service-infrastructure-sce")
include("distribution:isolation-terraform")
include("distribution:isolation-sce")
include("distribution:role-terraform")
include("distribution:role-sce")

include("distribution:ops-go")
include("distribution:ops-docker")

include("distribution:background-go")
include("distribution:background-docker")

include("distribution:service-go")
include("distribution:service-docker")

include("distribution:service-helm")
include("distribution:service-sce")

include("distribution:productcatalog")

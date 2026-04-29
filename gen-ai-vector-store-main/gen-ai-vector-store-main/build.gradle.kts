/*
 * Copyright (c) 2023 Pegasystems Inc.
 * All rights reserved.
 */

import com.pega.gradle.plugins.dockercli.tasks.image.DockerImageBuild

plugins {
    //sonar
    id("com.pega.quality") //SDEA plugin that handles versioning
    id("com.pega.release.smartversion")
    id("com.pega.securitybase")
    //id("com.pega.performance") temporary disabld
    id("com.pega.cloud.services.changelog")

    //Add plugin but do not APPLY.  This line here is to ensure there are no
    //ClassCastExceptions when referencing tasks from the distribution/docker
    //sub-project from other sub-projects (like service & distribution/helm)
    id("com.pega.dockercli.image") apply false
}


changelog {
    serviceOwnerEmail.set("JarvisScrumTeam@pega.com")
    defaultBranch.set("main")
}

// Define root project information here
description = "genai-vector-store"

/*
task("deployStaging") { doLast { readCredentials("staging") } }
task("deployProduction") { doLast { readCredentials("production") } }
task("deployIntegration") { doLast { readCredentials("integration") } }

fun readCredentials(identifier: String) {
    val homeDir = System.getProperty("user.home")
    fun readFile(fileName: String): List<String> = File(fileName).bufferedReader().readLines()
    println("deploying " + identifier + " with creds ${readFile("${homeDir}/.aws/credentials")}")
}
 */


security {
    enabled.set(true) //true by default
    includes.set(setOf("**/*.jar","**/*.zip"))
    excludes.set(setOf("**/*-docs.jar","**/*-sources.zip", "**/package.json"))
}

// Define bitbucketUser and bitbucketPassword in  ~/.gradle/gradle.properties
// You can use  HTTP token instead of Password,
// HTTP token can be generated on git.pega.com (Under "Manage Account" -> "HTTP access tokens")
val bitbucketUser: String by project
val bitbucketPassword: String by project

var bitbucketUsr: String = System.getenv("BITBUCKET_USR") ?: bitbucketUser
var bitbucketPwd: String = System.getenv("BITBUCKET_PSW") ?: bitbucketPassword

val githubUser: String by project
val githubPassword: String by project

var githubUsr: String = System.getenv("GITHUB_USR") ?: githubUser
var githubPwd: String = System.getenv("GITHUB_PSW") ?: githubPassword

var userHomeDir = System.getenv("HOME") ?: "."
var isCI = System.getenv("CI") ?: "false"

//
//performance {
//    val config = mutableMapOf(
//        "One-User" to mutableMapOf(
//            "Test" to listOf("default-config.yml", "one-user-test.yml")
//        ),
//        "Baseline" to mutableMapOf(
//            "Warmup" to listOf("default-config.yml", "baseline-warmup.yml"),
//            "Test" to listOf("default-config.yml", "baseline-test.yml")
//        ),
//        "Scale" to mutableMapOf(
//            "Warmup" to listOf("default-config.yml", "scale-warmup.yml"),
//            "Test" to listOf("default-config.yml", "scale-test.yml")
//        ),
//        "Endurance" to mutableMapOf(
//            "Test" to listOf("default-config.yml", "endurance-test.yml")
//        )
//    )
//
//    println("GenAIVectorStore::performance project: ${project}")
//
//    if (project.hasProperty("localTaurusConfigFile")) {
//        val localConfig = listOf("default-config.yml", project.property("localTaurusConfigFile").toString())
//        config["Baseline"]!!["Test"] = localConfig
//        config["One-User"]!!["Test"] = localConfig
//    }
//    println("GenAIVectorStore::performance config: ${config}")
//
//    testsConfig.set(config)
//    resiliencyTestsEnabled.set(listOf("Scale"))
//
//
//    properties.set(project.provider {
//        mapOf(
//            "scenarios.jmeter.properties.serviceURL" to "${project.property("performanceServiceEndpointURL")}",
//            "scenarios.jmeter.properties.isolationId" to "${project.property("performanceIsolationId")}",
//            "scenarios.jmeter.properties.collectionId" to "${project.property("performanceCollectionId")}",
//            "scenarios.jmeter.properties.tokenhack" to "${project.property("performanceTokenhack")}",
//            "scenarios.jmeter.properties.numberOfTestDocuments" to "${project.property("performanceNumberOfTestDocuments")}"
//            // "scenarios.jmeter.properties.uasURL" to "${project.property("performanceUASTokenURL")}",
//            // "scenarios.jmeter.properties.clientId" to "${aws.secrets["performanceAuth"].values["clientId"]}",
//            // "scenarios.jmeter.properties.clientSecret" to "${aws.secrets["performanceAuth"].values["clientSecret"]}",
//        )
//    })
//    println("GenAIVectorStore::performance properties: ${properties}")
//}

// Configure GIT to be able to get pkgs from gi.pega.io
tasks.register("configurePegaGitAccess") {
    dependsOn("configurePegaBitbucketGitAccess")
    dependsOn("configurePegaGithubGitAccess")
}

tasks.register("configurePegaBitbucketGitAccess") {
    val credLine = "machine git.pega.io login $bitbucketUsr password $bitbucketPwd"
    val netrcFile = file("$userHomeDir/.netrc")
    if (! netrcFile.exists()) {
        netrcFile.appendText("$credLine\n")
    } else {
        if (!netrcFile.readLines().contains(credLine)) {
            netrcFile.appendText("$credLine\n")
        }
    }
}

tasks.register("configurePegaGithubGitAccess") {
    val credLine = "machine github.com login $githubUsr password $githubPwd"
    val netrcFile = file("$userHomeDir/.netrc")
    if (! netrcFile.exists()) {
        netrcFile.appendText("$credLine\n")
    } else {
        if (!netrcFile.readLines().contains(credLine)) {
            netrcFile.appendText("$credLine\n")
        }
    }
}

tasks.register<Exec>("unitTestReport") {
    commandLine("make", "unit-test-report")
}

tasks.register<Exec>("integTestReport") {
    commandLine("make", "integration-test-report")
}

tasks.register("integTest") {
    dependsOn("integrationTest")
}

tasks.register("integrationTest") {
    if (isCI == "true") { dependsOn("configurePegaGitAccess") }
    dependsOn("integrationTestInfrastructureUp")
    dependsOn("integrationTestRun")
    // Commenting out because of BUG-858913
    //finalizedBy("integTestReport")
}


tasks.register<Exec>("integrationTestInfrastructureUp") {
    dependsOn(":distribution:service-docker:buildImage")
    dependsOn(":distribution:ops-docker:buildImage")
    dependsOn(":distribution:background-docker:buildImage")
    val dkBuildImage = project(":distribution:service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkOpsBuildImage = project(":distribution:ops-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBackgroundBuildImage = project(":distribution:background-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("OPS_DOCKER_IMAGE", dkOpsBuildImage.get().image.get())
        environment("BACKGROUND_DOCKER_IMAGE", dkBackgroundBuildImage.get().image.get())
    }
    commandLine("make", "integration-test-up")
}

tasks.register<Exec>("integrationTestInfrastructureDown") {
    mustRunAfter("integrationTestPrintDockerLogs")
    val dkBuildImage = project(":distribution:service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkOpsBuildImage = project(":distribution:ops-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBackgroundBuildImage = project(":distribution:background-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("OPS_DOCKER_IMAGE", dkOpsBuildImage.get().image.get())
        environment("BACKGROUND_DOCKER_IMAGE", dkBackgroundBuildImage.get().image.get())
    }
    commandLine("make", "integration-test-down")
}

tasks.register<Exec>("integrationTestRun") {
    mustRunAfter("integrationTestInfrastructureUp")
    finalizedBy("integrationTestPrintDockerLogs")
    finalizedBy("integrationTestInfrastructureDown")
    val dkBuildImage = project(":distribution:service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkOpsBuildImage = project(":distribution:ops-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBackgroundBuildImage = project(":distribution:background-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("OPS_DOCKER_IMAGE", dkOpsBuildImage.get().image.get())
        environment("BACKGROUND_DOCKER_IMAGE", dkBackgroundBuildImage.get().image.get())
    }
    commandLine("make", "integration-test-run")
}

tasks.register<Exec>("integrationTestPrintDockerLogs") {
    mustRunAfter("integrationTestRun")
    commandLine("make", "integration-test-print-docker-logs")
}

tasks.register("integrationTestWithoutCleanup") {
    if (isCI == "true") { dependsOn("configurePegaGitAccess") }
    dependsOn("integrationTestInfrastructureUp")
    dependsOn("integrationTestRunWithoutCleanup")
}

tasks.register<Exec>("integrationTestRunWithoutCleanup") {
    mustRunAfter("integrationTestInfrastructureUp")
    finalizedBy("integrationTestPrintDockerLogs")
    val dkBuildImage = project(":distribution:service-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkOpsBuildImage = project(":distribution:ops-docker").tasks.named<DockerImageBuild>("buildImage")
    val dkBackgroundBuildImage = project(":distribution:background-docker").tasks.named<DockerImageBuild>("buildImage")
    doFirst {
        environment("SERVICE_DOCKER_IMAGE", dkBuildImage.get().image.get())
        environment("OPS_DOCKER_IMAGE", dkOpsBuildImage.get().image.get())
        environment("BACKGROUND_DOCKER_IMAGE", dkBackgroundBuildImage.get().image.get())
    }
    commandLine("make", "integration-test-run")
}

// =============================================================================
// Pact Consumer Contract Tests (local development only)
// =============================================================================
// 
// gen-ai-vector-store is a CONSUMER of GenAI Gateway (gen-ai-hub-service).
// For CI/CD publishing, see distribution/service-go/build.gradle.kts
//
// Usage:
//   make pact-test              - Run pact tests locally

tasks.register<Exec>("pactTest") {
    description = "Run Pact consumer contract tests to generate contract files"
    group = "verification"
    commandLine("make", "pact-test")
    outputs.dir("internal/embedders/pact/pact")
}

tasks.register<Exec>("pactClean") {
    description = "Clean Pact generated files"
    group = "verification"
    commandLine("make", "pact-clean")
}

